# golang delve调试
>delve 是golang的调试器
>
>install `go get github.com/derekparker/delve/cmd/dlv`

## 1 示例程序

```golang
package main

import (
	"io"
	"log"
	"os"
	"strings"
	"sync"
)

func testGoroutine() {
	log.Println("i am in test ")
}

func main() {
	r := strings.NewReader("some io.Reader stream to be read\n")

	if n, err := io.Copy(os.Stdout, r); err != nil {
		log.Fatal(err)
	} else {
		log.Println(n, 1)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		testGoroutine()
		defer wg.Done()
	}()

	wg.Wait()
}
```

## 2 调试过程
>编译`go build -gcflags "-N -l" -o test hello.go`
>
>1 直接调试源码`dlv debug hello.go`
>
>2 调试二进制`dlv exec ./test`
>
>此处以调试源代码为例

```
dlv debug hello.go
```

>1 在main函数处设置断点

```
(dlv) b main.main
Breakpoint 1 set at 0x109699b for main.main() ./hello.go:15
```

>2 运行至断点处

```
(dlv) c
> main.main() ./hello.go:15 (hits goroutine(1):1 total:1) (PC: 0x109699b)
    14:
=>  15:	func main() {
    16:		r := strings.NewReader("some io.Reader stream to be read\n")
    17:
    18:		if n, err := io.Copy(os.Stdout, r); err != nil {
    19:			log.Fatal(err)
    20:		} else {
(dlv)
```

>3 通过行号设置断点

```
(dlv) b hello.go:16
Breakpoint 2 set at 0x10969b9 for main.main() ./hello.go:16
```

>4 显示所有断点

```
(dlv) bp
Breakpoint 1 at 0x109699b for main.main() ./hello.go:15 (1)
Breakpoint 2 at 0x10969b9 for main.main() ./hello.go:16 (0)
```

>5 删除某个断点

```
(dlv) clear 1
Breakpoint 1 cleared at 0x109699b for main.main() ./hello.go:15
(dlv) bp
Breakpoint 2 at 0x10969b9 for main.main() ./hello.go:16 (0)
```

> 6 显示当前代码运行位置

```
(dlv) ls
> main.main() ./hello.go:15 (hits goroutine(1):1 total:1) (PC: 0x109699b)
    14:
=>  15:	func main() {
    16:		r := strings.NewReader("some io.Reader stream to be read\n")
    17:
    18:		if n, err := io.Copy(os.Stdout, r); err != nil {
    19:			log.Fatal(err)
    20:		} else {
```

> 7 查看当前调用栈信息

```
(dlv) bt
0  0x000000000109699b in main.main
   at ./hello.go:15
1  0x0000000001028af6 in runtime.main
   at /Users/xiezhen/Documents/install/go/src/runtime/proc.go:195
2  0x00000000010508d1 in runtime.goexit
   at /Users/xiezhen/Documents/install/go/src/runtime/asm_amd64.s:2337
```

> 8 执行下一步

```
(dlv) n
> main.main() ./hello.go:16 (hits goroutine(1):1 total:1) (PC: 0x10969b9)
    15:	func main() {
=>  16:		r := strings.NewReader("some io.Reader stream to be read\n")
    17:
    18:		if n, err := io.Copy(os.Stdout, r); err != nil {
    19:			log.Fatal(err)
    20:		} else {
    21:			log.Println(n, 1)
(dlv) n
> main.main() ./hello.go:18 (PC: 0x10969d5)
    13:	}
    14:
    15:	func main() {
    16:		r := strings.NewReader("some io.Reader stream to be read\n")
    17:
=>  18:		if n, err := io.Copy(os.Stdout, r); err != nil {
    19:			log.Fatal(err)
    20:		} else {
    21:			log.Println(n, 1)
    22:		}
    23:
```

> 9 输出变量信息

```
(dlv) p r
*strings.Reader {
	s: "some io.Reader stream to be read\n",
	i: 0,
	prevRune: -1,}
```

> 10 进入函数

