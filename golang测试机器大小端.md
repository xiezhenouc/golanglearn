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

![内存布局](https://github.com/xiezhenouc/golanglearn/blob/master/%E5%9B%BE%E7%89%87%E8%AF%B4%E6%98%8E/%E5%86%85%E5%AD%98%E5%B8%83%E5%B1%80.png)


## 1 查看系统文件

## 2 代码测试

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