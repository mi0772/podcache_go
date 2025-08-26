package resp_test

import (
	"fmt"
	"mi0772/podcache/resp"
	"testing"
)

func TestCommand(t *testing.T) {
	cmd := "*1\r\n$4\r\nQUIT\r\n"

	c, err := resp.Parse(cmd)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(c)
}

func TestCommandQuit(t *testing.T) {
	cmd := "*1\r\n$4\r\nQUIT\r\n"
	c, err := resp.Parse(cmd)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(c)
}

func TestCommandSet(t *testing.T) {
	cmd := "*3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$3\r\nbar\r\n"
	c, err := resp.Parse(cmd)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(c)
}

func TestCommandGet(t *testing.T) {
	cmd := "*2\r\n$3\r\nGET\r\n$3\r\nfoo\r\n"
	c, err := resp.Parse(cmd)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(c)
}

func TestCommandDel(t *testing.T) {
	cmd := "*2\r\n$3\r\nDEL\r\n$3\r\nfoo\r\n"
	c, err := resp.Parse(cmd)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(c)
}

func TestCommandPing(t *testing.T) {
	cmd := "*1\r\n$4\r\nPING\r\n"
	c, err := resp.Parse(cmd)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(c)
}

func TestCommandClient(t *testing.T) {
	cmd := "*2\r\n$6\r\nCLIENT\r\n$4\r\nINFO\r\n"
	c, err := resp.Parse(cmd)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(c)
}

func TestCommandUnknown(t *testing.T) {
	cmd := "*1\r\n$7\r\nFOOBAR\r\n"
	c, err := resp.Parse(cmd)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(c)
}

func TestCommandIncr(t *testing.T) {
	cmd := "*2\r\n$4\r\nINCR\r\n$3\r\nfoo\r\n"
	c, err := resp.Parse(cmd)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(c)
}

func TestCommandUnlink(t *testing.T) {
	cmd := "*2\r\n$6\r\nUNLINK\r\n$3\r\nfoo\r\n"
	c, err := resp.Parse(cmd)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(c)
}
