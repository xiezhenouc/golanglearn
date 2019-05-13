package redis

import (
	"bufio"
	"bytes"
	"io"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"
)

type testConn struct {
	io.Reader
	io.Writer
}

func dialTestConn(r io.Reader, w io.Writer) (Conn, error) {
	return &conn{
		br: bufio.NewReader(r),
		bw: bufio.NewWriter(w),
	}, nil
}

func dialDefaultServer() (Conn, error) {
	network := "tcp"
	address := "10.99.198.211:8379"
	connectTimeout := time.Second
	readTimeout := time.Second
	writeTimeout := time.Second
	c, err := DialTimeout(network, address, connectTimeout, readTimeout, writeTimeout)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// 1 写入测试
var writeTests = []struct {
	args     []interface{}
	expected string
}{
	{
		[]interface{}{"SET", "key", "value"},
		"*3\r\n$3\r\nSET\r\n$3\r\nkey\r\n$5\r\nvalue\r\n",
	},
	{
		[]interface{}{"SET", "key", "value"},
		"*3\r\n$3\r\nSET\r\n$3\r\nkey\r\n$5\r\nvalue\r\n",
	},
	{
		[]interface{}{"SET", "key", byte(100)},
		"*3\r\n$3\r\nSET\r\n$3\r\nkey\r\n$3\r\n100\r\n",
	},
	{
		[]interface{}{"SET", "key", 100},
		"*3\r\n$3\r\nSET\r\n$3\r\nkey\r\n$3\r\n100\r\n",
	},
	{
		[]interface{}{"SET", "key", int64(math.MinInt64)},
		"*3\r\n$3\r\nSET\r\n$3\r\nkey\r\n$20\r\n-9223372036854775808\r\n",
	},
	{
		[]interface{}{"SET", "key", float64(1349673917.939762)},
		"*3\r\n$3\r\nSET\r\n$3\r\nkey\r\n$21\r\n1.349673917939762e+09\r\n",
	},
	{
		[]interface{}{"SET", "key", ""},
		"*3\r\n$3\r\nSET\r\n$3\r\nkey\r\n$0\r\n\r\n",
	},

	{
		[]interface{}{"SET", "key", nil},
		"*3\r\n$3\r\nSET\r\n$3\r\nkey\r\n$0\r\n\r\n",
	},
	{
		[]interface{}{"ECHO", true, false},
		"*3\r\n$4\r\nECHO\r\n$1\r\n1\r\n$1\r\n0\r\n",
	},
}

func TestWrite(t *testing.T) {
	for _, tt := range writeTests {
		var buf bytes.Buffer
		c, err := dialTestConn(nil, &buf)
		if err != nil {
			t.Errorf("dialTestConn err %v", err)
			break
		}
		err = c.Send(tt.args[0].(string), tt.args[1:]...)
		if err != nil {
			t.Errorf("Send %v return err %v", tt.args, err)
			continue
		}
		actual := buf.String()
		if actual != tt.expected {
			t.Errorf("Send %v actual %v want %v", tt.args, actual, tt.expected)
			continue
		}
	}
}

// 2 读取测试
var errorSentinel = &struct{}{}

var readTests = []struct {
	reply    string
	expected interface{}
}{
	{
		"+OK\r\n",
		"OK",
	},
	{
		"+PONG\r\n",
		"PONG",
	},
	{
		"@OK\r\n",
		errorSentinel,
	},
	{
		"$6\r\nfoobar\r\n",
		[]byte("foobar"),
	},
	{
		"$-1\r\n",
		nil,
	},
	{
		":1\r\n",
		int64(1),
	},
	{
		":-2\r\n",
		int64(-2),
	},
	{
		"*0\r\n",
		[]interface{}{},
	},
	{
		"*-1\r\n",
		nil,
	},
	{
		"*4\r\n$3\r\nfoo\r\n$3\r\nbar\r\n$5\r\nHello\r\n$5\r\nWorld\r\n",
		[]interface{}{[]byte("foo"), []byte("bar"), []byte("Hello"), []byte("World")},
	},
	{
		"*3\r\n$3\r\nfoo\r\n$-1\r\n$3\r\nbar\r\n",
		[]interface{}{[]byte("foo"), nil, []byte("bar")},
	},

	{
		// "x" is not a valid length
		"$x\r\nfoobar\r\n",
		errorSentinel,
	},
	{
		// -2 is not a valid length
		"$-2\r\n",
		errorSentinel,
	},
	{
		// "x"  is not a valid integer
		":x\r\n",
		errorSentinel,
	},
	{
		// missing \r\n following value
		"$6\r\nfoobar",
		errorSentinel,
	},
	{
		// short value
		"$6\r\nxx",
		errorSentinel,
	},
	{
		// long value
		"$6\r\nfoobarx\r\n",
		errorSentinel,
	},
}

func TestRead(t *testing.T) {
	for _, tt := range readTests {
		c, err := dialTestConn(strings.NewReader(tt.reply), nil)
		if err != nil {
			t.Errorf("dialTestConn err %v", err)
			break
		}
		actual, err := c.Receive()
		if tt.expected == errorSentinel {
			if err == nil {
				t.Errorf("receive %v %v err %v actual %v", tt.reply, tt.expected, err, actual)
				break
			}
			// err != nil 符合预期
		} else {
			if !reflect.DeepEqual(actual, tt.expected) {
				t.Errorf("receive replay %v actual %v expected %v", tt.reply, actual, tt.expected)
				break
			}
		}
	}
}

// 3 cmd命令测试(需要连接redis测试服务器)
var testCommands = []struct {
	args     []interface{}
	expected interface{}
}{
	{
		[]interface{}{"PING"},
		"PONG",
	},
	{
		[]interface{}{"SET", "foo", "bar"},
		"OK",
	},
	{
		[]interface{}{"GET", "foo"},
		[]byte("bar"),
	},
	{
		[]interface{}{"GET", "nokey"},
		nil,
	},
	{
		[]interface{}{"MGET", "nokey", "foo"},
		[]interface{}{nil, []byte("bar")},
	},
	{
		[]interface{}{"INCR", "mycounter"},
		int64(1),
	},
	{
		[]interface{}{"DEL", "mycounter"},
		int64(1),
	},
	{
		[]interface{}{"LPUSH", "mylist", "foo"},
		int64(1),
	},
	{
		[]interface{}{"LPUSH", "mylist", "bar"},
		int64(2),
	},
	{
		[]interface{}{"LRANGE", "mylist", 0, -1},
		[]interface{}{[]byte("bar"), []byte("foo")},
	},
	{
		[]interface{}{"MULTI"},
		"OK",
	},
	{
		[]interface{}{"LRANGE", "mylist", 0, -1},
		"QUEUED",
	},
	{
		[]interface{}{"PING"},
		"QUEUED",
	},
	{
		[]interface{}{"EXEC"},
		[]interface{}{
			[]interface{}{[]byte("bar"), []byte("foo")},
			"PONG",
		},
	},
	{
		[]interface{}{"DEL", "mylist"},
		int64(1),
	},
}

func TestCMD(t *testing.T) {
	c, err := dialDefaultServer()
	if err != nil {
		t.Errorf("connect to redis server error %v", err)
	}

	for _, cmd := range testCommands {
		actual, err := c.CMD(cmd.args[0].(string), cmd.args[1:]...)
		if err != nil {
			t.Errorf("redis cmd %v return error %v", cmd.args, err)
			continue
		}
		if !reflect.DeepEqual(actual, cmd.expected) {
			t.Errorf("redis cmd args %v actual %v want %v", cmd.args, actual, cmd.expected)
		}
	}
}
