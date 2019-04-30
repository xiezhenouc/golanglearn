# https://github.com/garyburd/redigo 数据截断问题记录

## 1 问题背景
>在写代码的时候，需要读取redis内容，因此使用https://github.com/garyburd/redigo这个sdk来进行redis的使用
>
>用的过程中发现，大的消息被截断了，报错误`redigo: long response line (possible server error or unsupported concurrent read by application)`
>
>通过测试，发现消息在大于4KB左右的时候，消息会被截断

## 2 问题定位
>通过查阅源redigo的代码，发现报错的代码如下

```
// redigo/redis/conn.go
func (c *conn) readLine() ([]byte, error) {
	p, err := c.br.ReadSlice('\n')
	if err == bufio.ErrBufferFull {
		return nil, protocolError("long response line")
	}
	if err != nil {
		return nil, err
	}
	i := len(p) - 2
	if i < 0 || p[i] != '\r' {
		return nil, protocolError("bad response line terminator")
	}
	return p[:i], nil
}
// c.br初始化
func NewConn(netConn net.Conn, readTimeout, writeTimeout time.Duration) Conn {
	return &conn{
		conn:         netConn,
		bw:           bufio.NewWriter(netConn),
		br:           bufio.NewReader(netConn),
		readTimeout:  readTimeout,
		writeTimeout: writeTimeout,
	}
}
```

>通过查询 ReadSlice 的源码，发现底层buf的初始值为4096，并且会返回底层的slice。

```
//https://golang.org/src/bufio/bufio.go
func NewReader(rd io.Reader) *Reader {
	return NewReaderSize(rd, defaultBufSize)
}
func (b *Reader) ReadSlice(delim byte) (line []byte, err error) {
	s := 0 // search start index
	for {
		....
		// Buffer full?
		if b.Buffered() >= len(b.buf) {
			b.r = b.w
			line = b.buf
			err = ErrBufferFull
			break
		}
		....
}
```

> ReadBytes、ReadString 和 ReadLine 核心都是调用ReadSlice，但是对其进行了扩充和封装。ReadBytes源码如下，可以明显看到，对buffer满的情况做了兼容，不会导致缓冲区溢出。

```
//https://golang.org/src/bufio/bufio.go
func (b *Reader) ReadBytes(delim byte) ([]byte, error) {
	// Use ReadSlice to look for array,
	// accumulating full buffers.
	var frag []byte
	var full [][]byte
	var err error
	for {
		var e error
		frag, e = b.ReadSlice(delim)
		if e == nil { // got final fragment
			break
		}
		if e != ErrBufferFull { // unexpected error
			err = e
			break
		}

		// Make a copy of the buffer.
		buf := make([]byte, len(frag))
		copy(buf, frag)
		full = append(full, buf)
	}

	// Allocate new buffer to hold the full pieces and the fragment.
	n := 0
	for i := range full {
		n += len(full[i])
	}
	n += len(frag)

	// Copy full pieces and fragment in.
	buf := make([]byte, n)
	n = 0
	for i := range full {
		n += copy(buf[n:], full[i])
	}
	copy(buf[n:], frag)
	return buf, err
}
```

## 3 问题复现
## 3.1 原问题复现
>发现问题之后，就考虑复现一下
>但是线下自己搭建一个redis之后，往一个key里面写入远大于1M的数据，都能正常读取。因此需要对比通过线下redis读取方式，和我遇到问题时候读取方式的区别。
>我在使用redis的时候，`其实是远程服务封装了redis协议`，现在出现问题说明远程服务`封装的redis协议`和`标准的redis协议`肯定有差异点
>
### 3.1.1 标准redis协议
>标准redis协议见`https://redis.io/topics/protocol`
>主要内容如下

```
RESP 协议描述
RESP协议在Redis 1.2中引入，但它成为与Redis 2.0中的Redis服务器通信的标准方式。 这是每一个Redis客户端中应该实现的协议。
RESP实际上是一个支持以下数据类型的序列化协议：单行字符串，错误信息，整型，多行字符串和数组。

RESP在Redis中用作请求 - 响应协议的方式如下：
客户端将命令作为字符串数组发送到Redis服务器。
服务器根据命令实现回复一种RESP类型数据。

在 RESP 中, 一些数据的类型通过它的第一个字节进行判断：

单行回复：回复的第一个字节是 "+"
错误信息：回复的第一个字节是 "-"
整形数字：回复的第一个字节是 ":"
多行字符串：回复的第一个字节是 "$"
数组：回复的第一个字节是 "*"

此外，RESP能够使用稍后指定的Bulk Strings或Array的特殊变体来表示Null值。
在RESP中，协议的不同部分始终以“\ r \ n”（CRLF）结束。
```
### 3.1.2 差异点
>通过打印`标准redis协议`和`封装Redis协议`的返回值，直接在得到二进制流的时候进行打印，结果如下
>
>`标准redis协议`的返回结果

```
$45056
```

>`封装Redis协议`