```
(dlv) n
> main.main() ./hello.go:18 (PC: 0x10b71e5)
    13:	}
    14:
    15:	func main() {
    16:		r := strings.NewReader("some io.Reader stream to be read\n")
    17:
=>  18:		if n, err := io.Copy(os.Stdout, r); err != nil {
    19:			log.Fatal(err)
    20:		} else {
    21:			log.Println(n, 1)
    22:		}
    23:
(dlv) s
> io.Copy() /Users/xiezhen/Documents/install/go/src/io/io.go:361 (PC: 0x105f028)
   356:	//
   357:	// If src implements the WriterTo interface,
   358:	// the copy is implemented by calling src.WriteTo(dst).
   359:	// Otherwise, if dst implements the ReaderFrom interface,
   360:	// the copy is implemented by calling dst.ReadFrom(src).
=> 361:	func Copy(dst Writer, src Reader) (written int64, err error) {
   362:		return copyBuffer(dst, src, nil)
   363:	}
   364:
   365:	// CopyBuffer is identical to Copy except that it stages through the
   366:	// provided buffer (if one is required) rather than allocating a
(dlv) s
> io.Copy() /Users/xiezhen/Documents/install/go/src/io/io.go:362 (PC: 0x105f063)
   357:	// If src implements the WriterTo interface,
   358:	// the copy is implemented by calling src.WriteTo(dst).
   359:	// Otherwise, if dst implements the ReaderFrom interface,
   360:	// the copy is implemented by calling dst.ReadFrom(src).
   361:	func Copy(dst Writer, src Reader) (written int64, err error) {
=> 362:		return copyBuffer(dst, src, nil)
   363:	}
   364:
   365:	// CopyBuffer is identical to Copy except that it stages through the
   366:	// provided buffer (if one is required) rather than allocating a
   367:	// temporary one. If buf is nil, one is allocated; otherwise if it has
(dlv) s
> io.copyBuffer() /Users/xiezhen/Documents/install/go/src/io/io.go:378 (PC: 0x105f16b)
   373:		return copyBuffer(dst, src, buf)
   374:	}
   375:
   376:	// copyBuffer is the actual implementation of Copy and CopyBuffer.
   377:	// if buf is nil, one is allocated.
=> 378:	func copyBuffer(dst Writer, src Reader, buf []byte) (written int64, err error) {
   379:		// If the reader has a WriteTo method, use it to do the copy.
   380:		// Avoids an allocation and a copy.
   381:		if wt, ok := src.(WriterTo); ok {
   382:			return wt.WriteTo(dst)
   383:		}
```

> 11 在第n层调用栈上执行指令

```
(dlv) bt
0  0x000000000105f16b in io.copyBuffer
   at /Users/xiezhen/Documents/install/go/src/io/io.go:378
1  0x000000000105f0c8 in io.Copy
   at /Users/xiezhen/Documents/install/go/src/io/io.go:362
2  0x00000000010b7234 in main.main
   at ./hello.go:18
3  0x000000000102c2f4 in runtime.main
   at /Users/xiezhen/Documents/install/go/src/runtime/proc.go:195
4  0x00000000010559d1 in runtime.goexit
   at /Users/xiezhen/Documents/install/go/src/runtime/asm_amd64.s:2337
(dlv) frame 2 ls
Goroutine 1 frame 2 at /Users/xiezhen/Documents/codes/golib/src/myhello/hello.go:18 (PC: 0x10b7234)
    13:	}
    14:
    15:	func main() {
    16:		r := strings.NewReader("some io.Reader stream to be read\n")
    17:
=>  18:		if n, err := io.Copy(os.Stdout, r); err != nil {
    19:			log.Fatal(err)
    20:		} else {
    21:			log.Println(n, 1)
    22:		}
    23:
```

> 12 查看goroutines，可以看到，最后一个routine是我们启动的

```
(dlv) n
> main.main() ./hello.go:33 (PC: 0x10b73bc)
    28:		go func() {
    29:			testGoroutine()
    30:			defer wg.Done()
    31:		}()
    32:
=>  33:		wg.Wait()
    34:	}
(dlv) goroutines
* Goroutine 1 - User: ./hello.go:33 main.main (0x10b73bc) (thread 442073)
  Goroutine 2 - User: /Users/xiezhen/Documents/install/go/src/runtime/proc.go:288 runtime.gopark (0x102c74d)
  Goroutine 3 - User: /Users/xiezhen/Documents/install/go/src/runtime/proc.go:288 runtime.gopark (0x102c74d)
  Goroutine 4 - User: /Users/xiezhen/Documents/install/go/src/runtime/proc.go:288 runtime.gopark (0x102c74d)
  Goroutine 5 - User: ./hello.go:28 main.main.func1 (0x10b7530)
[5 goroutines]
```

