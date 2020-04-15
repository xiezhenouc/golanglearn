# golang之map类型

> map是一种非常灵活的类型，本文将介绍map的赋值、删除、查询、扩容的详细过程

## 1 what map是什么
>map一般指`<k, v>`对，k一般是唯一的。

>和map相关的操作包括

>1 新建map

>2 map新增元素（增）

>3 map删除元素（删）

>4 map修改元素（改）

>5 map查找元素（查）

>实现map的数据结构，一般包括`哈希查找表（hash table）`，`搜索树（二叉搜索树/AVL树/红黑树）`

>哈希查找表，通过哈希函数间将key分配到不同的bucket中，即为不同的数组中，通过数组的索引即可实现快速访问。查找时间=hash函数计算时间+数据访问时间。如果发生哈希冲突，需要进行解决，开放地址法和链表法。开放地址法的就是冲突后再去找后面空闲的位置，链表法是如果冲突了通过链表的形式添加元素。查找复杂度一般是 O(1)、

>搜索树。实现方式是 二叉搜索树/AVL树/红黑树，查找复杂度一般 O(lgN)。 

## 2 why 为什么要用map
>方便&灵活

>可以实现数据的查找/添加/删除，操作效率高。

>多种基本类型的数据存储在同一个类型中。如 `map[string]int`，`string`作为`key`, `int`类型作为`value`

## 3 how map用法有哪些，到底是怎么实现的

### 3.1 map简单使用，对应的汇编代码&源代码

```golang
package main

import (
	"fmt"
)

func main() {
	// 创建
	myMap := make(map[string]int)

	// 赋值
	myMap["age"] = 15
	myMap["age"] = 20

	// 删除
	delete(myMap, "age")

	// 扩容（内部实现，可能扩容）
	myMap["a"] = -1
	myMap["b"] = -2
	myMap["c"] = -3

	// 遍历
	for k, v := range myMap {
		fmt.Println(k, v)
	}

	fmt.Println(myMap)
}
```

>汇编代码

