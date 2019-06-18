# 1 golang之slice类型

## 1.1 slice 基础概念
>slice翻译成中文是切片，它和数组相似，但又不完全相同，可以灵活的通过append进行扩容。

>可以通过下标进行访问，如果越界访问，会发生panic`panic: runtime error: index out of range`。

>在源码的`go/src/runtime/slice.go`文件中，定义了slice的结构如下

```golang
type slice struct {
    // 指向底层数组的指针
    array unsafe.Pointer 
    // 长度，切片可以用的长度，下标不能超过长度，否则panic
    len   int
    // 底层数组容量，容量>=长度，如果不扩容，容量是slice扩容的最大限度
	cap   int
}
```

> 底层数组可以被多个slice同时指向，如果其中一个slice通过index改变了一个数组元素的值，那么有可能影响到其他slice。


## 1.2 slice 使用
>可以通过以下方式创建

```golang
// 1 直接声明 nil slice
var mySlice []int
// 2 new关键字, new返回的是指向type slice结构的指针，加个*即为slice类型 nil slice
mySlice := *new([]int)
// 3 字面量
mySlice := []int{1, 2, 3}
// 4 make 
mySlice := make([]int, 2, 4)
// 5 从切片或者数组截取
mySlice := xSlice[0:2]
```

### 1.2.1 nil slice vs empty slice

> `var aSlice []int`创建出来的一个struct，len cap都是0，array指向nil。

> `bSlice := make([]int, 0, 0)`创建出来的一个struct，len cap都是0，array指向`非空`。

>验证code如下

```golang 
func main() {
	var aSlice []int
	var aa = *(*[3]int)(unsafe.Pointer(&aSlice))
	fmt.Println(aa, unsafe.Sizeof(aSlice))

	bSlice := make([]int, 0, 0)
	var bb = *(*[3]int)(unsafe.Pointer(&bSlice))
	fmt.Println(bb, unsafe.Sizeof(bSlice))
}
// 结果
[0 0 0] 24
[842350739104 0 0] 24
```

|  | nil slice | empty slice |
| ------ | ------ | ------ |
| 创建方式1 | var aSlice []int |  var aSlice = []int{} |
| 创建方式2 | var bSlice = *new([]int) | var bSlice = make([]int, 0, 0) |
| 长度 | 0 | 0 |
| 容量 | 0 | 0 |
| 和nil比较|true|false|

### 1.2.2 字面量

```golang
package main

import (
	"fmt"
)

func main() {
	s1 := []int{1, 2, 3, 8: 100}

	fmt.Println(s1)
}
// 结果
[1 2 3 0 0 0 0 0 100]
```

>注意`8: 100`，直接指定第九个元素值为100，中间元素默认都为0

### 1.2.3 make
>make 函数需要传入 切片类型，长度，容量 。容量字段可以不传。

> 1 make 关键字创建slice 

```golang
package main

import (
	"fmt"
)

func main() {
	s1 := make([]int, 5, 10)
	s1[2] = 10

	fmt.Println(s1)
}
```

> 2 这段代码对应的汇编代码

