# net/http包初探
>背景：同事遇到了一个问题，发现当用"Content-Type:application/x-www-form-urlencoded"进行HTTP请求的时候，程序异常退出了，最终的原因是他用的网络框架对request body处理有问题，导致异常退出。
>
>通过这个例子，想要探索下net/http一次完整http请求的全过程。我们将通过dlv进行一步步的调试。注意多个goroutines同时存在时候的调试。

## 1 源程序，golang版本(go version go1.9.2 darwin/amd64)

```golang
package main

import (
	"io"
	"log"
	"net/http"
)

func main() {
	// Hello world, the web server
	helloHandler := func(w http.ResponseWriter, req *http.Request) {
		req.ParseForm()
		api := req.Form.Get("a")
		io.WriteString(w, "Hello, world!"+api)
	}

	http.HandleFunc("/hello", helloHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

## 2 调试详细步骤
### 2.1 入口断点，进入http.ListenAndServe内部
```shell
➜ dlv debug hello.go
Type 'help' for list of commands.
(dlv) b main.main
Breakpoint 1 set at 0x12df728 for main.main() ./hello.go:9
(dlv) c
> main.main() ./hello.go:9 (hits goroutine(1):1 total:1) (PC: 0x12df728)
=>   9:	func main() {
    10:		// Hello world, the web server
    11:		helloHandler := func(w http.ResponseWriter, req *http.Request) {
    12:			req.ParseForm()
    13:			api := req.Form.Get("a")
    14:			io.WriteString(w, "Hello, world!"+api)
...

(dlv) n
> main.main() ./hello.go:18 (PC: 0x12df775)
=>  18:		log.Fatal(http.ListenAndServe(":8080", nil))
    19:	}
(dlv) s
> net/http.ListenAndServe() /Users/xiezhen/Documents/install/go/src/net/http/server.go:2880 (PC: 0x12af063)
  2879:	// ListenAndServe always returns a non-nil error.
=>2880:	func ListenAndServe(addr string, handler Handler) error {
  2881:		server := &Server{Addr: addr, Handler: handler}
  2882:		return server.ListenAndServe()
  2883:	}
  2884:
  2885:	// ListenAndServeTLS acts identically to ListenAndServe, except that it
```

### 2.2 net/http开始监听端口
>监听tcp

```shell
(dlv) s
> net/http.(*Server).ListenAndServe() /Users/xiezhen/Documents/install/go/src/net/http/server.go:2627 (PC: 0x12adfd8)
  2622:	// ListenAndServe listens on the TCP network address srv.Addr and then
  2623:	// calls Serve to handle requests on incoming connections.
  2624:	// Accepted connections are configured to enable TCP keep-alives.
  2625:	// If srv.Addr is blank, ":http" is used.
  2626:	// ListenAndServe always returns a non-nil error.
=>2627:	func (srv *Server) ListenAndServe() error {
  2628:		addr := srv.Addr
  2629:		if addr == "" {
  2630:			addr = ":http"
  2631:		}
  2632:		ln, err := net.Listen("tcp", addr)
(dlv) n
> net/http.(*Server).ListenAndServe() /Users/xiezhen/Documents/install/go/src/net/http/server.go:2632 (PC: 0x12ae043)
  2627:	func (srv *Server) ListenAndServe() error {
  2628:		addr := srv.Addr
  2629:		if addr == "" {
  2630:			addr = ":http"
  2631:		}
=>2632:		ln, err := net.Listen("tcp", addr)
  2633:		if err != nil {
  2634:			return err
  2635:		}
  2636:		return srv.Serve(tcpKeepAliveListener{ln.(*net.TCPListener)})
  2637:	}
```
### 2.3 等待被连接
>进入到处理连接的位置


```shell
> net/http.(*Server).Serve() /Users/xiezhen/Documents/install/go/src/net/http/server.go:2680 (PC: 0x12ae330)
  2675:	//
  2676:	// Serve always returns a non-nil error. After Shutdown or Close, the
  2677:	// returned error is ErrServerClosed.
  2678:	func (srv *Server) Serve(l net.Listener) error {
  2679:		defer l.Close()
=>2680:		if fn := testHookServerServe; fn != nil {
  2681:			fn(srv, l)
  2682:		}
  2683:		var tempDelay time.Duration // how long to sleep on accept failure
  2684:
  2685:		if err := srv.setupHTTP2_Serve(); err != nil {
```

>补充这个函数的具体代码实现

```golang
func (srv *Server) Serve(l net.Listener) error {
	...
	for {
	    //接受请求
		rw, e := l.Accept()
		...
		// 协程处理
		go c.serve(ctx)
	}
}
```

### 2.4 处理请求

>，当我们执行`rw, e := l.Accept()`时候，server程序会卡住，这是因为正在等待连接过来，需要另开一个shell，发起`curl -v "http://127.0.0.1:8080/hello" -X POST -d "a=1" -H "Content-Type:application/x-www-form-urlencoded"`请求，client请求会卡住，但是server程序会继续执行

```
(dlv) n
> net/http.(*Server).Serve() /Users/xiezhen/Documents/install/go/src/net/http/server.go:2696 (PC: 0x12ae5ab)
  2691:
  2692:		baseCtx := context.Background() // base is always background, per Issue 16220
  2693:		ctx := context.WithValue(baseCtx, ServerContextKey, srv)
  2694:		for {
  2695:			rw, e := l.Accept()
=>2696:			if e != nil {
...
...
(dlv) n
> net/http.(*Server).Serve() /Users/xiezhen/Documents/install/go/src/net/http/server.go:2720 (PC: 0x12ae9a8)
  2715:				return e
  2716:			}
  2717:			tempDelay = 0
  2718:			c := srv.newConn(rw)
  2719:			c.setState(c.rwc, StateNew) // before Serve can return
=>2720:			go c.serve(ctx)
  2721:		}
```

>这个时候注意一点，在执行2720行代码之前，我们先要在c.serve(ctx)函数的第一行加上一个断点，这样方便我们对多goroutine进行调试

```
(dlv) b /Users/xiezhen/Documents/install/go/src/net/http/server.go:1691
```

>执行2720之前，查看当前goroutines

```
(dlv) goroutines
* Goroutine 1 - User: /Users/xiezhen/Documents/install/go/src/net/http/server.go:2694 net/http.(*Server).Serve (0x12ae9e4) (thread 79452)
  Goroutine 2 - User: /Users/xiezhen/Documents/install/go/src/runtime/proc.go:288 runtime.gopark (0x1030d1d)
  Goroutine 3 - User: /Users/xiezhen/Documents/install/go/src/runtime/proc.go:288 runtime.gopark (0x1030d1d)
  Goroutine 4 - User: /Users/xiezhen/Documents/install/go/src/runtime/proc.go:288 runtime.gopark (0x1030d1d)
[3 goroutines]
```

>执行2720后，查看当前goroutines，发现多了一个新goroutine，切换到请求处理的goroutine上

```
(dlv) goroutines
* Goroutine 1 - User: /Users/xiezhen/Documents/install/go/src/net/http/server.go:2694 net/http.(*Server).Serve (0x12ae9e4) (thread 79452)
  Goroutine 2 - User: /Users/xiezhen/Documents/install/go/src/runtime/proc.go:288 runtime.gopark (0x1030d1d)
  Goroutine 3 - User: /Users/xiezhen/Documents/install/go/src/runtime/proc.go:288 runtime.gopark (0x1030d1d)
  Goroutine 4 - User: /Users/xiezhen/Documents/install/go/src/runtime/proc.go:288 runtime.gopark (0x1030d1d)
  Goroutine 5 - User: /Users/xiezhen/Documents/install/go/src/net/http/server.go:1690 net/http.(*conn).serve (0x12a89c0)
[5 goroutines]
(dlv) goroutine 5
Switched from 1 to 5 (thread 79452)
(dlv) ls
> net/http.(*conn).serve() /Users/xiezhen/Documents/install/go/src/net/http/server.go:1690 (PC: 0x12a89c0)
  1685:		}
  1686:		return false
  1687:	}
  1688:
  1689:	// Serve a new connection.
=>1690:	func (c *conn) serve(ctx context.Context) {
  1691:		c.remoteAddr = c.rwc.RemoteAddr().String()
  1692:		ctx = context.WithValue(ctx, LocalAddrContextKey, c.rwc.LocalAddr())
  1693:		defer func() {
  1694:			if err := recover(); err != nil && err != ErrAbortHandler {
  1695:				const size = 64 << 10
```

### 2.5 每个请求单独的goroutine大致流程

>`func (c *conn) serve(ctx context.Context)`函数过长，我们只挑主要流程来看

>`c.readRequest(ctx)`读取请求，将所有内容都存在w上，见 2.5.1
>`serverHandler{c.server}.ServeHTTP(w, w.req)`这个最终会走到我们实现的ServeHTTP方法，见 2.5.2
>`w.finishRequest()`请求结束，见 2.5.3 
```golang
// Serve a new connection.
func (c *conn) serve(ctx context.Context) {
    ....
	// HTTP/1.x from here on.
	ctx, cancelCtx := context.WithCancel(ctx)
	c.cancelCtx = cancelCtx
	defer cancelCtx()

	c.r = &connReader{conn: c}
	c.bufr = newBufioReader(c.r)
	c.bufw = newBufioWriterSize(checkConnErrorWriter{c}, 4<<10)

	for {
        // 1 读取请求，header信息均在此处处理完毕
		w, err := c.readRequest(ctx)
        ...

		// 2 具体的业务逻辑处理
		serverHandler{c.server}.ServeHTTP(w, w.req)
		w.cancelCtx()
		if c.hijacked() {
			return
		}
		// 3 请求结束
		w.finishRequest()
        ....
	}
}
```

#### 2.5.1 请求处理
>进入`func (c *conn) readRequest(ctx context.Context) (w *response, err error)`，然后再进入`/Users/xiezhen/Documents/install/go/src/net/http/request.go`文件的`func readRequest(b *bufio.Reader, deleteHostHeader bool) (req *Request, err error)`函数

>这个过程中，我们可以打印出请求的头部信息

```
(dlv) n
> net/http.(*conn).serve() /Users/xiezhen/Documents/install/go/src/net/http/server.go:1739 (PC: 0x12a9472)
...
=>1739:			w, err := c.readRequest(ctx)
  1740:			if c.r.remain != c.server.initialReadLimitSize() {
  1741:				// If we read any bytes off the wire, we're active.
  1742:				c.setState(c.rwc, StateActive)
  1743:			}
  1744:			if err != nil {
(dlv) s
> net/http.(*conn).readRequest() /Users/xiezhen/Documents/install/go/src/net/http/server.go:904 (PC: 0x12a2d7b)
=> 904:	func (c *conn) readRequest(ctx context.Context) (w *response, err error) {
   905:		if c.hijacked() {
   906:			return nil, ErrHijacked
   907:		}
   908:
   909:		var (
...
(dlv) n
> net/http.(*conn).readRequest() /Users/xiezhen/Documents/install/go/src/net/http/server.go:933 (PC: 0x12a3231)
=> 933:		req, err := readRequest(c.bufr, keepHostHeader)
   934:		if err != nil {
   935:			if c.r.hitReadLimit() {
   936:				return nil, errTooLarge
   937:			}
   938:			return nil, err
(dlv) s
> net/http.readRequest() /Users/xiezhen/Documents/install/go/src/net/http/request.go:919 
=> 919:	func readRequest(b *bufio.Reader, deleteHostHeader bool) (req *Request, err error) {
   920:		tp := newTextprotoReader(b)
   921:		req = new(Request)
   922:
   923:		// First line: GET /index.html HTTP/1.0
   924:		var s string
...
(dlv) n
> net/http.readRequest() /Users/xiezhen/Documents/install/go/src/net/http/request.go:925 (PC: 0x129af96)
   923:		// First line: GET /index.html HTTP/1.0
   924:		var s string
=> 925:		if s, err = tp.ReadLine(); err != nil {
   926:			return nil, err
   927:		}
(dlv) p s
"POST /hello HTTP/1.1"
...
(dlv) n
> net/http.readRequest() /Users/xiezhen/Documents/install/go/src/net/http/request.go:973 (PC: 0x129b6a9)
   971:		// Subsequent lines: Key: value.
   972:		mimeHeader, err := tp.ReadMIMEHeader()
=> 973:		if err != nil {
   974:			return nil, err
   975:		}
(dlv) p mimeHeader
net/textproto.MIMEHeader [
	"Host": [
		"127.0.0.1:8080",
	],
	"User-Agent": [
		"curl/7.43.0",
	],
	"Accept": ["*/*"],
	"Content-Type": [
		"application/x-www-form-urlencoded",
	],
	"Content-Length": ["3"],
]
```

#### 2.5.2 ServeHTTP

>这一部分是请求->路由->具体处理函数的过程

```
(dlv) n
> net/http.(*conn).serve() /Users/xiezhen/Documents/install/go/src/net/http/server.go:1801 (PC: 0x12a9baf)
=>1801:			serverHandler{c.server}.ServeHTTP(w, w.req)
  1802:			w.cancelCtx()
  1803:			if c.hijacked() {
  1804:				return
  1805:			}
  1806:			w.finishRequest()
(dlv) s
> net/http.serverHandler.ServeHTTP() /Users/xiezhen/Documents/install/go/src/net/http/server.go:2611 (PC: 0x12add93)
=>2611:	func (sh serverHandler) ServeHTTP(rw ResponseWriter, req *Request) {
  2612:		handler := sh.srv.Handler
  2613:		if handler == nil {
  2614:			handler = DefaultServeMux
  2615:		}
  2616:		if req.RequestURI == "*" && req.Method == "OPTIONS" {
```

>handler为空，说明走默认的DefaultServeMux，而在我们得程序中`http.HandleFunc("/hello", helloHandler)`这句已经注册了路由，可以通过打印DefaultServeMux来进行验证，可以明显看到DefaultServeMux.m中含有hello的路由信息

```
(dlv) p DefaultServeMux
*net/http.ServeMux {
	mu: sync.RWMutex {
		w: (*sync.Mutex)(0x14d89a0),
		writerSem: 0,
		readerSem: 0,
		readerCount: 0,
		readerWait: 0,},
	m: map[string]net/http.muxEntry [
		"/hello": (*net/http.muxEntry)(0xc4200a4268),
	],
	hosts: false,}
```

>继续往下走

```
(dlv) n
> net/http.serverHandler.ServeHTTP() /Users/xiezhen/Documents/install/go/src/net/http/server.go:2619 (PC: 0x12adf30)
=>2619:		handler.ServeHTTP(rw, req)
  2620:	}
(dlv) s
> net/http.(*ServeMux).ServeHTTP() /Users/xiezhen/Documents/install/go/src/net/http/server.go:2245
=>2245:	func (mux *ServeMux) ServeHTTP(w ResponseWriter, r *Request) {
  2246:		if r.RequestURI == "*" {
  2247:			if r.ProtoAtLeast(1, 1) {
  2248:				w.Header().Set("Connection", "close")
  2249:			}
  2250:			w.WriteHeader(StatusBadRequest)
```

> 可以看到这个函数的具体实现，h, _ := mux.Handler(r)将会取到具体的函数，然后执行server

```golang
// ServeHTTP dispatches the request to the handler whose
// pattern most closely matches the request URL.
func (mux *ServeMux) ServeHTTP(w ResponseWriter, r *Request) {
	if r.RequestURI == "*" {
		if r.ProtoAtLeast(1, 1) {
			w.Header().Set("Connection", "close")
		}
		w.WriteHeader(StatusBadRequest)
		return
	}
	h, _ := mux.Handler(r)
	h.ServeHTTP(w, r)
}

func (mux *ServeMux) Handler(r *Request) (h Handler, pattern string) {
    ...
}
```

```
(dlv) n
> net/http.(*ServeMux).ServeHTTP() /Users/xiezhen/Documents/install/go/src/net/http/server.go:2254 (PC: 0x12acc3f)
  2249:			}
  2250:			w.WriteHeader(StatusBadRequest)
  2251:			return
  2252:		}
  2253:		h, _ := mux.Handler(r)
=>2254:		h.ServeHTTP(w, r)
  2255:	}
  2256:
  2257:	// Handle registers the handler for the given pattern.
  2258:	// If a handler already exists for pattern, Handle panics.
  2259:	func (mux *ServeMux) Handle(pattern string, handler Handler) {
(dlv) s
> net/http.HandlerFunc.ServeHTTP() /Users/xiezhen/Documents/install/go/src/net/http/server.go:1917 (PC: 0x12aa9df)
  1912:	// with the appropriate signature, HandlerFunc(f) is a
  1913:	// Handler that calls f.
  1914:	type HandlerFunc func(ResponseWriter, *Request)
  1915:
  1916:	// ServeHTTP calls f(w, r).
=>1917:	func (f HandlerFunc) ServeHTTP(w ResponseWriter, r *Request) {
  1918:		f(w, r)
  1919:	}
  1920:
  1921:	// Helper handlers
  1922:
(dlv) s
> net/http.HandlerFunc.ServeHTTP() /Users/xiezhen/Documents/install/go/src/net/http/server.go:1918 (PC: 0x12aa9ed)
  1913:	// Handler that calls f.
  1914:	type HandlerFunc func(ResponseWriter, *Request)
  1915:
  1916:	// ServeHTTP calls f(w, r).
  1917:	func (f HandlerFunc) ServeHTTP(w ResponseWriter, r *Request) {
=>1918:		f(w, r)
  1919:	}
  1920:
  1921:	// Helper handlers
  1922:
  1923:	// Error replies to the request with the specified error message and HTTP code.
(dlv) s
> main.main.func1() ./hello.go:11 (PC: 0x12df883)
     6:		"net/http"
     7:	)
     8:
     9:	func main() {
    10:		// Hello world, the web server
=>  11:		helloHandler := func(w http.ResponseWriter, req *http.Request) {
    12:			req.ParseForm()
    13:			api := req.Form.Get("a")
    14:			io.WriteString(w, "Hello, world!"+api)
    15:		}
```

#### 2.5.3 结束请求

```golang
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

>w.conn.bufw.Flush()结果写回到client

```
(dlv) s
> bufio.(*Writer).Flush() /Users/xiezhen/Documents/install/go/src/bufio/bufio.go:560 (PC: 0x112315b)
   555:		b.n = 0
   556:		b.wr = w
   557:	}
   558:
   559:	// Flush writes any buffered data to the underlying io.Writer.
=> 560:	func (b *Writer) Flush() error {
   561:		if b.err != nil {
   562:			return b.err
   563:		}
   564:		if b.n == 0 {
   565:			return nil
```

```goalng
// Flush writes any buffered data to the underlying io.Writer.
func (b *Writer) Flush() error {
	if b.err != nil {
		return b.err
	}
	if b.n == 0 {
		return nil
	}
	n, err := b.wr.Write(b.buf[0:b.n])
	if n < b.n && err == nil {
		err = io.ErrShortWrite
	}
	if err != nil {
		if n > 0 && n < b.n {
			copy(b.buf[0:b.n-n], b.buf[n:b.n])
		}
		b.n -= n
		b.err = err
		return err
	}
	b.n = 0
	return nil
}
```

```
(dlv) ls
> bufio.(*Writer).Flush() /Users/xiezhen/Documents/install/go/src/bufio/bufio.go:568 (PC: 0x11232fb)
   563:		}
   564:		if b.n == 0 {
   565:			return nil
   566:		}
   567:		n, err := b.wr.Write(b.buf[0:b.n])
=> 568:		if n < b.n && err == nil {
   569:			err = io.ErrShortWrite
   570:		}
   571:		if err != nil {
   572:			if n > 0 && n < b.n {
   573:				copy(b.buf[0:b.n-n], b.buf[n:b.n])
(dlv) p string(b.buf)
"HTTP/1.1 200 OK\r\nDate: Fri, 17 May 2019 06:58:37 GMT\r\nContent-Le"
(dlv) p string(b.buf[64:])
"ngth: 14\r\nContent-Type: text/plain; charset=utf-8\r\n\r\nHello, worl"
(dlv) p string(b.buf[128:])
"d!1\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"
```
