# go mod 学习

## 模块管理
```
golang的依赖管理，一直是一个比较容易被诟病的点。之前是依靠gopath，后续出现了vendor godeps等工具，现在，官方推出了go mod
记录下go mod的学习点
```

## 快速开始
```golang
1 在github上新建两个代码库，这二者未来的关系是，gomodtest依赖gomodlib提供的函数
https://github.com/xiezhenouc/gomodlib
https://github.com/xiezhenouc/gomodtest


2 gomodlib
2.1 在gomodlib中，新建一个文件test.go，内容如下
package gomodlib

func Hi() {
    println("hi")
}

2.2 初始化go mod
go mod init github.com/xiezhenouc/gomodlib

2.3 下载依赖&清理不再使用的包
go mod tidy

2.4 提交代码到github，合入


3 gomodtest
3.1 新建一个main.go，code内容如下，其中调用了lib库的函数
package main

import (
	"github.com/xiezhenouc/gomodlib"
)

func main() {
    gomodlib.Hi()
}

3.2 初始化go mod
go mod init github.com/xiezhenouc/gomodlib

3.3 下载依赖&清理不再使用的包
go mod tidy

3.4 执行代码
$go run main.go
hi

3.5 观察go.mod
cat go.mod
module github.com/xiezhenouc/gomodtest

go 1.13

require github.com/xiezhenouc/gomodlib v0.0.0-20191211031514-6c90f9cd873e

3.6 查看下载下来的gomodlib的代码，在gopath的/pkg/mod/github.com/xiezhenouc/文件夹中

3.7 清空缓存，即清空 /pkg/mod/ 下面的文件
go clean --modcache

3.8 需要更新lib中指定分支的代码
go get github.com/xiezhenouc/gomodlib@hello
其他情况
go get foo@v1.2.3
go get foo@master
go get foo@e3702bed2


4 gomodlib库有更新
4.1 gomodlib中拉一个hello分支，新增一个函数，将其提交并合入
func Hello() {
    println("hello")
}

4.2 gomodtest中增加调用此函数的语句
package main

import (
	"github.com/xiezhenouc/gomodlib"
)

func main() {
	gomodlib.Hi()
	gomodlib.Hello()
}

4.3 更新gomodtest中go.mod
go get github.com/xiezhenouc/gomodlib@hello

4.4 执行
$ go run main.go
hi
hello

5 本地开发
5.1 本地gomodlib中，再次新增一个函数，先不提交
func Haha() {
    println("haha")
}

5.2 gomodtest中增加调用此函数的语句
package main

import (
	"github.com/xiezhenouc/gomodlib"
)

func main() {
	gomodlib.Hi()
	gomodlib.Hello()
	gomodlib.Haha()
}

5.3 更新gomodtest中go.mod，replace的意思是先用本地版本替代
replace github.com/xiezhenouc/gomodlib => /home/work/tmp/gomod/gomodlib

5.4 执行
$ go run main.go
hi
hello
haha
```

## 基本命令
```
1 初始化
go mod init github.com/xiezhenouc/gomodtest
2 清空缓存
go clean --modcache
3 整理&下载依赖
go mod tidy
4 更新某个分支、版本
go get github.com/xiezhenouc/gomodlib@hello
go get foo@v1.2.3
go get foo@master
go get foo@e3702bed2
5 开发环境
replace github.com/xiezhenouc/gomodlib => /home/work/tmp/gomod/gomodlib

```


## 参考资料
```
https://golang.org/doc/go1.13#modules
https://juejin.im/post/5c8e503a6fb9a070d878184a
```