```golang
// main函数声明，栈帧 232个字节
0x0000 00000 (test.go:7)	TEXT	"".main(SB), $232-0

// 是否需要栈分裂
0x0000 00000 (test.go:7)	MOVQ	(TLS), CX
0x0009 00009 (test.go:7)	LEAQ	-104(SP), AX
0x000e 00014 (test.go:7)	CMPQ	AX, 16(CX)
0x0012 00018 (test.go:7)	JLS	830

// caller bp压栈
0x0018 00024 (test.go:7)	SUBQ	$232, SP
0x001f 00031 (test.go:7)	MOVQ	BP, 224(SP)
0x0027 00039 (test.go:7)	LEAQ	224(SP), BP
0x002f 00047 (test.go:7)	FUNCDATA	$0, gclocals·3e27b3aa6b89137cce48b3379a2a6610(SB)
0x002f 00047 (test.go:7)	FUNCDATA	$1, gclocals·ebf41f04cb65af8ce6fb34bc5270cfb1(SB)

// 传参数，共四个参数
0x002f 00047 (test.go:7)	LEAQ	type.map[string]int(SB), AX
0x0036 00054 (test.go:9)	MOVQ	AX, (SP)
0x003a 00058 (test.go:9)	MOVQ	$0, 8(SP)
0x0043 00067 (test.go:9)	MOVQ	$0, 16(SP)
0x004c 00076 (test.go:9)	MOVQ	$0, 24(SP)
0x0055 00085 (test.go:9)	PCDATA	$0, $0

// runtime/hashmap.go 
// map创建
// func makemap(t *maptype, hint int64, h *hmap, bucket unsafe.Pointer) *hmap
0x0055 00085 (test.go:9)	CALL	runtime.makemap(SB)

// 返回值只有一个指针 *hmap ，指向新创建的hashmap
0x005a 00090 (test.go:9)	MOVQ	32(SP), AX
// h *hmap = "".myMap+56(SP)
0x005f 00095 (test.go:9)	MOVQ	AX, "".myMap+56(SP)

// 传参数，共三个参数
0x0064 00100 (test.go:9)	LEAQ	type.map[string]int(SB), CX
0x006b 00107 (test.go:12)	MOVQ	CX, (SP)
0x006f 00111 (test.go:12)	MOVQ	AX, 8(SP)
0x0074 00116 (test.go:12)	LEAQ	go.string."age"(SB), DX
0x007b 00123 (test.go:12)	MOVQ	DX, 16(SP)
0x0080 00128 (test.go:12)	MOVQ	$3, 24(SP)
0x0089 00137 (test.go:12)	PCDATA	$0, $1

// runtime/hashmap_fast.go
// func mapassign_faststr(t *maptype, h *hmap, ky string) unsafe.Pointer
0x0089 00137 (test.go:12)	CALL	runtime.mapassign_faststr(SB)

// 返回值只有一个指针，即为"age"键值指向的内存地址
0x008e 00142 (test.go:12)	MOVQ	32(SP), AX
// 15赋值
0x0093 00147 (test.go:12)	MOVQ	$15, (AX)

// 准备参数->mapassign_faststr->20赋值
0x009a 00154 (test.go:12)	LEAQ	type.map[string]int(SB), AX
0x00a1 00161 (test.go:13)	MOVQ	AX, (SP)
0x00a5 00165 (test.go:13)	MOVQ	"".myMap+56(SP), CX
0x00aa 00170 (test.go:13)	MOVQ	CX, 8(SP)
0x00af 00175 (test.go:13)	LEAQ	go.string."age"(SB), DX
0x00b6 00182 (test.go:13)	MOVQ	DX, 16(SP)
0x00bb 00187 (test.go:13)	MOVQ	$3, 24(SP)
0x00c4 00196 (test.go:13)	PCDATA	$0, $1
0x00c4 00196 (test.go:13)	CALL	runtime.mapassign_faststr(SB)
0x00c9 00201 (test.go:13)	MOVQ	32(SP), AX
0x00ce 00206 (test.go:13)	MOVQ	$20, (AX)

// 准备参数->mapdelete_faststr
0x00d5 00213 (test.go:13)	LEAQ	type.map[string]int(SB), AX
0x00dc 00220 (test.go:16)	MOVQ	AX, (SP)
0x00e0 00224 (test.go:16)	MOVQ	"".myMap+56(SP), CX
0x00e5 00229 (test.go:16)	MOVQ	CX, 8(SP)
0x00ea 00234 (test.go:16)	LEAQ	go.string."age"(SB), DX
0x00f1 00241 (test.go:16)	MOVQ	DX, 16(SP)
0x00f6 00246 (test.go:16)	MOVQ	$3, 24(SP)
0x00ff 00255 (test.go:16)	PCDATA	$0, $1
// runtime/hashmap_fast.go
// func mapdelete_faststr(t *maptype, h *hmap, ky string)
0x00ff 00255 (test.go:16)	CALL	runtime.mapdelete_faststr(SB)

// myMap["a"] = -1
// myMap["b"] = -2
// myMap["c"] = -3
0x0104 00260 (test.go:16)	LEAQ	type.map[string]int(SB), AX
0x010b 00267 (test.go:19)	MOVQ	AX, (SP)
0x010f 00271 (test.go:19)	MOVQ	"".myMap+56(SP), CX
0x0114 00276 (test.go:19)	MOVQ	CX, 8(SP)
0x0119 00281 (test.go:19)	LEAQ	go.string."a"(SB), DX
0x0120 00288 (test.go:19)	MOVQ	DX, 16(SP)
0x0125 00293 (test.go:19)	MOVQ	$1, 24(SP)
0x012e 00302 (test.go:19)	PCDATA	$0, $1
0x012e 00302 (test.go:19)	CALL	runtime.mapassign_faststr(SB)
0x0133 00307 (test.go:19)	MOVQ	32(SP), AX
0x0138 00312 (test.go:19)	MOVQ	$-1, (AX)
0x013f 00319 (test.go:19)	LEAQ	type.map[string]int(SB), AX
0x0146 00326 (test.go:20)	MOVQ	AX, (SP)
0x014a 00330 (test.go:20)	MOVQ	"".myMap+56(SP), CX
0x014f 00335 (test.go:20)	MOVQ	CX, 8(SP)
0x0154 00340 (test.go:20)	LEAQ	go.string."b"(SB), DX
0x015b 00347 (test.go:20)	MOVQ	DX, 16(SP)
0x0160 00352 (test.go:20)	MOVQ	$1, 24(SP)
0x0169 00361 (test.go:20)	PCDATA	$0, $1
0x0169 00361 (test.go:20)	CALL	runtime.mapassign_faststr(SB)
0x016e 00366 (test.go:20)	MOVQ	32(SP), AX
0x0173 00371 (test.go:20)	MOVQ	$-2, (AX)
0x017a 00378 (test.go:20)	LEAQ	type.map[string]int(SB), AX
0x0181 00385 (test.go:21)	MOVQ	AX, (SP)
0x0185 00389 (test.go:21)	MOVQ	"".myMap+56(SP), CX
0x018a 00394 (test.go:21)	MOVQ	CX, 8(SP)
0x018f 00399 (test.go:21)	LEAQ	go.string."c"(SB), DX
0x0196 00406 (test.go:21)	MOVQ	DX, 16(SP)
0x019b 00411 (test.go:21)	MOVQ	$1, 24(SP)
0x01a4 00420 (test.go:21)	PCDATA	$0, $1
0x01a4 00420 (test.go:21)	CALL	runtime.mapassign_faststr(SB)
0x01a9 00425 (test.go:21)	MOVQ	32(SP), AX
0x01ae 00430 (test.go:21)	MOVQ	$-3, (AX)

// 遍历
// 此部分有疑惑
// it *hiter = ""..autotmp_4+128(SP)
0x01b5 00437 (test.go:24)	LEAQ	""..autotmp_4+128(SP), DI
// X0寄存器清空
0x01bd 00445 (test.go:24)	XORPS	X0, X0
// DI寄存器-32
0x01c0 00448 (test.go:24)	ADDQ	$-32, DI
// ?
0x01c4 00452 (test.go:24)	DUFFZERO	$273

// 准备三个参数 
// t *maptype = type.map[string]int(SB)
0x01d7 00471 (test.go:24)	LEAQ	type.map[string]int(SB), AX
0x01de 00478 (test.go:24)	MOVQ	AX, (SP)

// h *hmap = "".myMap+56(SP)
0x01e2 00482 (test.go:24)	MOVQ	"".myMap+56(SP), CX
0x01e7 00487 (test.go:24)	MOVQ	CX, 8(SP)

// it *hiter = ""..autotmp_4+128(SP)
0x01ec 00492 (test.go:24)	LEAQ	""..autotmp_4+128(SP), DX
0x01f4 00500 (test.go:24)	MOVQ	DX, 16(SP)
0x01f9 00505 (test.go:24)	PCDATA	$0, $2

// runtime/hashmap.go
// func mapiterinit(t *maptype, h *hmap, it *hiter) 
0x01f9 00505 (test.go:24)	CALL	runtime.mapiterinit(SB)

// 去725判断是否遍历完毕
0x01fe 00510 (test.go:24)	JMP	725

// 说明有结果，取出key和value
// type hiter struct {
//    key         unsafe.Pointer 
//    value       unsafe.Pointer
//    ...

// v 返回值
0x0203 00515 (test.go:24)	MOVQ	""..autotmp_4+136(SP), CX
0x020b 00523 (test.go:24)	MOVQ	(CX), CX

// k 返回值
0x020e 00526 (test.go:24)	MOVQ	8(AX), DX
0x0212 00530 (test.go:24)	MOVQ	(AX), AX

// k ""..autotmp_6+80(SP)
0x0215 00533 (test.go:25)	MOVQ	AX, ""..autotmp_6+80(SP)
0x021a 00538 (test.go:25)	MOVQ	DX, ""..autotmp_6+88(SP)

// v ""..autotmp_7+48(SP)
0x021f 00543 (test.go:25)	MOVQ	CX, ""..autotmp_7+48(SP)

// 其余局部变量，后面转换类型的时候使用
0x0224 00548 (test.go:25)	MOVQ	$0, ""..autotmp_5+96(SP)
0x022d 00557 (test.go:25)	MOVQ	$0, ""..autotmp_5+104(SP)
0x0236 00566 (test.go:25)	MOVQ	$0, ""..autotmp_5+112(SP)
0x023f 00575 (test.go:25)	MOVQ	$0, ""..autotmp_5+120(SP)

// k ""..autotmp_6+80(SP)-> ...interface{}
0x0248 00584 (test.go:25)	LEAQ	type.string(SB), AX
0x024f 00591 (test.go:25)	MOVQ	AX, (SP)
0x0253 00595 (test.go:25)	LEAQ	""..autotmp_6+80(SP), CX
0x0258 00600 (test.go:25)	MOVQ	CX, 8(SP)
0x025d 00605 (test.go:25)	PCDATA	$0, $3
// func convT2Estring(t *_type, elem unsafe.Pointer) (e eface)
0x025d 00605 (test.go:25)	CALL	runtime.convT2Estring(SB)
0x0262 00610 (test.go:25)	MOVQ	24(SP), AX
0x0267 00615 (test.go:25)	MOVQ	16(SP), CX
0x026c 00620 (test.go:25)	MOVQ	CX, ""..autotmp_5+96(SP)
0x0271 00625 (test.go:25)	MOVQ	AX, ""..autotmp_5+104(SP)

// v ""..autotmp_7+48(SP) -> ...interface{}
0x0276 00630 (test.go:25)	LEAQ	type.int(SB), AX
0x027d 00637 (test.go:25)	MOVQ	AX, (SP)
0x0281 00641 (test.go:25)	LEAQ	""..autotmp_7+48(SP), CX
0x0286 00646 (test.go:25)	MOVQ	CX, 8(SP)
0x028b 00651 (test.go:25)	PCDATA	$0, $3
// func convT2E64(t *_type, elem unsafe.Pointer) (e eface)
0x028b 00651 (test.go:25)	CALL	runtime.convT2E64(SB)
0x0290 00656 (test.go:25)	MOVQ	24(SP), AX
0x0295 00661 (test.go:25)	MOVQ	16(SP), CX
0x029a 00666 (test.go:25)	MOVQ	CX, ""..autotmp_5+112(SP)
0x029f 00671 (test.go:25)	MOVQ	AX, ""..autotmp_5+120(SP)
// fmt.Println
0x02a4 00676 (test.go:25)	LEAQ	""..autotmp_5+96(SP), AX
0x02a9 00681 (test.go:25)	MOVQ	AX, (SP)
0x02ad 00685 (test.go:25)	MOVQ	$2, 8(SP)
0x02b6 00694 (test.go:25)	MOVQ	$2, 16(SP)
0x02bf 00703 (test.go:25)	PCDATA	$0, $3
0x02bf 00703 (test.go:25)	CALL	fmt.Println(SB)

// 准备参数，判断是否需要再次遍历 ""..autotmp_4+128(SP) = (it *hiter)
0x02c4 00708 (test.go:25)	LEAQ	""..autotmp_4+128(SP), AX
0x02cc 00716 (test.go:24)	MOVQ	AX, (SP)
0x02d0 00720 (test.go:24)	PCDATA	$0, $2

// func mapiternext(it *hiter)
0x02d0 00720 (test.go:24)	CALL	runtime.mapiternext(SB)

// 如果完毕，直接结束
// ""..autotmp_4+128(SP) = (it *hiter)，如果为空，说明已经无元素了
0x02d5 00725 (test.go:24)	MOVQ	""..autotmp_4+128(SP), AX
0x02dd 00733 (test.go:24)	TESTQ	AX, AX
0x02e0 00736 (test.go:24)	JNE	515

// fmt.Println
0x02e6 00742 (test.go:28)	MOVQ	$0, ""..autotmp_8+64(SP)
0x02ef 00751 (test.go:28)	MOVQ	$0, ""..autotmp_8+72(SP)
0x02f8 00760 (test.go:28)	LEAQ	type.map[string]int(SB), AX
0x02ff 00767 (test.go:28)	MOVQ	AX, ""..autotmp_8+64(SP)
0x0304 00772 (test.go:28)	MOVQ	"".myMap+56(SP), AX
0x0309 00777 (test.go:28)	MOVQ	AX, ""..autotmp_8+72(SP)
0x030e 00782 (test.go:28)	LEAQ	""..autotmp_8+64(SP), AX
0x0313 00787 (test.go:28)	MOVQ	AX, (SP)
0x0317 00791 (test.go:28)	MOVQ	$1, 8(SP)
0x0320 00800 (test.go:28)	MOVQ	$1, 16(SP)
0x0329 00809 (test.go:28)	PCDATA	$0, $4
0x0329 00809 (test.go:28)	CALL	fmt.Println(SB)
0x032e 00814 (test.go:29)	MOVQ	224(SP), BP
0x0336 00822 (test.go:29)	ADDQ	$232, SP
0x033d 00829 (test.go:29)	RET

// 栈分裂
0x033e 00830 (test.go:29)	NOP
0x033e 00830 (test.go:7)	PCDATA	$0, $-1
0x033e 00830 (test.go:7)	CALL	runtime.morestack_noctxt(SB)
0x0343 00835 (test.go:7)	JMP	0
```

