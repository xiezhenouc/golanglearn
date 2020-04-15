# go pprof使用
## 1 阻塞案例分析
>在开发一个离线worker模块的时候，遇到一个问题，worker偶发会阻塞住，但不知道为什么阻塞住了，因此在worker中添加了pprof，后来再次遇到阻塞的时候，根据查看full goroutine stack dump，看到每个goroutine运行到哪一行，很快就定位到问题。原来是在调用一个封装的lib库时候，传输的请求超时参数为`time.Duration(REQUESTTIMEOUT) * time.Millisecond`，但实际上lib库中已经乘过`time.Millisecond`这会导致请求超时变成一个超大的数字，所以下游偶发问题的时候，导致连接得不到释放，从而阻塞

## 2 pprof的使用
>通过上面的案例，可以看出，pprof很方便

```
package main

import (
	"fmt"
	myframework "github.com/xiezhenouc/golangwebframework"

	"net/http"
	_ "net/http/pprof"
)

type TestController struct {
	Ctx *myframework.Context
}

func (t *TestController) Init(context *myframework.Context) {
	t.Ctx = context
}

func (t *TestController) SayHi() {
	fmt.Fprintln(t.Ctx.Output, "say hi ...")
}

func (t *TestController) SayYes() {
	fmt.Fprintln(t.Ctx.Output, "say yes ...")
}

func init() {
	go func() {
		http.ListenAndServe("0.0.0.0:8888", nil)
	}()
}

func main() {
	fw := myframework.New()

	fw.AddAutoRouter("/test/", &TestController{})

	fw.Run(":8999")
}
```

## 3 web界面
```
/debug/pprof/

profiles:
0	block (查看导致阻塞同步的堆栈跟踪)
6	goroutine (查看当前所有运行的 goroutines 堆栈跟踪)
9	heap (查看活动对象的内存分配情况)
0	mutex (查看导致互斥锁的竞争持有者的堆栈跟踪)
8	threadcreate (查看创建新OS线程的堆栈跟踪)

full goroutine stack dump
```

## 4 火焰图
>业务代码

```golang
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"time"
)

var tz *time.Location

func main() {
	go func() {
		for {
			LocalTz()

			doSomething([]byte(`{"a": 1, "b": 2, "c": 3}`))
		}
	}()

	fmt.Println("start api server...")
	panic(http.ListenAndServe(":8080", nil))
}

func doSomething(s []byte) {
	var m map[string]interface{}
	err := json.Unmarshal(s, &m)
	if err != nil {
		panic(err)
	}

	s1 := make([]string, 0)
	s2 := ""
	for i := 0; i < 100; i++ {
		s1 = append(s1, string(s))
		s2 += string(s)
	}
}

func LocalTz() *time.Location {
	if tz == nil {
		tz, _ = time.LoadLocation("Asia/Shanghai")
	}
	return tz
}
```

>数据采集

```
go tool pprof http://127.0.0.1:8080/debug/pprof/profile -seconds 10

...
Saved profile in /Users/xxx/pprof/pprof.samples.cpu.001.pb.gz
```

>结果再输出

```
go tool pprof -http=:8081 /Users/xxx/pprof/pprof.samples.cpu.001.pb.gz
```

>火焰图

![火焰图](https://github.com/xiezhenouc/golanglearn/blob/master/%E5%9B%BE%E7%89%87%E8%AF%B4%E6%98%8E/火焰图.png)

>图中，从上往下是方法的调用栈，长度代表cpu时长

>可以看出 LocalTz() 函数占据了 50% cpu，开销非常惊人，快速定位到此函数