```
+{"_action":"xxx","_callid":"0.4","_logid":xxx}.....
```
>能够明显看到，原来封装的redis协议返回方式是单行回复，但是标准的Redis协议是多行回复，但多行回复后续为什么没有问题，只能再去看源码
### 3.1.3 多行回复没有问题的原因--问题的答案
>请看关键点问题，可以明显看到，`p := make([]byte, n) _, err = io.ReadFull(c.br, p)`，这个就是将底层的buffer空间扩大了。然后再`c.readLine()`，所以就没有问题了
>但是在首次调用c.readLine()时候，如果是用单行回复，`+...`这种形式，且`c.readLine()`的内部实现是`ReadSlice`，则当单行回复的消息太大的时候，`ReadSlice`只能读取4096字节消息，问题的根因

```
// redigo/redis/conn.go
func (c *conn) readReply() (interface{}, error) {
   // 首次调用c.readLine()
	line, err := c.readLine()
	if err != nil {
		return nil, err
	}
	if len(line) == 0 {
		return nil, protocolError("short response line")
	}
	switch line[0] {
	case '+':
		switch {
		case len(line) == 3 && line[1] == 'O' && line[2] == 'K':
			// Avoid allocation for frequent "+OK" response.
			return okReply, nil
		case len(line) == 5 && line[1] == 'P' && line[2] == 'O' && line[3] == 'N' && line[4] == 'G':
			// Avoid allocation in PING command benchmarks :)
			return pongReply, nil
		default:
			return string(line[1:]), nil
		}
	case '-':
		return Error(string(line[1:])), nil
	case ':':
		return parseInt(line[1:])
	case '$':
		n, err := parseLen(line[1:])
		if n < 0 || err != nil {
			return nil, err
		}
		// 关键点!!!
		p := make([]byte, n)
		_, err = io.ReadFull(c.br, p)
		if err != nil {
			return nil, err
		}
		if line, err := c.readLine(); err != nil {
			return nil, err
		} else if len(line) != 0 {
			return nil, protocolError("bad bulk string format")
		}
...
```
### 3.1.4 代码

```
package main

import (
	"fmt"
	"golib/redigo/redis"
)

func main() {
	conn, err := redis.Dial("tcp", "127.0.0.1:6379")
	// 远程服务ip port
	// conn, err := redis.Dial("tcp", ":")
	if err != nil {
		fmt.Println(err)
		return
	}
	for {
		data, err := redis.String(conn.Do("GET", "xxx"))
		if err != nil {
			fmt.Println("for err", err)
			continue
		}
		if len(data) > 4096 {
			fmt.Println("succ")
		} else {
			//fmt.Println("fail")
		}
	}
}
```

## 3.2 redis monitor模式复现
>在网上查询这个问题的时候，发现新的版本已经修复了。具体的issue为`https://github.com/gomodule/redigo/issues/380`，PR为`https://github.com/gomodule/redigo/pull/381/files`
>issue的主要问题是使用了redis的monitor功能
>redis monitor功能介绍，主要是能看到redis的操作记录

```
语法
redis Monitor 命令基本语法如下：
redis 127.0.0.1:6379> MONITOR 
可用版本
>= 1.0.0

返回值
总是返回 OK 。

实例
redis 127.0.0.1:6379> MONITOR 
OK
1410855382.370791 [0 127.0.0.1:60581] "info"
1410855404.062722 [0 127.0.0.1:60581] "get" "a"
```

>可以复现的代码如下，注意conn.Receive()这个函数

```
package main

import (
	"fmt"
	"golib/redigo/redis"
)

func main() {
	conn, err := redis.Dial("tcp", "127.0.0.1:6379")
	if err != nil {
		fmt.Println(err)
		return
	}
	_, err = conn.Do("monitor")
	if err != nil {
		fmt.Println(err)
		return
	}
	for {
		data, err := conn.Receive()
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Printf("%s\n", data)
	}
}
```

>conn.Receive()里面还是最终调用到了ReadSlice位置

```
// redigo/redis/conn.go
func (c *conn) Receive() (reply interface{}, err error) {
	if c.readTimeout != 0 {
		c.conn.SetReadDeadline(time.Now().Add(c.readTimeout))
	}
	if reply, err = c.readReply(); err != nil {
		return nil, c.fatal(err)
	}
	...
}

func (c *conn) readReply() (interface{}, error) {
	line, err := c.readLine()
	if err != nil {
		return nil, err
	}
	...
}
```

>将monitor模式下server返回打印出来，结果如下

```
+1556622707.140954 [0 127.0.0.1:35436] "SET" "netdisk_filesearch_1" "1122.......
bufio: buffer full
redigo: long response line (possible server error or unsupported concurrent read by application)
```

## 4 总结
>1 遇到问题的时候，先看是否能复现。
>
>如本次情况，在标准redis情况下不能复现，在封装的redis情况下必须，然后就找不同
>
>2 定位问题的思路。
>
>先走到协议的最底层，通过查看收到的二进制流开始查看
 