```
"".main STEXT size=238 args=0x0 locals=0x60
	// main函数定义，栈帧大小96
	0x0000 00000 (hello.go:7)	TEXT	"".main(SB), $96-0

	// 判断栈是否需要扩容，如果需要跳228，调用runtime.morestack_noctxt(SB)
	0x0000 00000 (hello.go:7)	MOVQ	(TLS), CX
	0x0009 00009 (hello.go:7)	CMPQ	SP, 16(CX)
	0x000d 00013 (hello.go:7)	JLS	228

	// caller bp压栈，sp bp修改，栈帧结构
	0x0013 00019 (hello.go:7)	SUBQ	$96, SP
	0x0017 00023 (hello.go:7)	MOVQ	BP, 88(SP)
	0x001c 00028 (hello.go:7)	LEAQ	88(SP), BP
	0x0021 00033 (hello.go:7)	FUNCDATA	$0, gclocals·69c1753bd5f81501d95132d08af04464(SB)
	0x0021 00033 (hello.go:7)	FUNCDATA	$1, gclocals·57cc5e9a024203768cbab1c731570886(SB)

	// 调用makeslice函数准备，三个参数 func makeslice(et *_type, len, cap int) slice
	0x0021 00033 (hello.go:7)	LEAQ	type.int(SB), AX
	0x0028 00040 (hello.go:8)	MOVQ	AX, (SP)
	0x002c 00044 (hello.go:8)	MOVQ	$5, 8(SP)
	0x0035 00053 (hello.go:8)	MOVQ	$10, 16(SP)
	0x003e 00062 (hello.go:8)	PCDATA	$0, $0
	0x003e 00062 (hello.go:8)	CALL	runtime.makeslice(SB)

	// 返回值也是slice类型，所以需要三个寄存器，三个局部变量
	0x0043 00067 (hello.go:8)	MOVQ	24(SP), AX
	0x0048 00072 (hello.go:8)	MOVQ	32(SP), CX
	0x004d 00077 (hello.go:8)	MOVQ	40(SP), DX

	// 判断2是否超过，超过的话调用runtime.panicindex，索引越界
	0x0052 00082 (hello.go:9)	CMPQ	CX, $2
	0x0056 00086 (hello.go:9)	JLS	221

	//赋值 s1[2] = 10
	0x005c 00092 (hello.go:9)	MOVQ	$10, 16(AX)

	// 通过AX CX DX将make slice 返回值move到内存其他位置，也称为局部变量，这样就构造出来了slice
	0x0064 00100 (hello.go:11)	MOVQ	AX, ""..autotmp_2+64(SP)
	0x0069 00105 (hello.go:11)	MOVQ	CX, ""..autotmp_2+72(SP)
	0x006e 00110 (hello.go:11)	MOVQ	DX, ""..autotmp_2+80(SP)
	0x0073 00115 (hello.go:11)	MOVQ	$0, ""..autotmp_1+48(SP)
	0x007c 00124 (hello.go:11)	MOVQ	$0, ""..autotmp_1+56(SP)

	// 调用 func convT2Eslice(t *_type, elem unsafe.Pointer) (e eface) {
	0x0085 00133 (hello.go:11)	LEAQ	type.[]int(SB), AX
	0x008c 00140 (hello.go:11)	MOVQ	AX, (SP)
	0x0090 00144 (hello.go:11)	LEAQ	""..autotmp_2+64(SP), AX
	0x0095 00149 (hello.go:11)	MOVQ	AX, 8(SP)
	0x009a 00154 (hello.go:11)	PCDATA	$0, $1
	0x009a 00154 (hello.go:11)	CALL	runtime.convT2Eslice(SB)

	// 结果返回 
	0x009f 00159 (hello.go:11)	MOVQ	16(SP), AX
	0x00a4 00164 (hello.go:11)	MOVQ	24(SP), CX

	// 局部变量
	0x00a9 00169 (hello.go:11)	MOVQ	AX, ""..autotmp_1+48(SP)
	0x00ae 00174 (hello.go:11)	MOVQ	CX, ""..autotmp_1+56(SP)

	// 调用fmt.Println，两个1，一个是长度，一个是容量
	// 这块感觉应该是新创建了一个slice，长度和容量均为1，然后新slice底层array指向的slice的eface地址，最后可以通过反射得出slice里面的基础类型，以及长度，打印出这个slice。查看的源码如下。
	0x00b3 00179 (hello.go:11)	LEAQ	""..autotmp_1+48(SP), AX
	0x00b8 00184 (hello.go:11)	MOVQ	AX, (SP)
	0x00bc 00188 (hello.go:11)	MOVQ	$1, 8(SP)
	0x00c5 00197 (hello.go:11)	MOVQ	$1, 16(SP)
	0x00ce 00206 (hello.go:11)	PCDATA	$0, $1
	0x00ce 00206 (hello.go:11)	CALL	fmt.Println(SB)

	// 栈帧返回
	0x00d3 00211 (hello.go:12)	MOVQ	88(SP), BP
	0x00d8 00216 (hello.go:12)	ADDQ	$96, SP
	0x00dc 00220 (hello.go:12)	RET

	// 索引值超出范围
	0x00dd 00221 (hello.go:9)	PCDATA	$0, $0
	0x00dd 00221 (hello.go:9)	CALL	runtime.panicindex(SB)
	0x00e2 00226 (hello.go:9)	UNDEF
	0x00e4 00228 (hello.go:9)	NOP
	0x00e4 00228 (hello.go:7)	PCDATA	$0, $-1

	// 栈分裂
	0x00e4 00228 (hello.go:7)	CALL	runtime.morestack_noctxt(SB)
	0x00e9 00233 (hello.go:7)	JMP	0
```

