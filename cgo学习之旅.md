# cgo学习之旅

## 1 背景
C/C++语言是经典的编程语言，golang相对于这些经典语言来说，是新生代的翘楚。

C/C++语言的很多开源库都是历经考验，性能优化到极致，所以go需要把这些利用起来。

CGO就是这么一个工具，CGO能够讲积累的C/C++软件资产利用起来。

在实际的开发工作中，有两次遇到go模块需要调用C服务，苦于自己技术有限，所以没能自己搞上，现在回顾一下，把
自己的这块短板补上。

## 2 go调用c

### 2.1 go调用c源码

#### 2.2.1 go调用C的函数
> C语言和Golang语言在相同文件中

```golang
package main

/*
#include<stdio.h>

static void SayHello(const char* s) {
	puts(s);
}
*/
import "C"

func main() {
	println("hello cgo")
	C.puts(C.CString("hello world \n"))
	C.SayHello(C.CString("hello world say hello\n"))
}
```
```
注释中写C的函数，golang中直接调用，二者在相同文件中
```

>C语言和Golang语言处于不同的文件中

```golang
// test.go
package main

// void SayHello(const char* s);
import "C"

func main() {
	C.SayHello(C.CString("Hello, World\n"))
}
```
```C
// hello.c
#include<stdio.h>

void SayHello(const char* s) {
	puts(s);
}
```

```
执行时候，需要 go build .
```

#### 2.2.2 C调用golang的函数


### 2.2 go调用c静态库

### 2.3 go调用c动态库

### 3 go导出C动态接口



