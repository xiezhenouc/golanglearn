# golang语言与汇编
>背景 最近在学习golang源码相关的东西，发现如果只是看golang的代码，只能是明白这个关键字能够做什么事，但是实际上是怎么实现，自己还是不了解，基于此，想要好好学习一下golang的汇编。
>
>前面，我写了一篇c语言的汇编学习，首先，先熟悉下对应的汇编概念。接下来我们来看下golang的汇编有何特殊之处
>
>golang的汇编叫plan 9汇编
>
>本文参考资料 
>
>https://mp.weixin.qq.com/s/obnnVkO2EiFnuXk_AIDHWw
>
>http://xargin.com/go-and-plan9-asm/

## 1 golang源代码

```golang
package main

//go:noinline
func add(a, b int32) (int32, bool, int64) {
	return a + b, true, 1
}

func main() {
	add(10, 32)
}
```

>分析，我们通过查看该源码对应的汇编，以及我们通过对源程序编译后的可执行文件反汇编对应着来看

### 1.1 编译，GDB调试

```
go build -gcflags "-N -l" hello.go
gdb ./hello
b main.main
b main.add
r
c

```

### 1.2 反汇编

>main函数

```
Dump of assembler code for function main.main:
=> 0x0000000000450a90 <+0>:	mov    %fs:0xfffffffffffffff8,%rcx
   0x0000000000450a99 <+9>:	cmp    0x10(%rcx),%rsp
   0x0000000000450a9d <+13>:	jbe    0x450aca <main.main+58>
   0x0000000000450a9f <+15>:	sub    $0x20,%rsp
   0x0000000000450aa3 <+19>:	mov    %rbp,0x18(%rsp)
   0x0000000000450aa8 <+24>:	lea    0x18(%rsp),%rbp
   0x0000000000450aad <+29>:	movabs $0x200000000a,%rax
   0x0000000000450ab7 <+39>:	mov    %rax,(%rsp)
   0x0000000000450abb <+43>:	callq  0x450a50 <main.add>
   0x0000000000450ac0 <+48>:	mov    0x18(%rsp),%rbp
   0x0000000000450ac5 <+53>:	add    $0x20,%rsp
   0x0000000000450ac9 <+57>:	retq
   0x0000000000450aca <+58>:	callq  0x448830 <runtime.morestack_noctxt>
   0x0000000000450acf <+63>:	jmp    0x450a90 <main.main>
```

>add函数

```
Dump of assembler code for function main.add:
=> 0x0000000000450a50 <+0>:	movl   $0x0,0x10(%rsp)
   0x0000000000450a58 <+8>:	movb   $0x0,0x14(%rsp)
   0x0000000000450a5d <+13>:	movq   $0x0,0x18(%rsp)
   0x0000000000450a66 <+22>:	mov    0x8(%rsp),%eax
   0x0000000000450a6a <+26>:	add    0xc(%rsp),%eax
   0x0000000000450a6e <+30>:	mov    %eax,0x10(%rsp)
   0x0000000000450a72 <+34>:	movb   $0x1,0x14(%rsp)
   0x0000000000450a77 <+39>:	movq   $0x1,0x18(%rsp)
   0x0000000000450a80 <+48>:	retq
```
 
### 1.3 源代码对应的汇编 plan9汇编

> main函数

```
"".main STEXT size=65 args=0x0 locals=0x20
	0x0000 00000 (hello.go:8)	TEXT	"".main(SB), $32-0
	0x0000 00000 (hello.go:8)	MOVQ	(TLS), CX
	0x0009 00009 (hello.go:8)	CMPQ	SP, 16(CX)
	0x000d 00013 (hello.go:8)	JLS	58
	0x000f 00015 (hello.go:8)	SUBQ	$32, SP
	0x0013 00019 (hello.go:8)	MOVQ	BP, 24(SP)
	0x0018 00024 (hello.go:8)	LEAQ	24(SP), BP
	0x001d 00029 (hello.go:8)	FUNCDATA	$0, gclocals·33cdeccccebe80329f1fdbee7f5874cb(SB)
	0x001d 00029 (hello.go:8)	FUNCDATA	$1, gclocals·33cdeccccebe80329f1fdbee7f5874cb(SB)
	0x001d 00029 (hello.go:8)	MOVQ	$137438953482, AX
	0x0027 00039 (hello.go:9)	MOVQ	AX, (SP)
	0x002b 00043 (hello.go:9)	PCDATA	$0, $0
	0x002b 00043 (hello.go:9)	CALL	"".add(SB)
	0x0030 00048 (hello.go:10)	MOVQ	24(SP), BP
	0x0035 00053 (hello.go:10)	ADDQ	$32, SP
	0x0039 00057 (hello.go:10)	RET
	0x003a 00058 (hello.go:10)	NOP
	0x003a 00058 (hello.go:8)	PCDATA	$0, $-1
	0x003a 00058 (hello.go:8)	CALL	runtime.morestack_noctxt(SB)
	0x003f 00063 (hello.go:8)	JMP	0
```

> add 函数

```
"".add STEXT nosplit size=29 args=0x18 locals=0x0
	0x0000 00000 (hello.go:4)	TEXT	"".add(SB), NOSPLIT, $0-24
	0x0000 00000 (hello.go:4)	FUNCDATA	$0, gclocals·54241e171da8af6ae173d69da0236748(SB)
	0x0000 00000 (hello.go:4)	FUNCDATA	$1, gclocals·33cdeccccebe80329f1fdbee7f5874cb(SB)
	0x0000 00000 (hello.go:4)	MOVL	"".b+12(SP), AX
	0x0004 00004 (hello.go:4)	MOVL	"".a+8(SP), CX
	0x0008 00008 (hello.go:5)	ADDL	CX, AX
	0x000a 00010 (hello.go:5)	MOVL	AX, "".~r2+16(SP)
	0x000e 00014 (hello.go:5)	MOVB	$1, "".~r3+20(SP)
	0x0013 00019 (hello.go:5)	MOVQ	$1, "".~r4+24(SP)
	0x001c 00028 (hello.go:5)	RET
```