> FUNCDATA 和 PCDATA 是编译器产生的，和垃圾收集相关，暂时不用关心

> 先看关键函数调用

```golang
// 创建 slice
CALL	runtime.makeslice(SB)
func makeslice(et *_type, len, cap int) slice {
	// NOTE: The len > maxElements check here is not strictly necessary,
	// but it produces a 'len out of range' error instead of a 'cap out of range' error
	// when someone does make([]T, bignumber). 'cap out of range' is true too,
	// but since the cap is only being supplied implicitly, saying len is clearer.
	// See issue 4085.
	maxElements := maxSliceCap(et.size)
	if len < 0 || uintptr(len) > maxElements {
		panic(errorString("makeslice: len out of range"))
	}

	if cap < len || uintptr(cap) > maxElements {
		panic(errorString("makeslice: cap out of range"))
	}

	// 申请一块内存
	p := mallocgc(et.size*uintptr(cap), et, true)
	return slice{p, len, cap}
}

// 类型转换 func Println(a ...interface{}) (n int, err error)
CALL	runtime.convT2Eslice(SB)
// func Println(a ...interface{}) (n int, err error)
// 实参和形参类型不一致，需要进行类型转换
// 过程：调用 mallocgc 分配一块内存，把数据 copy 进到新的内存，然后返回这块内存的地址，*_type 则直接返回传入的参数。
type _type struct {
	size       uintptr
	ptrdata    uintptr // size of memory prefix holding all pointers
	hash       uint32
	tflag      tflag
	align      uint8
	fieldalign uint8
	kind       uint8
	alg        *typeAlg
	// gcdata stores the GC type data for the garbage collector.
	// If the KindGCProg bit is set in kind, gcdata is a GC program.
	// Otherwise it is a ptrmask bitmap. See mbitmap.go for details.
	gcdata    *byte
	str       nameOff
	ptrToThis typeOff
}
type eface struct {
	_type *_type
	data  unsafe.Pointer
}

func convT2Eslice(t *_type, elem unsafe.Pointer) (e eface) {
	if raceenabled {
		raceReadObjectPC(t, elem, getcallerpc(unsafe.Pointer(&t)), funcPC(convT2Eslice))
	}
	if msanenabled {
		msanread(elem, t.size)
	}
	var x unsafe.Pointer
	if v := *(*slice)(elem); uintptr(v.array) == 0 {
		x = unsafe.Pointer(&zeroVal[0])
	} else {
		x = mallocgc(t.size, t, true)
		*(*slice)(x) = *(*slice)(elem)
	}
	e._type = t
	e.data = x
	return
}

// 打印
CALL	fmt.Println(SB)
// src/fmt/print.go
func Println(a ...interface{}) (n int, err error)
func Fprintln(w io.Writer, a ...interface{}) (n int, err error)
func (p *pp) doPrintln(a []interface{})
func (p *pp) printArg(arg interface{}, verb rune)
func (p *pp) printValue(value reflect.Value, verb rune, depth int) {
...
	case reflect.Array, reflect.Slice:
...
			p.buf.WriteByte('[')
			for i := 0; i < f.Len(); i++ {
				p.printValue(f.Index(i), verb, depth+1)
			}
			p.buf.WriteByte(']')
....

// 栈分裂，栈不够扩容
CALL	runtime.morestack_noctxt(SB)
```

