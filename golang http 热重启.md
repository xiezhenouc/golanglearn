# golang http 热重启 demo

>实现思路参考 https://www.cnblogs.com/sunsky303/p/9778466.html
>
>golang http服务器热重启/热升级/热更新
>
>在日常业务中，我们经常会遇到一个问题，就是在线业务的不停服升级。
>
>在L(inux)N(ginx)M(ySQL)P(HP)架构下，依靠Nginx的reload功能。Nginx reload时候，会创建新的worker去承接新请求，老的worker不再接受新的请求，老请求结束后老worker退出，只剩下新worker继续工作，完成无损热升级。
>
>当然，可以使用nginx做一层代理，后端使用golang http做server。但也可以直接用golang做server。本文将会介绍如何进行golang http的热重启。官方golang 1.8版本提供了ShutDown方法，可以借助这个基础，进行开发。开源项目中facebook的grace做了同样的事情 ( https://github.com/facebookarchive/grace )，不过过程稍微复杂一点。
>
>本文做一个简单的demo示例。
>

## 1 主要步骤和细节

### 1.1 原理
> 1 监听信号 （监听 SIGUSR2 信号）
> 信号概念:是Unix、类Unix以及其他POSIX兼容的操作系统中进程间通讯的一种有限制的方式。它是一种异步的通知机制，用来提醒进程一个事件已经发生。常见信号。Ctrl-C发送INT信号（SIGINT）；默认情况下，这会导致进程终止。Ctrl-Z发送TSTP信号（SIGTSTP）；默认情况下，这会导致进程挂起。kill 允许用户向进程发送信号。kill -9 = kill -s SIGKILL 

```shell
$ kill -l
 1) SIGHUP	 2) SIGINT	 3) SIGQUIT	 4) SIGILL	 5) SIGTRAP
 6) SIGABRT	 7) SIGBUS	 8) SIGFPE	 9) SIGKILL	10) SIGUSR1
11) SIGSEGV	12) SIGUSR2	13) SIGPIPE	14) SIGALRM	15) SIGTERM
16) SIGSTKFLT	17) SIGCHLD	18) SIGCONT	19) SIGSTOP	20) SIGTSTP
21) SIGTTIN	22) SIGTTOU	23) SIGURG	24) SIGXCPU	25) SIGXFSZ
26) SIGVTALRM	27) SIGPROF	28) SIGWINCH	29) SIGIO	30) SIGPWR
31) SIGSYS	34) SIGRTMIN	35) SIGRTMIN+1	36) SIGRTMIN+2	37) SIGRTMIN+3
38) SIGRTMIN+4	39) SIGRTMIN+5	40) SIGRTMIN+6	41) SIGRTMIN+7	42) SIGRTMIN+8
43) SIGRTMIN+9	44) SIGRTMIN+10	45) SIGRTMIN+11	46) SIGRTMIN+12	47) SIGRTMIN+13
48) SIGRTMIN+14	49) SIGRTMIN+15	50) SIGRTMAX-14	51) SIGRTMAX-13	52) SIGRTMAX-12
53) SIGRTMAX-11	54) SIGRTMAX-10	55) SIGRTMAX-9	56) SIGRTMAX-8	57) SIGRTMAX-7
58) SIGRTMAX-6	59) SIGRTMAX-5	60) SIGRTMAX-4	61) SIGRTMAX-3	62) SIGRTMAX-2
63) SIGRTMAX-1	64) SIGRTMAX
```
> 2 收到 SIGUSR2 信号，fork新的子进程，（新的子进程使用相同的命令启动），将父进程服务监听的文件描述符传递给子进程
>
> 3 子进程监听父进程的socket，这个时候子进程和父进程都可以接收请求
>
> 4 子进程启动成功后，父进程会停止接收新的请求，等待旧连接处理完成(golang 1.8 shutdown())或者超时
>
> 5 父进程退出。热重启完成。

### 1.2 源代码实现&详解

```golang
// test.go
package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

var (
	// 第二次重启命令 ./test -graceful
	graceful = flag.Bool("graceful", false, "for grace restart")
	// 全局server，后续Shutdown使用
	server *http.Server
	// 父进程和子进程传递的socket
	listener net.Listener
)

func handler(w http.ResponseWriter, r *http.Request) {
	// 第二次go build时，修改打印内容
	w.Write([]byte("hello world eee"))
}

func main() {
	http.HandleFunc("/hello", handler)
	server = &http.Server{Addr: ":8321"}

	flag.Parse()
	var err error
	if *graceful {
		// 子进程继承fd
		// 为什么是3呢，而不是1 0 或者其他数字？
		// 这是因为父进程将 fd 给子进程了
		// 而子进程里0，1，2是预留给 标准输入、输出和错误的
		// 所以父进程给的第一个fd在子进程里顺序排就是从3开始了；
		// 如果fork的时候cmd.ExtraFiles给了两个文件句柄，那么子进程里还可以用4开始
		f := os.NewFile(3, "")
		listener, err = net.FileListener(f)
	} else {
		// 父进程监听端口
		listener, err = net.Listen("tcp", server.Addr)
	}

	if err != nil {
		log.Fatalf("err %v", err)
	}

	log.Println("start :6888")
	log.Println("pid :", os.Getpid())

	go func() {
		err := server.Serve(listener)
		log.Println(err)
	}()

	// 父进程监听信号
	signalHandler()
}

func signalHandler() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR2)
	for {
		sig := <-ch
		log.Printf("signal:%v", sig)
		ctx, _ := context.WithTimeout(context.Background(), time.Second*10)
		switch sig {
		// 监听 SIGINT SIGTERM 信号
		case syscall.SIGINT, syscall.SIGTERM:
			log.Println("stop")
			signal.Stop(ch)
			server.Shutdown(ctx)
			log.Println("graceful shutdown")
			return
		// 监听 SIGUSR2 信号，热重启
		case syscall.SIGUSR2:
			log.Println("reload")
			// 启动子进程
			err := reload()
			if err != nil {
				log.Fatal("reload err ", err)
			}
			// 父进程优雅退出
			server.Shutdown(ctx)
			log.Println("graceful reload")
			return
		}
	}
}

func reload() error {
	// 父进程的listener准备往子进程传递
	tl, ok := listener.(*net.TCPListener)
	if !ok {
		return errors.New("listener is not *net.Listener")
	}
	f, err := tl.File()
	if err != nil {
		return err
	}

	args := []string{"-graceful"}
	cmd := exec.Command(os.Args[0], args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.ExtraFiles = []*os.File{f}

	// 子进程启动
	return cmd.Start()
}
```

### 1.3 运行

> 1 启动服务

```
$ go build test.go
$ ./test &
2019/06/11 19:31:45 start :6888
2019/06/11 19:31:45 pid : 21816

``` 

> 2 运行测试程序，源代码如下

```python
#!/usr/bin/env python
# -*- coding: utf-8 -*-
import os
import commands
import time

i = 0
while True:
    i += 1
    (status, output) = commands.getstatusoutput('''curl "http://127.0.0.1:8321/hello"''')
    if 'hello' not in output:
        print 'error'
        print status, output
        time.sleep(10)
    else:
        print status, output
```

> 输出如下

```
0   % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current
                                 Dload  Upload   Total   Spent    Left  Speed
100    15  100    15    0     0     15      0  0:00:01 --:--:--  0:00:01 15000
hello world eee
...
```

> 3 修改test.go，将`hello world eee`改为`hello world fff`，重新编译，热重启

```
$ go build test.go
$ kill -s SIGUSR2 21816
$ ps aux | grep test
work     15026  0.0  0.0  54448  3080 pts/0    Sl   19:37   0:00 ./test -graceful
work     15327  0.0  0.0 101140   896 pts/0    S+   19:37   0:00 grep test
```

> 4 查看python输出，已经修改为

```
0   % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current
                                 Dload  Upload   Total   Spent    Left  Speed
100    15  100    15    0     0     15      0  0:00:01 --:--:--  0:00:01 15000
hello world fff
...
```

## 2 go1.8 Shutdown源码
>todo