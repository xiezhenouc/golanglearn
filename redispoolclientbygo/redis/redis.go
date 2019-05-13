// redis conn interface
package redis

import "net"

type Conn interface {
	// close connection
	Close() error

	// err
	Err() error

	// execute redis command
	CMD(cmd string, args ...interface{}) (interface{}, error)

	// send cmd to redis server
	Send(cmd string, args ...interface{}) error

	// receive cmd from redis server
	Receive() (interface{}, error)

	// redis server addr
	RemoteAddr() net.Addr
}
