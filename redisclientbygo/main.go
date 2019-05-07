package main

import (
	"fmt"
	"redisbygo/redis"
	"time"
)

func main() {
	network := "tcp"
	address := "10.99.198.211:8379"
	connectTimeout := time.Second
	readTimeout := time.Second
	writeTimeout := time.Second
	c, err := redis.DialTimeout(network, address, connectTimeout, readTimeout, writeTimeout)
	if err != nil {
		fmt.Println(err)
		return
	}

	key := "name"
	v, err := c.CMD("get", key)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("value is %s\n", v)
}
