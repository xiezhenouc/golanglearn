package snowflake

import (
	"fmt"
	"testing"
)

func TestSnowFlake(t *testing.T) {
	worker, err := NewWorker(1)
	if err != nil {
		fmt.Printf("new worker err %s ", err)
		return
	}

	ch := make(chan int64)
	goroutineNums := 100000
	for i := 0; i < goroutineNums; i++ {
		go func() {
			ch <- worker.GetID()
		}()
	}

	m := make(map[int64]int)
	for i := 0; i < goroutineNums; i++ {
		id := <-ch
		if _, ok := m[id]; ok {
			t.Error("error ! ID repeat!")
			return
		}
		m[id] = 1
	}

	fmt.Println("all", goroutineNums, "snowflake id get success")
}