>具体的栈帧图可以参考《码农桃花源》画的非常详细。

### 1.2.4 截取
>截取可以通过数组截取，也可以通过slice截取。

>新slice和老slice使用相同的底层数组，所以修改会影响彼此。

>但是如果发生了append，使得新slice指向了新array，则这两者就不再相关。

>`问题的关键在于两者是否会共用底层数组`

```golang
func main() {
	tmp := []int{123, 456, 789}
	fmt.Println(tmp[0:2]) // data low <= i < high [low, high)，左闭右开
	fmt.Println(tmp[0:2:3]) // data low <= i < high [low, high)，左闭右开，最后是max，用作容量
}
```

## 1.3 slice vs 数组
> slice底层是数组。

> 数组是定长的，其类型包含长度，确定好长度之后，不能再次修改。[3]int 和 [4]int 就是不同的类型。

## 1.4 append 内容
> 使用

```golang
func main() {
	tmp := []int{123, 456, 789}

	newtmp := append(tmp, 999)

	fmt.Println(newtmp)
}
```

> 汇编  func growslice(et *_type, old slice, cap int) slice 

```
	// et *_type
	0x004d 00077 (hello.go:8)	LEAQ	type.int(SB), CX
	0x0054 00084 (hello.go:10)	MOVQ	CX, (SP)
	// old slice
	0x0058 00088 (hello.go:10)	MOVQ	AX, 8(SP)
	0x005d 00093 (hello.go:10)	MOVQ	$3, 16(SP)
	0x0066 00102 (hello.go:10)	MOVQ	$3, 24(SP)
	// cap int
	0x006f 00111 (hello.go:10)	MOVQ	$4, 32(SP)
	0x0078 00120 (hello.go:10)	CALL	runtime.growslice(SB)
```

> append size相关代码

```
func growslice(et *_type, old slice, cap int) slice {

	newcap := old.cap
	doublecap := newcap + newcap
	// 首次申请的长度大于2倍，直接赋值为想要申请的长度
	if cap > doublecap {
		newcap = cap
	} else {
		if old.len < 1024 {
			// 双倍增长
			newcap = doublecap
		} else {
			for newcap < cap {
				// 增长25%
				newcap += newcap / 4
			}
		}
	}
	...
			// 内存对齐，（更多一点）
			capmem = roundupsize(uintptr(newcap))
	...
```

> 实例

```golang
// test
func main() {
	tmp := make([]int, 3)
	tmp[0] = 123
	tmp[1] = 456
	tmp[2] = 789

	oldlen := float64(cap(tmp))
	newlen := float64(cap(tmp))
	myMap := make(map[int]int)
	for i := 0; i < 10000; i++ {
		len := cap(tmp)
		tmp = append(tmp, 1)

		oldlen = newlen
		newlen = float64(cap(tmp))
		// 扩容时候打印扩容了多少
		if oldlen != newlen {
			fmt.Println(oldlen, newlen, newlen/oldlen*1.0)
		}

		if _, ok := myMap[len]; ok {
			continue
		}
		myMap[len] = 1
	}

}
// result
3 6 2
6 12 2
12 24 2
24 48 2
48 96 2
96 192 2
192 384 2
384 768 2
768 1536 2
1536 2048 1.3333333333333333
2048 2560 1.25
2560 3408 1.33125
3408 5120 1.5023474178403755
5120 7168 1.4
7168 9216 1.2857142857142858
9216 12288 1.3333333333333333
```

> 网上结果

```
当原 slice 容量小于 1024 的时候，新 slice 容量变成原来的 2 倍；原 slice 容量超过 1024，新 slice 容量变成原来的1.25倍。
```

