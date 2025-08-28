package server

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"mi0772/podcache/cache"
	"mi0772/podcache/resp"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

const MAX_COMMAND_SIZE = 512 * 1024 * 1024

var (
	ErrMissingKey     = errors.New("missing key")
	ErrMissingValue   = errors.New("missing key or value")
	ErrNotInteger     = errors.New("value is not an integer")
	ErrInvalidCommand = errors.New("invalid command")
)

type PodCacheServer struct {
	port    int
	cache   *cache.PodCache
	running bool
}

func NewPodCacheServer(cache *cache.PodCache) *PodCacheServer {
	return &PodCacheServer{
		cache: cache,
		port:  getPort(),
	}
}

func (s *PodCacheServer) Start(ctx context.Context) error {
	slog.Info("TCP Server", "phase", "starting")

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		slog.Error("TCP Server", "phase", "starting", "err", err)
		log.Fatal("failed to start server")
	}
	defer listener.Close()

	s.running = true

	slog.Info("TCP Server", "phase", "started", "port", s.port)
	// Graceful shutdown
	go func() {
		<-ctx.Done()
		s.running = false
		listener.Close()
	}()

	for s.running {
		conn, err := listener.Accept()
		if err != nil {
			if s.running {
				log.Printf("error accepting connection: %v", err)
			}
			continue
		}
		go s.handleConnection(conn)
	}

	return nil
}

func (s *PodCacheServer) handleConnection(conn net.Conn) {
	defer conn.Close()

	// Set timeouts if it's a TCP connection
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetReadDeadline(time.Now().Add(30 * time.Second))
		tcpConn.SetWriteDeadline(time.Now().Add(30 * time.Second))
	}

	client := &Client{
		conn:   conn,
		reader: bufio.NewReader(conn),
		writer: bufio.NewWriter(conn),
	}

	for {
		command, err := s.readCommand(client)
		if err != nil {
			if !isConnectionClosed(err) {
				log.Printf("error reading command: %v", err)
				client.sendError("Invalid command")
			}
			return
		}

		if err := s.executeCommand(client, command); err != nil {
			if errors.Is(err, errQuit) {
				return
			}
			log.Printf("error executing command: %v", err)
		}
	}
}

var errQuit = errors.New("client quit")

func (s *PodCacheServer) readCommand(client *Client) (*resp.Command, error) {

	reader := bufio.NewReader(client.reader)
	return resp.ParseFromReader(reader)

}

func (s *PodCacheServer) executeCommand(client *Client, cmd *resp.Command) error {
	switch cmd.Type {
	case resp.RESP_PING:
		return s.handlePing(client)
	case resp.RESP_CLIENT:
		return s.handleClient(client, cmd.Arguments)
	case resp.RESP_QUIT:
		client.sendOK("BYE")
		return errQuit
	case resp.RESP_GET:
		return s.handleGet(client, cmd.Arguments)
	case resp.RESP_SET:
		return s.handleSet(client, cmd.Arguments)
	case resp.RESP_INCR, resp.RESP_INCRBY:
		return s.handleIncrement(client, cmd)
	case resp.RESP_DEL, resp.RESP_UNLINK:
		return s.handleDelete(client, cmd.Arguments)
	default:
		return client.sendError("Unknown command")
	}
}

func (s *PodCacheServer) handlePing(client *Client) error {
	return client.sendOK("PONG")
}

func (s *PodCacheServer) handleClient(client *Client, args []string) error {
	if len(args) == 0 {
		return client.sendOK("OK")
	}

	switch strings.ToUpper(args[0]) {
	case "LIST":
		info := fmt.Sprintf("id=1 addr=%s age=0 idle=0 flags=N",
			client.conn.RemoteAddr().String())
		return client.sendBulkString(info)
	case "SETNAME":
		if len(args) < 2 {
			return client.sendError("wrong number of arguments for 'client setname'")
		}
		return client.sendOK("OK")
	case "GETNAME":
		return client.sendBulkString("")
	default:
		return client.sendOK("OK")
	}
}

