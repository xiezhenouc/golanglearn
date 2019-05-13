package redis

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"
)

// TCP 连接
type conn struct {
	mu sync.Mutex

	err error

	conn net.Conn

	readTimeout time.Duration
	br          *bufio.Reader

	writeTimeout time.Duration
	bw           *bufio.Writer
}

// 创建TCP连接
func DialTimeout(network, address string, connectTimeout, readTimeout, writeTimeout time.Duration) (Conn, error) {
	netConn, err := net.DialTimeout(network, address, connectTimeout)
	if err != nil {
		return nil, err
	}
	c := &conn{
		conn:         netConn,
		readTimeout:  readTimeout,
		br:           bufio.NewReader(netConn),
		bw:           bufio.NewWriter(netConn),
		writeTimeout: writeTimeout,
	}
	return c, nil
}

func (c *conn) WriteLen(prefix byte, n int) error {
	var cmdbuffer []byte
	cmdbuffer = append(cmdbuffer, prefix)

	nStr := strconv.FormatInt(int64(n), 10) + "\r\n"
	cmdbuffer = append(cmdbuffer, []byte(nStr)...)

	_, err := c.bw.Write(cmdbuffer)
	return err
}

func (c *conn) WriteString(s string) error {
	c.WriteLen('$', len(s))
	if _, err := c.bw.WriteString(s); err != nil {
		return err
	}
	if _, err := c.bw.WriteString("\r\n"); err != nil {
		return err
	}
	//fmt.Println("request src: ", s+"\r\n")
	return nil
}

func (c *conn) WriteBytes(p []byte) error {
	c.WriteLen('$', len(p))
	if _, err := c.bw.Write(p); err != nil {
		return err
	}
	if _, err := c.bw.WriteString("\r\n"); err != nil {
		return err
	}
	//fmt.Println("request src: ", string(p)+"\r\n")
	return nil
}

func (c *conn) WriteCommand(cmd string, args []interface{}) error {
	c.WriteLen('*', 1+len(args))
	if err := c.WriteString(cmd); err != nil {
		return err
	}
	for _, arg := range args {
		var err error
		switch argNow := arg.(type) {
		case string:
			err = c.WriteString(argNow)
		case bool:
			if argNow {
				err = c.WriteString("1")
			} else {
				err = c.WriteString("0")
			}
		case nil:
			err = c.WriteString("")
		default:
			var buf bytes.Buffer
			fmt.Fprint(&buf, arg)
			err = c.WriteBytes(buf.Bytes())
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *conn) ReadLine() ([]byte, error) {
	p, err := c.br.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	i := len(p) - 2
	if i < 0 || p[i] != '\r' {
		return nil, errors.New("response end is not \r\n")
	}
	//fmt.Println("responese src:", string(p[:i]))
	return p[:i], nil
}

func (c *conn) ParseInt(p []byte) (int64, error) {
	pInt64, err := strconv.ParseInt(string(p), 10, 64)
	return pInt64, err
}

func (c *conn) ReadReply() (interface{}, error) {
	line, err := c.ReadLine()
	if err != nil {
		return nil, err
	}
	if len(line) == 0 {
		return nil, errors.New("response len is 0")
	}
	switch line[0] {
	case '+':
		return string(line[1:]), nil
	case '-':
		return errors.New(string(line[1:])), nil
	case ':':
		return c.ParseInt(line[1:])
	case '$':
		n, err := c.ParseInt(line[1:])
		if err != nil || n == -1 {
			return nil, err
		}
		if n < 0 {
			return nil, errors.New("n less -1 ")
		}
		line, err := c.ReadLine()
		if int64(len(string(line))) != n {
			return nil, errors.New("$n is not coresponding with string")
		}
		return line, err
	case '*':
		n, err := c.ParseInt(line[1:])
		if err != nil || n == -1 {
			return nil, err
		}
		if n < 0 {
			return nil, errors.New("n less -1 ")
		}

		r := make([]interface{}, n)
		for i := range r {
			r[i], err = c.ReadReply()
			if err != nil {
				return nil, err
			}
		}
		return r, nil
	}
	return nil, errors.New("unexpected response line")
}

func (c *conn) CMD(cmd string, args ...interface{}) (interface{}, error) {
	// 1 向redis服务器发送cmd
	if c.writeTimeout != 0 {
		c.conn.SetWriteDeadline(time.Now().Add(c.writeTimeout))
	}

	if err := c.WriteCommand(cmd, args); err != nil {
		return nil, c.fatal(err)
	}

	if err := c.bw.Flush(); err != nil {
		return nil, err
	}

	// 2 从redis服务器解析结果
	if c.readTimeout != 0 {
		c.conn.SetReadDeadline(time.Now().Add(c.readTimeout))
	}

	reply, err := c.ReadReply()
	return reply, c.fatal(err)
}

func (c *conn) Send(cmd string, args ...interface{}) error {
	if c.writeTimeout != 0 {
		c.conn.SetWriteDeadline(time.Now().Add(c.writeTimeout))
	}

	if err := c.WriteCommand(cmd, args); err != nil {
		return c.fatal(err)
	}

	if err := c.bw.Flush(); err != nil {
		return err
	}

	return nil
}

func (c *conn) Receive() (interface{}, error) {
	if c.readTimeout != 0 {
		c.conn.SetReadDeadline(time.Now().Add(c.readTimeout))
	}

	reply, err := c.ReadReply()
	return reply, c.fatal(err)
}

func (c *conn) Close() error {
	c.mu.Lock()
	err := c.err
	if err == nil {
		c.err = errors.New("redis client : closed")
		err = c.conn.Close()
	}
	c.mu.Unlock()
	return err
}

func (c *conn) Err() error {
	c.mu.Lock()
	err := c.err
	c.mu.Unlock()
	return err
}

func (c *conn) RemoteAddr() net.Addr {
	c.mu.Lock()
	addr := c.conn.RemoteAddr()
	c.mu.Unlock()

	return addr
}

func (c *conn) fatal(err error) error {
	c.mu.Lock()
	if c.err == nil {
		c.err = err
	}
	c.mu.Unlock()

	return err
}