> 实际结果

>1 很明显"原 slice 容量超过 1024，新 slice 容量变成原来的1.25倍"，不对，上图结果可以验证
>2 "当原 slice 容量小于 1024 的时候，新 slice 容量变成原来的 2 倍"，也不对，反例如下。注意要看源码

```golang
func main() {
	tmp := make([]int, 3, 3)
	tmp[0] = 123
	tmp[1] = 456
	tmp[2] = 789

	fmt.Println(len(tmp), cap(tmp))
	tmp = append(tmp, 7, 8, 9, 10, 11)
	fmt.Println(len(tmp), cap(tmp))
}
//result
3 3
8 8
//解释，如果是按照 "小于 1024 的时候，新 slice 容量变成原来的 2 倍"，那么容量只能是3->6->12，但现在是8。原因是源码有这一段
	// 首次申请的长度大于2倍，直接赋值为想要申请的长度
	if cap > doublecap {
		newcap = cap
	}
	// 内存对齐
	capmem = roundupsize(uintptr(newcap))
```

>内存对齐源码

```golang
func roundupsize(size uintptr) uintptr {
	if size < _MaxSmallSize {
		if size <= smallSizeMax-8 {
			return uintptr(class_to_size[size_to_class8[(size+smallSizeDiv-1)/smallSizeDiv]])
		} else {
			return uintptr(class_to_size[size_to_class128[(size-smallSizeMax+largeSizeDiv-1)/largeSizeDiv]])
		}
	}
	if size+_PageSize < size {
		return size
	}
	return round(size, _PageSize)
}

class_to_size和size_to_class8是由执行go run mksizeclasses.go得到的，会在sizeclasses.go文件中
```

## 1.5 为什么nil slice可以直接append
>源码，直接申请了一块空间，作为nil->empty底层指向的地址，后续append会指向新的

```golang
	if et.size == 0 {
		if cap < old.cap {
			panic(errorString("growslice: cap out of range"))
		}
		// append should not create a slice with nil pointer but non-zero len.
		// We assume that append doesn't need to preserve old.array in this case.
		return slice{unsafe.Pointer(&zerobase), old.len, cap}
	}

// malloc.go
// base address for all 0-byte allocations
var zerobase uintptr
```

## 1.6 传 slice 和 slice指针 区别
> slice类型其实是一个结构体，传slice指针当然会使得传的参数大小小一些，但是不管是传slice还是slice指针，如果修改了slice底层指向的数组的元素的值，则就会改变它。

>在slice作为一个函数实参的时候，实际上是值赋值，没有引用传递，上文中其实可以看到。但是值赋值的指针的值指向的是同一块内存地址。所以任何更改都是真的针对底层数组修改。

>`Go 语言的函数参数传递，只有值传递，没有引用传递`

## 1.7 总结

> 切片是对底层数组的一个抽象，描述了它的一个片段。

> 切片实际上是一个结构体，它有三个字段：长度，容量，底层数据的地址。

> 多个切片可能共享同一个底层数组，这种情况下，对其中一个切片或者底层数组的更改，会影响到其他切片。

> append 函数会在切片容量不够的情况下，调用 growslice 函数获取所需要的内存，这称为扩容，扩容会改变元素原来的位置。

> 扩容策略并不是简单的扩为原切片容量的 2 倍或 1.25 倍，还有内存对齐的操作。扩容后的容量 >= 原容量的 2 倍或 1.25 倍。

> 当直接用切片作为函数参数时，可以改变切片的元素，不能改变切片本身；想要改变切片本身，可以将改变后的切片返回，函数调用者接收改变后的切片或者将切片指针作为函数参数。

## 1.8 参考资料

>《码农桃花源》 https://mp.weixin.qq.com/s/MTZ0C9zYsNrb8wyIm2D8BA
> https://blog.golang.org/go-slices-usage-and-internals