### 3.2 基础类型说明

```golang
// A header for a Go map.
type hmap struct {
	// 元素个数，调用 len(map) 时，直接返回此值
	count     int
	// buckets 的对数 log_2
	flags     uint8
	B         uint8  // log_2 of # of buckets (can hold up to loadFactor * 2^B items)
	noverflow uint16 // approximate number of overflow buckets; see incrnoverflow for details
	hash0     uint32 // hash seed

	buckets    unsafe.Pointer // array of 2^B Buckets. may be nil if count==0.
	oldbuckets unsafe.Pointer // previous bucket array of half the size, non-nil only when growing
	nevacuate  uintptr        // progress counter for evacuation (buckets less than this have been evacuated)

	extra *mapextra // optional fields
}

// mapextra holds fields that are not present on all maps.
type mapextra struct {
	// If both key and value do not contain pointers and are inline, then we mark bucket
	// type as containing no pointers. This avoids scanning such maps.
	// However, bmap.overflow is a pointer. In order to keep overflow buckets
	// alive, we store pointers to all overflow buckets in hmap.overflow.
	// Overflow is used only if key and value do not contain pointers.
	// overflow[0] contains overflow buckets for hmap.buckets.
	// overflow[1] contains overflow buckets for hmap.oldbuckets.
	// The indirection allows to store a pointer to the slice in hiter.
	overflow [2]*[]*bmap

	// nextOverflow holds a pointer to a free overflow bucket.
	nextOverflow *bmap
}

// A bucket for a Go map.
type bmap struct {
	// tophash generally contains the top byte of the hash value
	// for each key in this bucket. If tophash[0] < minTopHash,
	// tophash[0] is a bucket evacuation state instead.
	tophash [bucketCnt]uint8
	// Followed by bucketCnt keys and then bucketCnt values.
	// NOTE: packing all the keys together and then all the values together makes the
	// code a bit more complicated than alternating key/value/key/value/... but it allows
	// us to eliminate padding which would be needed for, e.g., map[int64]int8.
	// Followed by an overflow pointer.
}

// 迭代结构
// A hash iteration structure.
// If you modify hiter, also change cmd/internal/gc/reflect.go to indicate
// the layout of this structure.
type hiter struct {
	key         unsafe.Pointer // Must be in first position.  Write nil to indicate iteration end (see cmd/internal/gc/range.go).
	value       unsafe.Pointer // Must be in second position (see cmd/internal/gc/range.go).
	t           *maptype
	h           *hmap
	buckets     unsafe.Pointer // bucket ptr at hash_iter initialization time
	bptr        *bmap          // current bucket
	overflow    [2]*[]*bmap    // keeps overflow buckets alive
	startBucket uintptr        // bucket iteration started at
	offset      uint8          // intra-bucket offset to start from during iteration (should be big enough to hold bucketCnt-1)
	wrapped     bool           // already wrapped around from end of bucket array to beginning
	B           uint8
	i           uint8
	bucket      uintptr
	checkBucket uintptr
}
```

