package resp

import (
	"errors"
	"strconv"
	"strings"
)

var (
	ErrParseIncomplete = errors.New("parse incomplete")
	ErrParseSyntax     = errors.New("parse syntax")
)

const MIN_BUFFER_SIZE = 4

const (
	RESP_SET    RespCommand = "SET"
	RESP_GET    RespCommand = "GET"
	RESP_DEL    RespCommand = "DEL"
	RESP_PING   RespCommand = "PING"
	RESP_QUIT   RespCommand = "QUIT"
	RESP_CLIENT RespCommand = "CLIENT"
	RESP_UNKNOW RespCommand = "UNKNOW"
	RESP_INCR   RespCommand = "INCR"
	RESP_UNLINK RespCommand = "UNLINK"
)

type RespCommand string

func convert(r string) RespCommand {
	switch r {
	case "SET":
		return RESP_SET
	case "GET":
		return RESP_GET
	case "DEL":
		return RESP_DEL
	case "PING":
		return RESP_PING
	case "QUIT":
		return RESP_QUIT
	case "CLIENT":
		return RESP_CLIENT
	case "INCR":
		return RESP_INCR
	case "UNLINK":
		return RESP_UNLINK
	default:
		return RESP_UNKNOW
	}
}

type Command struct {
	command   RespCommand
	arguments []string
}

type CommandBuffer struct {
	data   []byte
	pos    int
	length int
}

func Parse(command string) (*Command, error) {
	if len(command) < MIN_BUFFER_SIZE {
		return nil, ErrParseIncomplete
	}

	buffer := CommandBuffer{
		data:   []byte(command),
		length: len(command),
	}

	// deve iniziare con '*'
	if buffer.Read(1) != '*' {
		return nil, ErrParseSyntax
	}

	numElements, err := buffer.ReadInteger()
	if err != nil {
		return nil, err
	}

	result := &Command{}
	for i := 0; i < numElements; i++ {
		s, err := buffer.ReadBulkString()
		if err != nil {
			return nil, err
		}
		if i == 0 {
			result.command = convert(strings.ToUpper(s))
		} else {
			result.arguments = append(result.arguments, s)
		}
	}

	return result, nil
}

func (b *CommandBuffer) Read(count int) byte {
	if b.pos+count > b.length {
		return 0
	}
	result := b.data[b.pos]
	b.pos++
	return result
}

func (b *CommandBuffer) FindNextCRLF() int {
	start := b.pos
	end := b.length

	for start < end-1 {
		if b.data[start] == '\r' && b.data[start+1] == '\n' {
			return start
		}
		start++
	}
	return 0
}

func (b *CommandBuffer) ReadBulkString() (string, error) {
	if b.Peek() != '$' {
		return "", ErrParseSyntax
	}
	b.Skip(1)
	strlen, err := b.ReadInteger()
	if err != nil {
		return "", ErrParseSyntax
	}
	if strlen == -1 {
		return "", nil
	}
	v := b.data[b.pos : b.pos+strlen]
	b.pos += strlen + 2
	return string(v), nil
}

func (b *CommandBuffer) ReadInteger() (int, error) {
	nextCRFL := b.FindNextCRLF()
	if nextCRFL == 0 {
		return 0, ErrParseSyntax
	}
	v := b.data[b.pos:nextCRFL]
	r, err := strconv.Atoi(string(v))
	if err != nil {
		return 0, ErrParseSyntax
	}
	b.pos = nextCRFL + 2
	return r, nil
}

func (b *CommandBuffer) Peek() byte {
	if b.pos == b.length {
		return 0
	}
	return b.data[b.pos]
}

func (b *CommandBuffer) Skip(count int) {
	if b.pos+count >= b.length {
		return
	}
	b.pos += count
}