## 2 详细分析
>接下来，我们将对plan9汇编的每一句都进行分析

## 2.1 main函数

```
0x0000 00000 (hello.go:8)	TEXT	"".main(SB), $32-0
```

>`0x0000`代表相对偏移地址 
>
>`TEXT	"".main` Text指令声明了`"".main`是.text段的一部分，并表明跟在这个声明后的是函数的函数体
>
>在链接期，"" 这个空字符会被替换为当前的包名。也就是说，`"".main`在链接到二进制文件后会变成 main.main
>
>`(SB)` SB 是一个虚拟寄存器，保存了静态基地址(static-base) 指针，即我们程序地址空间的开始地址。
>
>`"".main(SB)`表明我们的符号位于某个固定的相对地址空间起始处的偏移位置 (最终是由链接器计算得到的)。换句话来讲，它有一个直接的绝对地址: 是一个全局的函数符号。
>
>`$32-0`，32代表当前栈帧的大小，共32; 0代表参数列表长度，无形参和返回参数

```
	0x0000 00000 (hello.go:8)	MOVQ	(TLS), CX
	0x0009 00009 (hello.go:8)	CMPQ	SP, 16(CX)
	0x000d 00013 (hello.go:8)	JLS	58
    ...
	0x003a 00058 (hello.go:10)	NOP
	0x003a 00058 (hello.go:8)	PCDATA	$0, $-1
	0x003a 00058 (hello.go:8)	CALL	runtime.morestack_noctxt(SB)
	0x003f 00063 (hello.go:8)	JMP	0
```

>这一段是用作stack split。goroutine初始分配栈空间2KB，随着运行，可能会栈溢出，因此需要动态增长。
>
>为了防止这种情况发生，runtime 确保 goroutine 在超出栈范围时，会创建一个相当于原来两倍大小的新栈，并将原来栈的上下文拷贝到新栈上。
这个过程被称为 栈分裂(stack-split)，这样使得 goroutine 栈能够动态调整大小。


```
	0x000f 00015 (hello.go:8)	SUBQ	$32, SP
	0x0013 00019 (hello.go:8)	MOVQ	BP, 24(SP)
	0x0018 00024 (hello.go:8)	LEAQ	24(SP), BP
```

>SP - 32，代表当前栈帧的大小， 栈是倒着增长的，SP越小，栈越大（BP不变的情况下）
>
>保留BP在 SP+24 - SP+32 八个字节的位置
>
>新BP等于SP+24所在的位置（老BP存储位置的下面）

```
	0x0000 00000 (hello.go:4)	FUNCDATA	$0, gclocals·54241e171da8af6ae173d69da0236748(SB)
	0x0000 00000 (hello.go:4)	FUNCDATA	$1, gclocals·33cdeccccebe80329f1fdbee7f5874cb(SB)
```

>`FUNCDATA`用作GC，后续再看

```
	0x001d 00029 (hello.go:8)	MOVQ	$137438953482, AX
	0x0027 00039 (hello.go:9)	MOVQ	AX, (SP)
	0x002b 00043 (hello.go:9)	PCDATA	$0, $0
	0x002b 00043 (hello.go:9)	CALL	"".add(SB)
```

>`$137438953482`二进制是

```
echo 'obase=2;137438953482' | bc
```

```
10000000000000000000000000000000001010
拆分
10 0000  32 
0000 0000 0000 0000 0000 0000 0000 1010 10
```

>CALL指令会将返回地址八个字节压栈，SP会变为SP-8

```
	0x0030 00048 (hello.go:10)	MOVQ	24(SP), BP
	0x0035 00053 (hello.go:10)	ADDQ	$32, SP
	0x0039 00057 (hello.go:10)	RET
```

>执行完add函数之后，将BP恢复，将SP恢复，退出

## 2.2 add函数

```
	0x0000 00000 (hello.go:4)	TEXT	"".add(SB), NOSPLIT, $0-24
```

>与main函数一致，NOSPLIT代表不进行stack split算法，0代表函数栈帧为0，24代表形参+返回值共占24个字节。
>
>`func add(a, b int32) (int32, bool, int64)`，int32一个4字节，共3个=12字节，bool一个占4字节（对齐），int64一个8字节

```
	0x0000 00000 (hello.go:4)	MOVL	"".b+12(SP), AX
	0x0004 00004 (hello.go:4)	MOVL	"".a+8(SP), CX
```

>SP+12代表32，SP+8代表10。（注意此处的SP和main函数时候的SP差8个字节，这八个字节是add函数的返回地址）

```
	0x0008 00008 (hello.go:5)	ADDL	CX, AX
	0x000a 00010 (hello.go:5)	MOVL	AX, "".~r2+16(SP)
	0x000e 00014 (hello.go:5)	MOVB	$1, "".~r3+20(SP)
	0x0013 00019 (hello.go:5)	MOVQ	$1, "".~r4+24(SP)
```

>相加结果后放在 SP+16的位置，SP+20放true，SP+24放数字1，然后返回

## 2.3 执行时候的示意图
>下面这个时刻是add函数即将ret时候的状态图

![栈帧说明](https://github.com/xiezhenouc/golanglearn/blob/master/%E5%9B%BE%E7%89%87%E8%AF%B4%E6%98%8E/golang汇编栈帧.png)
