package main

import (
	"fmt"
	"github.com/xiezhenouc/golanglearn/redispoolclientbygo/redis"
	"time"
)

type Redis struct {
	_pool *redis.Pool
}

func (r *Redis) initHandlerPool() (*redis.Pool, error) {
	connTimeout := 500 * time.Millisecond
	readTimeout := 1000 * time.Millisecond
	writeTimeout := 1000 * time.Millisecond

	//init redis connection pool
	_pool := &redis.Pool{
		MaxIdle:     1,
		MaxActive:   2,
		IdleTimeout: time.Duration(1) * time.Second,
		Dial: func() (redis.Conn, error) {
			address := "10.99.198.211:8379"
			c, err := redis.DialTimeout("tcp", address, connTimeout, readTimeout, writeTimeout)
			if err != nil {
				return nil, err
			}
			return c, err
		},
		Wait: true,
		TestPing: func(c redis.Conn, t time.Time) error {
			if time.Since(t) < time.Minute {
				return nil
			}

			_, err := c.CMD("PING")
			return err
		},
	}

	return _pool, nil
}

func (t *Redis) Get(key string) (interface{}, error) {
	conn := t._pool.Get()
	defer conn.Close()
	if err := conn.Err(); err != nil {
		return "", fmt.Errorf("[get connection failed] [error:%s]", err)
	}

	v, err := conn.CMD("GET", key)
	if err == nil {
		return v, nil
	}

	return "", fmt.Errorf("[get key failed] [error:%s]", err)
}

func main() {
	// 1 initial
	RecordCache := &Redis{}
	var err error
	RecordCache._pool, err = RecordCache.initHandlerPool()
	if err != nil {
		fmt.Println(err)
		return
	}

	// 2 operation
	key := "name"
	val, err := RecordCache.Get(key)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(string(val.([]byte)))
	}

	// 3 close pool
	RecordCache._pool.Close()
}
