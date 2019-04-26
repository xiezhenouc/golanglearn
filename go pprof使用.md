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