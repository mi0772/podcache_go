package server

import (
	"bufio"
	"fmt"
	"log"
	"mi0772/podcache/cache"
	"mi0772/podcache/resp"
	"net"
	"os"
	"strconv"
)

type PodCacheServer struct {
	port   int
	status int
	cache  *cache.PodCache
}

func NewPodCacheServer(cache *cache.PodCache) *PodCacheServer {
	return &PodCacheServer{status: 0, cache: cache}
}

func (s *PodCacheServer) Bootstrap() {
	fmt.Println("Bootstrapping podcache server")

	s.port = getPort()

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()
	s.status = 1
	fmt.Printf("podcache server started on port %d\n", s.port)
	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go s.handleClientConnection(conn)
	}
}

func (s *PodCacheServer) handleClientConnection(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	buffer := make([]byte, 4096)

	for {
		n, err := reader.Read(buffer)
		if err != nil {
			break
		}

		//parse comando resp
		
	}
}

func (s *PodCacheServer) SendIntegerResponse(writer *bufio.Writer, v int) {
	_, _ = writer.WriteString(fmt.Sprintf(":%d\r\n", v))
	_ = writer.Flush()
}

func (s *PodCacheServer) SendOkResponse(writer *bufio.Writer, v string) {
	_, _ = writer.WriteString(fmt.Sprintf("+%s\r\n", v))
	_ = writer.Flush()
}

func (s *PodCacheServer) SendErrorResponse(writer *bufio.Writer, v string) {
	_, _ = writer.WriteString(fmt.Sprintf("-ERR %s\r\n", v))
	_ = writer.Flush()
}

func (s *PodCacheServer) SendBulkStringResponse(writer *bufio.Writer, v string) {
	if len(v) == 0 {
		_, _ = writer.WriteString("$-1\r\n") // NULL bulk string
	} else {
		_, _ = writer.WriteString(fmt.Sprintf("$%d\r\n%s\r\n", len(v), v))
	}
	_ = writer.Flush()
}

func getPort() int {
	if cport, found := os.LookupEnv("PODCACHE_PORT"); found {
		fmt.Printf("PODCACHE_PORT found on osenv : %s\n", cport)
		p, err := strconv.Atoi(cport)
		if err != nil {
			panic("PODCACHE_PORT should be a number")
		}
		return p
	}
	return 6379
}
