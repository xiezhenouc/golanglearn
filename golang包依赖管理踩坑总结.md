# golang 第三方依赖管理

## 1 vendor
>我们在写代码的时候，经常是通过import其他开源模块的包来进行工作的，其他模块需要在$GOPATH or $GOROOT中存在才可以，否则我们go build的时候会报错，报找不到对应的代码。
>
>举例说明，如，我们建立一个新的代码库 github.com/xiezhenouc/godependencydemo。
>main.go的主要内容是调用另外一个代码库的web框架，从而实现自己的业务模块。
>具体操作过程 
>
>go get github.com/xiezhenouc/godependencydemo
>
>go get github.com/xiezhenouc/golangwebframework
>
>cd $GOPATH/src/github.com/xiezhenouc/godependencydemo && go run main.go
>
>访问 http://127.0.0.1:8999/test/sayhi 出现"say hi ..."即为成功

```
package main

import (
	"fmt"
	myframework "github.com/xiezhenouc/golangwebframework"
)

type TestController struct {
	Ctx *myframework.Context
}

func (t *TestController) Init(context *myframework.Context) {
	t.Ctx = context
}

func (t *TestController) SayHi() {
	fmt.Fprintln(t.Ctx.Output, "say hi ...")
}

func (t *TestController) SayYes() {
	fmt.Fprintln(t.Ctx.Output, "say yes ...")
}

func main() {
	fw := myframework.New()

	fw.AddAutoRouter("/test/", &TestController{})

	fw.Run(":8999")

}
```
### 1.1 问题
>如果我们依赖的github.com/xiezhenouc/golangwebframework升级了，且升级无法兼容前面的版本，或者这个代码库作者直接删除对应代码，那导致的后果就是我们的github.com/xiezhenouc/godependencydemo也无法运行了，尽管我们没有做任何代码的改动。
>
>所以，就有人提出来了vendor。go vendor 是go 1.5 官方引入管理包依赖的方式，1.6正式引入。
>
>其基本思路是，将引用的外部包的源代码放在当前工程的vendor目录下面，相当于copy一份代码到自己的目录里面。如果别人升级代码，也不会影响到自己工程的代码。
>
>go 1.6以后编译go代码会优先从vendor目录先寻找依赖包。

## 2 godep
>但vendor还会有问题，vendor目录中只有依赖包的代码，没有依赖包的版本信息，对于升级/追溯问题，有点困难。如何方便的得到本项目依赖了哪些包，并方便的将其拷贝到vendor目录下？手动一个一个拷贝太傻了。
>基于此，有人提出了godep。

### 2.1 godep save
>还按照我们上述的例子来说，我们首先进入
>
>cd $GOPATH/src/github.com/xiezhenouc/godependencydemo
>
>确保当前目录只有main.go文件，删除其他所有文件
>
>然后执行`godep save`，这个时候，会发现多了Godeps文件夹和vendor目录（对于不支持vendor的早期版本，会将依赖的代码从GOPATH/src中copy到Godeps/_workspace/里）
>
>Godeps/Godeps.json内容如下。说明godependencydemo依赖于github.com/xiezhenouc/golangwebframework，github.com/xiezhenouc/golangwebframework的版本信息是4bb157000ee75b8dd28e150190ea224d90b41a6f。

```
{
	"ImportPath": "myhello",
	"GoVersion": "go1.9",
	"GodepVersion": "v80",
	"Deps": [
		{
			"ImportPath": "github.com/xiezhenouc/golangwebframework",
			"Rev": "4bb157000ee75b8dd28e150190ea224d90b41a6f"
		}
	]
}
```
>vendor目录的内容是vendor/github.com/xiezhenouc/golangwebframework就是依赖包的源代码拷贝副本。通过这个操作，依赖包已经放到自己的代码路径下了。

#### 2.1.1 注意：
>1 如果出现godep: Package (github.com/xiezhenouc/golangwebframework) not found，说明golangwebframework根本没有下载到$GOPATH中，需要go get github.com/xiezhenouc/golangwebframework
>
>2 如果出现directory "directory "/Users/xiezhen/Documents/codes/golib/src/github.com/goinaction/code/chapter3/words" is not using a known version control system" is not using a known version control system，说明这个目录没有版本信息，自己可以cd到对应的目录下，git status看下，只通过git init无法解决这个问题，这个依赖的包必须有已经commit的信息才可以

#### 2.1.2 总结
>godep save成功 == 依赖的代码包在$GOPATH中&依赖的代码包有已经提交过的commit版本信息
>
>如果依旧不成功，建议每一步debug下

### 2.2 godep restore
#### 2.2.1 场景
>如果我们已经pull下来了一个开源库，开源库中有Godeps文件夹
>
>执行godep restore之后会做两件事
>
>1 go get Godeps中依赖的代码包（如果本地已有就不执行这一步）
>
>2 cd到对应目录 git checkout 到对应的版本

#### 2.2.2 总结
>godep restore成功 == 当前目录下有正确的Godeps/Godeps.json && 依赖的代码库可以checkout到Godeps.json中的版本上
>
>如果依旧不成功，建议每一步debug下




