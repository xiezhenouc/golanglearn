# 生产者消费者
```
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// 共享数据
	dataShare := make(chan int)

	// m个生产者, n个消费者
	m, n := 4, 3

	for i := 0; i < m; i++ {
		// 生产者
		go func(i int) {
			for {
				fmt.Printf("producer %d\n", i)
				dataShare <- i
			}
		}(i)
	}

	for i := 0; i < n; i++ {
		// 消费者
		go func(i int) {
			for msg := range dataShare {
				fmt.Printf("consumer %d msg %d\n", i, msg)
			}
		}(i)
	}

	// 等待结束，需要发送结束信号
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan
	fmt.Println("exit...")
}
```