> 13 switch到想要查看的goroutine，可以继续执行

```
(dlv) goroutine 5
Switched from 1 to 5 (thread 442073)
(dlv) ls
> main.main.func1() ./hello.go:28 (PC: 0x10b7530)
    23:			log.Println(n, 1)
    24:		}
    25:
    26:		var wg sync.WaitGroup
    27:		wg.Add(1)
=>  28:		go func() {
    29:			testGoroutine()
    30:			defer wg.Done()
    31:		}()
    32:
    33:		wg.Wait()
(dlv) n
> main.main.func1() ./hello.go:29 (PC: 0x10b754d)
    24:		}
    25:
    26:		var wg sync.WaitGroup
    27:		wg.Add(1)
    28:		go func() {
=>  29:			testGoroutine()
    30:			defer wg.Done()
    31:		}()
    32:
    33:		wg.Wait()
```

> 14 查看goroutine信息

```
(dlv) goroutine
Thread 442283 at ./hello.go:29
Goroutine 5:
	Runtime: ./hello.go:29 main.main.func1 (0x10b754d)
	User: ./hello.go:29 main.main.func1 (0x10b754d)
	Go: ./hello.go:28 main.main (0x10b73bc)
	Start: ./hello.go:28 main.main.func1 (0x10b7530)
```

> 15 重启

```
(dlv) r
Process restarted with PID 21142
```

> 16 反汇编，可以看到0x12345678对应的值

```
> main.main() ./hello.go:17 (hits goroutine(1):1 total:1) (PC: 0x10b71ab)
    12:		for {
    13:			log.Println("i am in test ")
    14:		}
    15:	}
    16:
=>  17:	func main() {
    18:		var x int32
    19:		x++
    20:		x = 32
    21:
    22:		x = 0x12345678
(dlv) disassemble
TEXT main.main(SB) /Users/xiezhen/Documents/codes/golib/src/myhello/hello.go
	hello.go:17	0x10b7190	65488b0c25a0080000		mov rcx, qword ptr gs:[0x8a0]
	hello.go:17	0x10b7199	488d842450ffffff		lea rax, ptr [rsp+0xffffff50]
	hello.go:17	0x10b71a1	483b4110			cmp rax, qword ptr [rcx+0x10]
	hello.go:17	0x10b71a5	0f86a9030000			jbe 0x10b7554
=>	hello.go:17	0x10b71ab*	4881ec30010000			sub rsp, 0x130
	hello.go:17	0x10b71b2	4889ac2428010000		mov qword ptr [rsp+0x128], rbp
	hello.go:17	0x10b71ba	488dac2428010000		lea rbp, ptr [rsp+0x128]
	hello.go:18	0x10b71c2	c744243800000000		mov dword ptr [rsp+0x38], 0x0
	hello.go:19	0x10b71ca	c744243c00000000		mov dword ptr [rsp+0x3c], 0x0
	hello.go:19	0x10b71d2	c744243801000000		mov dword ptr [rsp+0x38], 0x1
	hello.go:20	0x10b71da	c744243820000000		mov dword ptr [rsp+0x38], 0x20
	hello.go:22	0x10b71e2	c744243878563412		mov dword ptr [rsp+0x38], 0x12345678
```

> 17 查看包级变量

```
(dlv) vars main
main.statictmp_0 = "i am in test "
main.statictmp_1 = 1
main.initdone· = 2
runtime.main_init_done = chan bool 0/0
runtime.mainStarted = true
```

> 18 函数参数和变量,进入函数之后可以通过args和locals命令查看函数的参数和局部变量：

```
(dlv) args
(no args)
(dlv) locals
x = 0
r = *strings.Reader nil
wg = sync.WaitGroup {noCopy: sync.noCopy {}, state1: [12]uint8 [...], sema: 0}
```