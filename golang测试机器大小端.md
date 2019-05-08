# 机器大小端

## 原理

>大端模式，是指数据的高位，保存在内存的低地址中，而数据的低位，保存在内存的高地址中；
>
>小端模式，是指数据的高位，保存在内存的高地址中，而数据的低位，保存在内存的低地址中；
>
>如下图所示

![大小端说明](https://github.com/xiezhenouc/golanglearn/blob/master/%E5%9B%BE%E7%89%87%E8%AF%B4%E6%98%8E/%E5%A4%A7%E5%B0%8F%E7%AB%AF.png)


## 分析
>可以通过很多方式查看机器的大小端
>写一个代码，测试下



## 1 查看系统文件

## 2 代码测试
>c中可以用union共享内存的方式来进行实验

```golang
package main

import (
	"fmt"
	"unsafe"
)

func main() {
	// 4个字节
	var num int32
	num = 0x12345678

	// 1个字节
	var p1 *byte
	p1 = (*byte)(unsafe.Pointer(&num))

	fmt.Printf("%x\n", *p1)

	// 2个字节
	var p2 *int16
	p2 = (*int16)(unsafe.Pointer(&num))

	fmt.Printf("%x\n", *p2)

	if *p1 == 0x78 && *p2 == 0x5678 {
		fmt.Println("little endian")
	} else {
		fmt.Println("big endian")
	}
}

```

## 3 GDB调试
>可以通过GDB调试进行查看，明显可以看到内存里面的数据是如何分布的
>
> b n设置断点 break n；r开始执行 run；n执行下一步 next；p打印 print
>
>x 是GDB中用来检查内存的命令，其使用方法是：
>
> x/nfu address
>
>其中 n 表示要重复打印的次数，默认值为1;*f* 表示输出的格式，支持 x(十六进制)、d(十进制)、 u(无符号整型)、 o(八进制)、t(二进制)、a(地址值)、c(字符型)、f(浮点型)、s(字符串)这几种格式，默认使用 x ;u表示每个输出的宽度，可以选择b(1字节)、h(2字节)、w(4字节)和g(8字节)，默认为4字节(w)。
>
>这里，我们用x/xb，十六进制，单个字节打印


```
//防止编译器优化
>go build -gcflags "-N -l" -o test hello.go
>gdb ./test
(gdb) layout src
(gdb) list
//在p1 p2均赋值后位置设置下断点
(gdb) b 19
(gdb) print &num
$2 = (int32 *) 0xc420041eec
(gdb) print p1
$3 = (uint8 *) 0xc420041eec "xV4\022\354\036\004", <incomplete sequence \304>
(gdb) print p2
$4 = (int16 *) 0xc420041eec

(gdb) x/xb 0xc420041eec
0xc420041eec:   0x78
(gdb) x/xb 0xc420041eed
0xc420041eed:   0x56
(gdb) x/xb 0xc420041eee
0xc420041eee:   0x34
(gdb) x/xb 0xc420041eef
0xc420041eef:   0x12
```

![内存布局](https://github.com/xiezhenouc/golanglearn/blob/master/%E5%9B%BE%E7%89%87%E8%AF%B4%E6%98%8E/%E5%86%85%E5%AD%98%E5%B8%83%E5%B1%80.png)

## 4 命令总结
>hexdump命令，查看查看“二进制”文件的十六进制编码，使用

```
-C：输出规范的十六进制和ASCII码

$ echo abcd > tmp
$ hexdump -C tmp
00000000  61 62 63 64 0a                                    |abcd.|
00000005 
```

>od命令，查看文件内容

```
使用单字节八进制解释进行输出
$ od -b tmp
0000000 141 142 143 144 012
0000005
使用ASCII码进行输出，注意其中包括转义字符
$ od -c tmp
0000000   a   b   c   d  \n
0000005
```