func (s *PodCacheServer) handleGet(client *Client, args []string) error {
	if len(args) < 1 {
		return client.sendError(ErrMissingKey.Error())
	}

	value, err := s.cache.Get(args[0])
	if err != nil {
		return client.sendError(err.Error())
	}

	if value == nil {
		return client.sendNullBulkString()
	}

	return client.sendBulkString(string(value))
}

func (s *PodCacheServer) handleSet(client *Client, args []string) error {
	if len(args) < 2 {
		return client.sendError(ErrMissingValue.Error())
	}

	if err := s.cache.Put(args[0], []byte(args[1])); err != nil {
		return client.sendError(err.Error())
	}

	return client.sendOK("OK")
}

func (s *PodCacheServer) handleIncrement(client *Client, cmd *resp.Command) error {
	if len(cmd.Arguments) < 1 {
		return client.sendError(ErrMissingKey.Error())
	}

	key := cmd.Arguments[0]
	increment := 1

	if cmd.Type == resp.RESP_INCRBY && len(cmd.Arguments) >= 2 {
		var err error
		increment, err = strconv.Atoi(cmd.Arguments[1])
		if err != nil {
			return client.sendError("increment must be an integer")
		}
	}

	currentValue, err := s.cache.Get(key)
	if err != nil {
		return client.sendError(err.Error())
	}

	var newValue int
	if currentValue == nil {
		newValue = increment
	} else {
		currentInt, err := strconv.Atoi(string(currentValue))
		if err != nil {
			return client.sendError(ErrNotInteger.Error())
		}
		newValue = currentInt + increment
	}

	if err := s.cache.Put(key, []byte(strconv.Itoa(newValue))); err != nil {
		return client.sendError(err.Error())
	}

	return client.sendInteger(newValue)
}

func (s *PodCacheServer) handleDelete(client *Client, args []string) error {
	if len(args) == 0 {
		return client.sendInteger(0)
	}

	deleted := 0
	for _, key := range args {
		if s.cache.Evict(key) {
			deleted++
		}
	}

	return client.sendInteger(deleted)
}

// Client rappresenta una connessione client
type Client struct {
	conn   net.Conn
	reader *bufio.Reader
	writer *bufio.Writer
}

func (c *Client) sendOK(message string) error {
	_, err := c.writer.WriteString(fmt.Sprintf("+%s\r\n", message))
	if err != nil {
		return err
	}
	return c.writer.Flush()
}

func (c *Client) sendError(message string) error {
	_, err := c.writer.WriteString(fmt.Sprintf("-ERR %s\r\n", message))
	if err != nil {
		return err
	}
	return c.writer.Flush()
}

func (c *Client) sendInteger(value int) error {
	_, err := c.writer.WriteString(fmt.Sprintf(":%d\r\n", value))
	if err != nil {
		return err
	}
	return c.writer.Flush()
}

func (c *Client) sendBulkString(value string) error {
	if len(value) == 0 {
		return c.sendNullBulkString()
	}

	_, err := c.writer.WriteString(fmt.Sprintf("$%d\r\n%s\r\n", len(value), value))
	if err != nil {
		return err
	}
	return c.writer.Flush()
}

func (c *Client) sendNullBulkString() error {
	_, err := c.writer.WriteString("$-1\r\n")
	if err != nil {
		return err
	}
	return c.writer.Flush()
}

func getPort() int {
	if portStr, exists := os.LookupEnv("PODCACHE_PORT"); exists {
		slog.Debug("PODCACHE_PORT found in environment: %s\n", portStr)
		port, err := strconv.Atoi(portStr)
		if err != nil {
			slog.Error("PODCACHE_PORT must be a valid number: %v", err)
			os.Exit(1)
		}
		return port
	}
	return 6379
}

func isConnectionClosed(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "connection reset") ||
		strings.Contains(err.Error(), "broken pipe") ||
		strings.Contains(err.Error(), "use of closed network connection"))
}
