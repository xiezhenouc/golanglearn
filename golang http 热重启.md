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

```golang
// 500ms检查一次是否还有idle connections
var shutdownPollInterval = 500 * time.Millisecond

// Shutdown gracefully shuts down the server without interrupting any
// active connections. Shutdown works by first closing all open
// listeners, then closing all idle connections, and then waiting
// indefinitely for connections to return to idle and then shut down.
// If the provided context expires before the shutdown is complete,
// Shutdown returns the context's error, otherwise it returns any
// error returned from closing the Server's underlying Listener(s).
//
// When Shutdown is called, Serve, ListenAndServe, and
// ListenAndServeTLS immediately return ErrServerClosed. Make sure the
// program doesn't exit and waits instead for Shutdown to return.
//
// Shutdown does not attempt to close nor wait for hijacked
// connections such as WebSockets. The caller of Shutdown should
// separately notify such long-lived connections of shutdown and wait
// for them to close, if desired.
func (srv *Server) Shutdown(ctx context.Context) error {
    // 数字1 标记当前处于正在退出状态， func (s *Server) shuttingDown() bool函数会判断这个值
	atomic.AddInt32(&srv.inShutdown, 1)
	defer atomic.AddInt32(&srv.inShutdown, -1)

	srv.mu.Lock()
    // 将listen fd进行关闭，新连接无法建立，l.Accept() 将失败
    lnerr := srv.closeListenersLocked()
    
    // 把server.go的done chan给close掉，将Serve()主函数退出，不再接受新请求 return ErrServerClosed
    srv.closeDoneChanLocked()
    
    // 如果注册了shutdown的回调方法，执行回调方法
	for _, f := range srv.onShutdown {
		go f()
	}
	srv.mu.Unlock()

	ticker := time.NewTicker(shutdownPollInterval)
	defer ticker.Stop()
	for {
        // 关闭空闲连接
		if srv.closeIdleConns() {
			return lnerr
		}
		select {
        // ctx 超时，直接退出
		case <-ctx.Done():
            return ctx.Err()
        // 每500ms检查一下idle
		case <-ticker.C:
		}
	}
}

// 判断是否需要http长连接 or 判断是否处于关闭状态
func (s *Server) doKeepAlives() bool {
	return atomic.LoadInt32(&s.disableKeepAlives) == 0 && !s.shuttingDown()
}

// 判断server是否处于正在退出状态
func (s *Server) shuttingDown() bool {
	return atomic.LoadInt32(&s.inShutdown) != 0
}

// 将listen fd进行关闭，新连接无法建立
func (s *Server) closeListenersLocked() error {
	var err error
	for ln := range s.listeners {
		if cerr := ln.Close(); cerr != nil && err == nil {
			err = cerr
		}
		delete(s.listeners, ln)
	}
	return err
}

// 执行closeListenersLocked后，l.Accept()无法建立连接，直接返回close
func (srv *Server) Serve(l net.Listener) error {
    ...
	baseCtx := context.Background() // base is always background, per Issue 16220
	ctx := context.WithValue(baseCtx, ServerContextKey, srv)
	for {
        // 当执行closeListenersLocked后，e返回错误
		rw, e := l.Accept()
		if e != nil {
			select {
            // 把server.go的done chan给close掉，将Serve()主函数退出，不再接受新请求 return ErrServerClosed
			case <-srv.getDoneChan():
				return ErrServerClosed
			default:
			}
			...
			return e
		}
		tempDelay = 0
		c := srv.newConn(rw)
        c.setState(c.rwc, StateNew) // before Serve can return
        // 每一个连接单独启一个goroutine
		go c.serve(ctx) 
	}
}

// get done chan
func (s *Server) getDoneChanLocked() chan struct{} {
	if s.doneChan == nil {
		s.doneChan = make(chan struct{})
	}
	return s.doneChan
}

// 把server.go的done chan给close掉，将Serve()主函数退出，不再接受新请求 return ErrServerClosed
func (s *Server) closeDoneChanLocked() {
	ch := s.getDoneChanLocked()
	select {
	case <-ch:
		// Already closed. Don't close again.
	default:
		// Safe to close here. We're the only closer, guarded
		// by s.mu.
		close(ch)
	}
}

// 关闭所有空闲连接，注意！是空闲连接，正在使用中的连接不会关闭
// closeIdleConns closes all idle connections and reports whether the
// server is quiescent.
func (s *Server) closeIdleConns() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	quiescent := true
	for c := range s.activeConn {
		st, ok := c.curState.Load().(ConnState)
		if !ok || st != StateIdle {
			quiescent = false
			continue
		}
		c.rwc.Close()
		delete(s.activeConn, c)
	}
	return quiescent
}

// net/net.go 关闭fd
// Close closes the connection.
func (c *conn) Close() error {
	if !c.ok() {
		return syscall.EINVAL
	}
	err := c.fd.Close()
	if err != nil {
		err = &OpError{Op: "close", Net: c.fd.net, Source: c.fd.laddr, Addr: c.fd.raddr, Err: err}
	}
	return err
}

// 一个connection的完整生命周期
// Serve a new connection.
func (c *conn) serve(ctx context.Context) {
    ...
    ctx = context.WithValue(ctx, LocalAddrContextKey, c.rwc.LocalAddr())
    ...

	// HTTP/1.x from here on.
	ctx, cancelCtx := context.WithCancel(ctx)
	c.cancelCtx = cancelCtx
	defer cancelCtx()

	c.r = &connReader{conn: c}
	c.bufr = newBufioReader(c.r)
	c.bufw = newBufioWriterSize(checkConnErrorWriter{c}, 4<<10)

	for {
        // 读取请求
		w, err := c.readRequest(ctx)

		c.curReq.Store(w)

        // 执行业务逻辑
		// HTTP cannot have multiple simultaneous active requests.[*]
		// Until the server replies to this request, it can't read another,
		// so we might as well run the handler in this goroutine.
		// [*] Not strictly true: HTTP pipelining. We could let them all process
		// in parallel even if their responses need to be serialized.
		// But we're not going to implement HTTP pipelining because it
		// was never deployed in the wild and the answer is HTTP/2.
		serverHandler{c.server}.ServeHTTP(w, w.req)
		w.cancelCtx()
		if c.hijacked() {
			return
        }
        
        // 结束请求
		w.finishRequest()

        // 连接设置为空闲
		c.setState(c.rwc, StateIdle)
		c.curReq.Store((*response)(nil))

        // 处于shutdown模式下，协程退出，完成对请求方的响应
		if !w.conn.server.doKeepAlives() {
			// We're in shutdown mode. We might've replied
			// to the user without "Connection: close" and
			// they might think they can send another
			// request, but such is life with HTTP/1.1.
			return
		}
        ... 
	}
}

// server向请求方发送数据
func (w *response) finishRequest() {
	w.handlerDone.setTrue()

	if !w.wroteHeader {
		w.WriteHeader(StatusOK)
	}

	w.w.Flush()
	putBufioWriter(w.w)
	w.cw.close()
	w.conn.bufw.Flush()

	w.conn.r.abortPendingRead()

	// Close the body (regardless of w.closeAfterReply) so we can
	// re-use its bufio.Reader later safely.
	w.reqBody.Close()

	if w.req.MultipartForm != nil {
		w.req.MultipartForm.RemoveAll()
	}
}
```