# golang 调度学习

## 1 线程切换
```golang
线程有三个状态
1 Waiting 如等待IO、系统调用，等待某件事情的发生
2 Runnable 只要给我CPU我就能运行
3 Executing 正在占用CPU执行

线程切换
操作系统用一个处于Runnable的线程将CPU上正处于Executing状态的线程换下来的过程。
新上场的线程会变为Executing。下场的会成为Runnable或者Waiting状态。

线程能做的事情分为 计算型 IO型
计算型切换前后需要先将计算任务暂停，损失比较大。
IO型，如果由于需要IO，切换掉CPU后，能充分利用到CPU
```

## 2 函数调用过程分析
```
函数栈帧的空间主要由函数参数和返回值、局部变量和被调用其它函数的参数和返回值空间组成。

Go plan9汇编通过栈传递函数参数和返回值。

调用子函数时候，先将参数在栈顶准备好，再执行Call指令，call=push下一条指令的IP + JMP到子函数

进入到子函数的栈帧，首先会将caller bp压栈，然后扩展栈顶。执行完此函数后，ret=会将之前push的ip弹出，并跳到上一个函数的下一条指令。
```

## 3 goroutine
```
goroutine是抽象于线程之上的概念

goroutine vs 线程
1 内存占用，goroutine初始内存2KB，可动态扩容，操作系统线程1MB。goroutine更轻量
2 创建和销毁，goroutine由runtime管理，用户级操作，消耗小；Thread创建、销毁内核级，线程池，消耗大
3 切换，goroutine 200ns，只需切换三个寄存器Program Counter, Stack Pointer and BP；Thread多 1000-1500ns。
```

## 4 M:N模型
```
Go runtime负责goroutine的生老病死，从创建到销毁，一手包办

Runtime在启动的时候，会申请M个线程，之后创建的N个goroutine都依附在这M个线程上执行，这就是M:N模型

![M:N模型说明](https://github.com/xiezhenouc/golanglearn/blob/master/%E5%9B%BE%E7%89%87%E8%AF%B4%E6%98%8E/MN%E6%A8%A1%E5%9E%8B.png)

在同一个时刻上，一个thread上只有一个goroutine上运行，当goroutine发生阻塞时，runtime会把goroutine调度走
```

## 5 什么是scheduler
```
Go程序的执行分为两层
Go runtime和Go program。运行时和用户程序。
运行时和操作系统打交道。
用户程序和运行时打交道。
运行时：内存管理 goroutine管理 channel通信 等作用
```

## 6 scheduler底层原理
```
操作系统看到的都是多线程执行。
多goroutines执行是runtime层面的。

有三个基础的结构体实现goroutine的调度，g m p

g代表一个goroutine，表示goroutine栈的一些字段，指示当前goroutine的状态，指示当前执行的指令地址 即PC

m代表内核线程，包含正在运行的goroutine字段

p代表虚拟的处理器，维护一个runnable状态的g队列，m需要获得p才能运行g

核心结构体sched，总览全局

调度目标：在内核线程上调度goroutines

核心思想
1 复用线程
2 限制同时运行的线程数为N，N等于CPU的核心数目
3 线程私有run queues，并且可以从其他线程stealing goroutine运行，线程阻塞后，可以将run queues传给其他线程

为啥需要p？
因为当前线程阻塞的时候，可以把任务调度给其他的线程

Go scheduler会启动一个后台线程sysmon，用来检查长时间运行的goroutine，将其调度到global goroutines，这是一个全局的run queues，以示惩罚

局限性
1 FIFO 没有优先级
2 没有抢占式
3 没有充分利用局部性
```

## 7 总览
```
物理核数 2 
逻辑核数 4 （超线程）

Go程序启动后，会给每个逻辑核心分配一个P，同时会给每一个P分配一个M（内核线程），这些内核线程仍然由 os scheduler 来调度

总结下，本地启动一个Go程序时候，会得到4个系统线程去执行任务，每个线程搭配一个P。

在初始化的时候，Go程序会有一个G，G会在M上得到运行，内核线程在cpu上调度，G在M上调度

全局可运行队列(GRQ) 存储global runnable goroutine
本地可运行队列(LRQ) 存储local runnable goroutine

![GPM](https://github.com/xiezhenouc/golanglearn/blob/master/%E5%9B%BE%E7%89%87%E8%AF%B4%E6%98%8E/GPM.png)

os scheduler 抢占式调度
go scheduler 协作式调度，但是由runtime调度，实际在用户程序层面，可以理解成抢占式

![gpm_workflow](https://github.com/xiezhenouc/golanglearn/blob/master/%E5%9B%BE%E7%89%87%E8%AF%B4%E6%98%8E/gpm_workflow.png)

```

## 8 调度时机

```
go 关键字
GC
系统调用
内存同步访问
```

## 9 work stealing
```
Go scheduler的职责就是将所有处于runnable的goroutines均匀分布在P上运行M

当一个P发现自己的LRQ中已经没有G时，会从其他P"偷来"一些G执行。精神感人肺腑！自己的工作干完了，主动为别人分担，这被称为work-stealing。

M:N模型，任何一个时刻，M个goroutines(G)分配到N个内核线程(M)上执行，这些内核线程跑在最多GOMAXPROCS的逻辑处理器P上。
每个M必须依附一个P，每个P同一时刻只能运行一个M。
如果P上的M阻塞了，其他的M依附在这个P上，继续执行这个P上的LRQ。

![gpm2] (https://github.com/xiezhenouc/golanglearn/blob/master/%E5%9B%BE%E7%89%87%E8%AF%B4%E6%98%8E/gpm2.png)

实际上，go scheduler每一轮调度要做的工作就是找到处于runnable的goroutine，并执行它。

找到顺序
1 检查LRQ
2 如果LRQ空了，找GRQ
3 如果GRQ空了，work-stealing 其他P的LRQ

![steal](https://github.com/xiezhenouc/golanglearn/blob/master/%E5%9B%BE%E7%89%87%E8%AF%B4%E6%98%8E/steal.png)

```

## 10 同步异步系统调用
```
当G需要进行系统调用时，根据调用的类型，它所依附的M有两种情况 同步和异步

对于同步的情况，M会被阻塞，进而从P上调度下来，G依然依附于M。之后新M依附于P，继续执行P上的LRQ。当G上的系统调用结束后，G再次加入LRQ中。

![同步](https://github.com/xiezhenouc/golanglearn/blob/master/%E5%9B%BE%E7%89%87%E8%AF%B4%E6%98%8E/同步.png)

对于异步的情况，M不会被阻塞，G的异步请求会被绑定到 network poller，等到系统调用结束，G才会重新回到P上。M没有被阻塞，可以继续执行P上的LRQ。

![异步](https://github.com/xiezhenouc/golanglearn/blob/master/%E5%9B%BE%E7%89%87%E8%AF%B4%E6%98%8E/异步.png)

```