### 3.3 创建 makemap

```golang
// ./runtime/type.go
type maptype struct {
	typ           _type
	key           *_type
	elem          *_type
	bucket        *_type // internal type representing a hash bucket
	hmap          *_type // internal type representing a hmap
	keysize       uint8  // size of key slot
	indirectkey   bool   // store ptr to key instead of key itself
	valuesize     uint8  // size of value slot
	indirectvalue bool   // store ptr to value instead of value itself
	bucketsize    uint16 // size of bucket
	reflexivekey  bool   // true if k==k for all keys
	needkeyupdate bool   // true if we need to update key on an overwrite
}

// make(map[k]v, hint)的内部实现是makemap，这个函数实现了Go的map的初始化
// 如果编译器认为map或者第一个bucket可以创建在堆区，hint或者bucket参数可能不为空
// 如果h!=nil，map可以直接继续在h创建
// 如果bucket!=nil，bucket可以直接在第一个bucket创建
// 以我们例子为例，此处 各个参数的值为（map[string]int， 0， 0， 0）
func makemap(t *maptype, hint int64, h *hmap, bucket unsafe.Pointer) *hmap {
	if sz := unsafe.Sizeof(hmap{}); sz > 48 || sz != t.hmap.size {
		println("runtime: sizeof(hmap) =", sz, ", t.hmap.size =", t.hmap.size)
		throw("bad hmap size")
	}

	if hint < 0 || hint > int64(maxSliceCap(t.bucket.size)) {
		hint = 0
	}

	if !ismapkey(t.key) {
		throw("runtime.makemap: unsupported map key type")
	}

	// check compiler's and reflect's math
	if t.key.size > maxKeySize && (!t.indirectkey || t.keysize != uint8(sys.PtrSize)) ||
		t.key.size <= maxKeySize && (t.indirectkey || t.keysize != uint8(t.key.size)) {
		throw("key size wrong")
	}
	if t.elem.size > maxValueSize && (!t.indirectvalue || t.valuesize != uint8(sys.PtrSize)) ||
		t.elem.size <= maxValueSize && (t.indirectvalue || t.valuesize != uint8(t.elem.size)) {
		throw("value size wrong")
	}

	// invariants we depend on. We should probably check these at compile time
	// somewhere, but for now we'll do it here.
	if t.key.align > bucketCnt {
		throw("key align too big")
	}
	if t.elem.align > bucketCnt {
		throw("value align too big")
	}
	if t.key.size%uintptr(t.key.align) != 0 {
		throw("key size not a multiple of key align")
	}
	if t.elem.size%uintptr(t.elem.align) != 0 {
		throw("value size not a multiple of value align")
	}
	if bucketCnt < 8 {
		throw("bucketsize too small for proper alignment")
	}
	if dataOffset%uintptr(t.key.align) != 0 {
		throw("need padding in bucket (key)")
	}
	if dataOffset%uintptr(t.elem.align) != 0 {
		throw("need padding in bucket (value)")
	}

    // 扩容
    // find size parameter which will hold the requested # of elements
    // 慢慢增大B，找到合适的hint
	B := uint8(0)
	for ; overLoadFactor(hint, B); B++ {
	}

    // allocate initial hash table
    // 如果B=0，就惰性初始化
	// if B == 0, the buckets field is allocated lazily later (in mapassign)
	// If hint is large zeroing this memory could take a while.
	buckets := bucket
	var extra *mapextra
	if B != 0 {
		var nextOverflow *bmap
        // 创建bucket数组
		buckets, nextOverflow = makeBucketArray(t, B)
		if nextOverflow != nil {
            // 串联?
			extra = new(mapextra)
			extra.nextOverflow = nextOverflow
		}
	}

    // 首次make走如下逻辑
	// initialize Hmap
	if h == nil {
		h = (*hmap)(newobject(t.hmap))
	}
	h.count = 0
	h.B = B
	h.extra = extra
	h.flags = 0
	h.hash0 = fastrand()
	h.buckets = buckets
	h.oldbuckets = nil
	h.nevacuate = 0
	h.noverflow = 0

	return h
}


// 一个桶内包含多少值
// Maximum number of key/value pairs a bucket can hold.
bucketCntBits = 3
// 1<<3 = 8个
bucketCnt     = 1 << bucketCntBits

// Maximum average load of a bucket that triggers growth.
loadFactor = 6.5

// overLoadFactor reports whether count items placed in 1<<B buckets is over loadFactor.
func overLoadFactor(count int64, B uint8) bool {
    // count >= bucketCnt 元素个数比桶数还多
    // (uint64(1)<<B) 创建2^B个桶
    // loadFactor*float32((uint64(1)<<B)) 一共包含6.5*2^B个桶，如果当前元素超过这个，就扩容
	return count >= bucketCnt && float32(count) >= loadFactor*float32((uint64(1)<<B))
}
```

### 3.4 赋值 mapassign_faststr

### 3.5 删除 mapdelete_faststr

### 3.6 遍历 初始化mapiterinit 下一个mapiternext
