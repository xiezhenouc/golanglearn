# 1 并发模型
## 1.1 并发与并行
```
摩尔定律，每年晶体管和电阻数量double

阿姆达尔定律，一个程序能从并行上获得性能提升的上限取决于有多少代码必须写成串行的

并发可能造成的问题
数据竞争，多个线程同时读写同一个变量，造成异常
原子性，每个操作是否是1或者0
临界区。把多个操作封装成一个临界区，只有一个线程能够进入
死锁。每个线程都在等待另外一个线程释放资源，导致形成一个环。
活锁。两个人迎面走向对方，一个人往左走，另外一个人也往左走，然后一个人往右走，另外一个人也往右走。看起来一直在运行，其实工作进度不会前进。
饥饿。有一个线程特别耗资源，其他所有线程都必须等待很久都运行不了。

并发和并行的区别
并发是同一时间应对多件事的能力
并行是同一时间做多件事的能力

并发是逻辑上具备同时处理多个任务的能力；并行是物理上同时执行多个任务。

Concurrency is a property of the code; parallelism is a property of the running program.

goroutine抽象于线程之上
```
## 1.2 什么是CSP
```
进程间通信
多线程+内存同步访问，通过加锁在大型项目中容易出现各种问题
goroutine = 线程
channel = mutex（内存同步访问）
```
# 2 什么是channel
```
goroutine用于并发执行任务
channel用于goroutine之前的同步和通信，是goroutine间的管道

不要通过共享内存来通信，而要通过通信来实现内存共享
```
## 2.1 channel实现CSP
```
chan T 		声明一个双向通道
chan<- t 	声明一个只用于发送的通道
<-chan T    声明一个只用于接收的通道

channel是引用类型，空值是nil，make初始化

缓存型vs非缓冲型
```
# 3 为什么要用channel
>channel+goroutine，业务开发简单

# 4 channel实现原理
```
chan的发送和接收，在编译期间转换成底层的发送和接收函数
不带缓冲的chan = 同步模式
带缓冲的chan = 异步模式
```
## 4.1 数据结构
```
type hchan struct {
	// chan里元素数量
	qcount   uint           // total data in the queue
	// chan 底层循环数组的长度
	dataqsiz uint           // size of the circular queue
	// 指向底层数组的指针
	// 只针对有缓冲的channel
	buf      unsafe.Pointer // points to an array of dataqsiz elements
	// chan中元素大小
	elemsize uint16
	// chan是否关闭
	closed   uint32
	// chan中元素的类型
	elemtype *_type // element type
	// 已发送元素在循环数组中的索引
	sendx    uint   // send index
	// 已接收元素在循环数组中的索引
	recvx    uint   // receive index
	// 等待接收的goroutine队列
	recvq    waitq  // list of recv waiters
	// 等待发送的goroutine队列
	sendq    waitq  // list of send waiters

	// 保护hchan所有的字段
	lock mutex
}

// 双向
type waitq struct {
	first *sudog
	last  *sudog
}
```
## 4.2 创建
```
	// 无缓冲channel
	ch := make(chan int)
	// 有缓冲channel
	ch1 := make(chan int, 5)
```
创建函数原型

```
func makechan(t *chantype, size int64) *hchan {
	elem := t.elem

	//异常检查
	...

	var c *hchan
	// 如果元素不含指针 or size大小为0（无缓冲channel）
	// 只进行一次内存分配
	if elem.kind&kindNoPointers != 0 || size == 0 {
		// Allocate memory in one call.
		// Hchan does not contain pointers interesting for GC in this case:
		// buf points into the same allocation, elemtype is persistent.
		// SudoG's are referenced from their owning thread so they can't be collected.
		// TODO(dvyukov,rlh): Rethink when collector can move allocated objects.
		// 如果元素类型不含指针，gc不会扫描chan中的元素
		// 申请内存的大小 = hchan结构的大小 + 元素大小 * 元素个数
		c = (*hchan)(mallocgc(hchanSize+uintptr(size)*elem.size, nil, true))
		// 有缓冲channel 而且 元素大小不为0 (struct{}大小为0)
		if size > 0 && elem.size != 0 {
			// 相对寻址
			c.buf = add(unsafe.Pointer(c), hchanSize)
		} else {
			// race detector uses this location for synchronization
			// Also prevents us from pointing beyond the allocation (see issue 9401).
			// 非缓冲 or 有缓冲&元素大小为0，直接从c开始地址开始算
			c.buf = unsafe.Pointer(c)
		}
	} else {
		// 两次内存分配，分开管理
		c = new(hchan)
		c.buf = newarray(elem, int(size))
	}
	c.elemsize = uint16(elem.size)
	c.elemtype = elem
	c.dataqsiz = uint(size)


	return c
}

```
## 4.3 接收
通过汇编找到接收函数入口，两种形式，一种带ok，另外一种不带
```
// entry points for <- c from compiled code
//go:nosplit
func chanrecv1(c *hchan, elem unsafe.Pointer) {
	chanrecv(c, elem, true)
}

//go:nosplit
func chanrecv2(c *hchan, elem unsafe.Pointer) (received bool) {
	_, received = chanrecv(c, elem, true)
	return
}
```
核心函数chanrecv
```
// chanrecv receives on channel c and writes the received data to ep.
// ep may be nil, in which case received data is ignored.
// If block == false and no elements are available, returns (false, false).
// Otherwise, if c is closed, zeros *ep and returns (true, false).
// Otherwise, fills in *ep with an element and returns (true, true).
// A non-nil ep must point to the heap or the caller's stack.
// 从c接收数据，将结果写在ep指向的地址上。
// ep为nil的时候，说明接收的数据是被忽略的。
// block=false，非阻塞型&无数据，返回(false, false)
// c关闭，将返回(true, false)
// 如果ep非空，应该指向堆区or调用者栈

func chanrecv(c *hchan, ep unsafe.Pointer, block bool) (selected, received bool) {
	// raceenabled: don't need to check ep, as it is always on the stack
	// or is new memory allocated by reflect.

	if debugChan {
		print("chanrecv: chan=", c, "\n")
	}

	// 如果是一个nil的channel
	if c == nil {
		// 如果不阻塞，直接返回
		if !block {
			return
		}
		// 否则，挂起 
		gopark(nil, nil, "chan receive (nil chan)", traceEvGoStop, 2)
		throw("unreachable")
	}

	// Fast path: check for failed non-blocking operation without acquiring the lock.
	//
	// After observing that the channel is not ready for receiving, we observe that the
	// channel is not closed. Each of these observations is a single word-sized read
	// (first c.sendq.first or c.qcount, and second c.closed).
	// Because a channel cannot be reopened, the later observation of the channel
	// being not closed implies that it was also not closed at the moment of the
	// first observation. We behave as if we observed the channel at that moment
	// and report that the receive cannot proceed.
	//
	// The order of operations is important here: reversing the operations can lead to
	// incorrect behavior when racing with a close.
	// 非阻塞情况下，快速检查到失败，不用获取锁，快速返回
	// 如果是非缓冲型，且发送队列为空
	// 如果是缓冲型，当前buffer中为空
	// 且channel当前是打开状态
	if !block && (c.dataqsiz == 0 && c.sendq.first == nil ||
		c.dataqsiz > 0 && atomic.Loaduint(&c.qcount) == 0) &&
		atomic.Load(&c.closed) == 0 {
		return
	}

	var t0 int64
	if blockprofilerate > 0 {
		t0 = cputicks()
	}


	// 加锁，临界区
	lock(&c.lock)

	// channel已经关闭，buffer中统计是0
	// 缓冲型，关闭+buf无元素
	// 非缓冲型，关闭
	if c.closed != 0 && c.qcount == 0 {
		if raceenabled {
			raceacquire(unsafe.Pointer(c))
		}
		unlock(&c.lock)
		if ep != nil {
			// 从一个已关闭的channel进行接收，且未忽略返回值
			// 那么接收的值是一个该类型的零值
			// typedmemclr根据类型清空ep相应地址的内存
			typedmemclr(c.elemtype, ep)
		}
		return true, false
	}

	// 等待发送队列不空，说明buf是满的，两种可能
	// 1.1 非缓冲型。直接内存拷贝。
	// 1.2 缓冲型，但是buf满了。接收循环数组头部的元素，并将发送者发送的元素放在循环数组尾部。
	if sg := c.sendq.dequeue(); sg != nil {
		// Found a waiting sender. If buffer is size 0, receive value
		// directly from sender. Otherwise, receive from head of queue
		// and add sender's value to the tail of the queue (both map to
		// the same buffer slot because the queue is full).
		// c:channel sg: sender goroutine 解锁回调函数 
		recv(c, sg, ep, func() { unlock(&c.lock) }, 3)
		return true, true
	}
	// 缓冲型，buf里有元素，可以正常接收
	if c.qcount > 0 {
		// Receive directly from queue
		// 直接从循环数组找到要接收的元素
		qp := chanbuf(c, c.recvx)
		if raceenabled {
			raceacquire(qp)
			racerelease(qp)
		}
		if ep != nil {
			// 代码里，没有忽略要接收的值，将qp指向的值复制到ep指向的位置
			// val <- ch
			typedmemmove(c.elemtype, ep, qp)
		}
		// qp位置数据清空
		typedmemclr(c.elemtype, qp)
		// recv index ++
		c.recvx++
		// 循环数组，如果到size，从0再次开始
		if c.recvx == c.dataqsiz {
			c.recvx = 0
		}
		// buf中个数减少一个
		c.qcount--
		// 解锁
		unlock(&c.lock)
		return true, true
	}

	// 非阻塞，解锁，返回
	if !block {
		unlock(&c.lock)
		return false, false
	}

	// 接下来是阻塞情况
	// no sender available: block on this channel.
	// 构造sudog
	gp := getg()
	mysg := acquireSudog()
	mysg.releasetime = 0
	if t0 != 0 {
		mysg.releasetime = -1
	}

	// 待接收数据的地址保存下来
	// No stack splits between assigning elem and enqueuing mysg
	// on gp.waiting where copystack can find it.
	mysg.elem = ep
	mysg.waitlink = nil
	gp.waiting = mysg
	mysg.g = gp
	mysg.selectdone = nil
	mysg.c = c
	gp.param = nil
	// 进入channel的等待队列
	c.recvq.enqueue(mysg)
	// 将当前goroutine挂起
	goparkunlock(&c.lock, "chan receive", traceEvGoBlockRecv, 3)

	// 被唤醒
	// someone woke us up
	if mysg != gp.waiting {
		throw("G waiting list is corrupted")
	}
	gp.waiting = nil
	if mysg.releasetime > 0 {
		blockevent(mysg.releasetime-t0, 2)
	}
	closed := gp.param == nil
	gp.param = nil
	mysg.c = nil
	releaseSudog(mysg)
	return true, !closed
}
```

```

// recv processes a receive operation on a full channel c.
// There are 2 parts:
// 1) The value sent by the sender sg is put into the channel
//    and the sender is woken up to go on its merry way.
// 2) The value received by the receiver (the current G) is
//    written to ep.
// For synchronous channels, both values are the same.
// For asynchronous channels, the receiver gets its data from
// the channel buffer and the sender's data is put in the
// channel buffer.
// Channel c must be full and locked. recv unlocks c with unlockf.
// sg must already be dequeued from c.
// A non-nil ep must point to the heap or the caller's stack.
func recv(c *hchan, sg *sudog, ep unsafe.Pointer, unlockf func(), skip int) {
	// 如果是非缓冲型的，直接拷贝
	if c.dataqsiz == 0 {
		if raceenabled {
			racesync(c, sg)
		}
		if ep != nil {
			// sender goroutine -> receiver goroutine
			// copy data from sender
			recvDirect(c.elemtype, sg, ep)
		}
	} else {
		// 队列满了，把队尾元素取出来，把发送者数据入队列
		// Queue is full. Take the item at the
		// head of the queue. Make the sender enqueue
		// its item at the tail of the queue. Since the
		// queue is full, those are both the same slot.
		qp := chanbuf(c, c.recvx)
		if raceenabled {
			raceacquire(qp)
			racerelease(qp)
			raceacquireg(sg.g, qp)
			racereleaseg(sg.g, qp)
		}
		// copy data from queue to receiver
		if ep != nil {
			typedmemmove(c.elemtype, ep, qp)
		}
		// copy data from sender to queue
		typedmemmove(c.elemtype, qp, sg.elem)
		c.recvx++
		if c.recvx == c.dataqsiz {
			c.recvx = 0
		}
		c.sendx = c.recvx // c.sendx = (c.sendx+1) % c.dataqsiz
	}
	sg.elem = nil
	gp := sg.g
	// 解锁
	unlockf()
	gp.param = unsafe.Pointer(sg)
	if sg.releasetime != 0 {
		sg.releasetime = cputicks()
	}
	// 唤醒接收的 goroutine. skip和打印栈相关，暂时不理会
	goready(gp, skip+1)
}
```
```
goroutine是用户态的协程，Go runtime进行管理
内核线程由OS进行管理，goroutine更加轻量
一个内核线程可以管理多个goroutine，当其中一个阻塞时候，内核线程可以调度其他的goroutine来运行，内核线程本身不会阻塞。
M:N 模型通常由三部分构成：M、P、G。
M 是内核线程，负责运行 goroutine；
P 是 context，保存 goroutine 运行所需要的上下文，它还维护了可运行（runnable）的 goroutine 列表；
G 则是待运行的 goroutine。
M 和 P 是 G 运行的基础。
```
## 4.4 发送
```
// 向队列发送3
ch <- 3
```

>通过汇编，可以找到，对应的函数为

```
// entry point for c <- x from compiled code
//go:nosplit
func chansend1(c *hchan, elem unsafe.Pointer) {
	chansend(c, elem, true, getcallerpc(unsafe.Pointer(&c)))
}

/*
 * generic single channel send/recv
 * If block is not nil,
 * then the protocol will not
 * sleep but return if it could
 * not complete.
 *
 * sleep can wake up with g.param == nil
 * when a channel involved in the sleep has
 * been closed.  it is easiest to loop and re-run
 * the operation; we'll see that it's now closed.
 */
func chansend(c *hchan, ep unsafe.Pointer, block bool, callerpc uintptr) bool {
	// 如果channel为nil
	if c == nil {
		// 非阻塞，直接返回false
		if !block {
			return false
		}
		gopark(nil, nil, "chan send (nil chan)", traceEvGoStop, 2)
		throw("unreachable")
	}

	if debugChan {
		print("chansend: chan=", c, "\n")
	}

	if raceenabled {
		racereadpc(unsafe.Pointer(c), callerpc, funcPC(chansend))
	}

	// Fast path: check for failed non-blocking operation without acquiring the lock.
	//
	// After observing that the channel is not closed, we observe that the channel is
	// not ready for sending. Each of these observations is a single word-sized read
	// (first c.closed and second c.recvq.first or c.qcount depending on kind of channel).
	// Because a closed channel cannot transition from 'ready for sending' to
	// 'not ready for sending', even if the channel is closed between the two observations,
	// they imply a moment between the two when the channel was both not yet closed
	// and not ready for sending. We behave as if we observed the channel at that moment,
	// and report that the send cannot proceed.
	//
	// It is okay if the reads are reordered here: if we observe that the channel is not
	// ready for sending and then observe that it is not closed, that implies that the
	// channel wasn't closed during the first observation.
	// 如果非阻塞，未关闭下：
	// 非缓冲型&等待接收goroutine为空，缓冲型&元素已经满了
	// 直接返回
	if !block && c.closed == 0 && ((c.dataqsiz == 0 && c.recvq.first == nil) ||
		(c.dataqsiz > 0 && c.qcount == c.dataqsiz)) {
		return false
	}

	var t0 int64
	if blockprofilerate > 0 {
		t0 = cputicks()
	}

	// 加锁
	lock(&c.lock)

	if c.closed != 0 {
		// 如果channel已经关闭
		unlock(&c.lock)
		panic(plainError("send on closed channel"))
	}

	// 如果等待接收队列中有元素。直接内存拷贝
	if sg := c.recvq.dequeue(); sg != nil {
		// Found a waiting receiver. We pass the value we want to send
		// directly to the receiver, bypassing the channel buffer (if any).
		send(c, sg, ep, func() { unlock(&c.lock) }, 3)
		return true
	}

	// 缓冲型，还有缓冲空间，将元素放到buf
	if c.qcount < c.dataqsiz {
		// Space is available in the channel buffer. Enqueue the element to send.
		qp := chanbuf(c, c.sendx)
		if raceenabled {
			raceacquire(qp)
			racerelease(qp)
		}
		typedmemmove(c.elemtype, qp, ep)
		c.sendx++
		if c.sendx == c.dataqsiz {
			c.sendx = 0
		}
		c.qcount++
		unlock(&c.lock)
		return true
	}

	if !block {
		unlock(&c.lock)
		return false
	}

	// Block on the channel. Some receiver will complete our operation for us.
	gp := getg()
	mysg := acquireSudog()
	mysg.releasetime = 0
	if t0 != 0 {
		mysg.releasetime = -1
	}
	// No stack splits between assigning elem and enqueuing mysg
	// on gp.waiting where copystack can find it.
	mysg.elem = ep
	mysg.waitlink = nil
	mysg.g = gp
	mysg.selectdone = nil
	mysg.c = c
	gp.waiting = mysg
	gp.param = nil
	c.sendq.enqueue(mysg)
	goparkunlock(&c.lock, "chan send", traceEvGoBlockSend, 3)

	// someone woke us up.
	if mysg != gp.waiting {
		throw("G waiting list is corrupted")
	}
	gp.waiting = nil
	if gp.param == nil {
		if c.closed == 0 {
			throw("chansend: spurious wakeup")
		}
		panic(plainError("send on closed channel"))
	}
	gp.param = nil
	if mysg.releasetime > 0 {
		blockevent(mysg.releasetime-t0, 2)
	}
	mysg.c = nil
	releaseSudog(mysg)
	return true
}
```

>详细看一下send

```
// send processes a send operation on an empty channel c.
// The value ep sent by the sender is copied to the receiver sg.
// The receiver is then woken up to go on its merry way.
// Channel c must be empty and locked.  send unlocks c with unlockf.
// sg must already be dequeued from c.
// ep must be non-nil and point to the heap or the caller's stack.
func send(c *hchan, sg *sudog, ep unsafe.Pointer, unlockf func(), skip int) {
	if raceenabled {
		if c.dataqsiz == 0 {
			racesync(c, sg)
		} else {
			// Pretend we go through the buffer, even though
			// we copy directly. Note that we need to increment
			// the head/tail locations only when raceenabled.
			qp := chanbuf(c, c.recvx)
			raceacquire(qp)
			racerelease(qp)
			raceacquireg(sg.g, qp)
			racereleaseg(sg.g, qp)
			c.recvx++
			if c.recvx == c.dataqsiz {
				c.recvx = 0
			}
			c.sendx = c.recvx // c.sendx = (c.sendx+1) % c.dataqsiz
		}
	}
	if sg.elem != nil {
		// sg是目标goroutine ep是要放入channel的元素
		sendDirect(c.elemtype, sg, ep)
		sg.elem = nil
	}
	gp := sg.g
	unlockf()
	gp.param = unsafe.Pointer(sg)
	if sg.releasetime != 0 {
		sg.releasetime = cputicks()
	}
	goready(gp, skip+1)
}



```

>不同goroutine之间的内存拷贝

```
>这里涉及到一个 goroutine 直接写另一个 goroutine 栈的操作，一般而言，不同 goroutine 的栈是各自独有的。而这也违反了 GC 的一些假设。为了不出问题，写的过程中增加了写屏障，保证正确地完成写操作。这样做的好处是减少了一次内存 copy：不用先拷贝到 channel 的 buf，直接由发送者到接收者，没有中间商赚差价，效率得以提高，完美。

// Sends and receives on unbuffered or empty-buffered channels are the
// only operations where one running goroutine writes to the stack of
// another running goroutine. The GC assumes that stack writes only
// happen when the goroutine is running and are only done by that
// goroutine. Using a write barrier is sufficient to make up for
// violating that assumption, but the write barrier has to work.
// typedmemmove will call bulkBarrierPreWrite, but the target bytes
// are not in the heap, so that will not help. We arrange to call
// memmove and typeBitsBulkBarrier instead.

func sendDirect(t *_type, sg *sudog, src unsafe.Pointer) {
	// src is on our stack, dst is a slot on another stack.

	// Once we read sg.elem out of sg, it will no longer
	// be updated if the destination's stack gets copied (shrunk).
	// So make sure that no preemption points can happen between read & use.
	dst := sg.elem
	typeBitsBulkBarrier(t, uintptr(dst), uintptr(src), t.size)
	memmove(dst, src, t.size)
}

func recvDirect(t *_type, sg *sudog, dst unsafe.Pointer) {
	// dst is on our stack or the heap, src is on another stack.
	// The channel is locked, so src will not move during this
	// operation.
	src := sg.elem
	typeBitsBulkBarrier(t, uintptr(dst), uintptr(src), t.size)
	memmove(dst, src, t.size)
}
```


## 4.5 关闭

# 5 channel进阶
## 5.1 发送和接收元素的本质
## 5.2 资源泄露
## 5.3 happened before
## 5.4 如何优雅地关闭channel
## 5.5 关闭的Channel仍能读出数据

# 6 channel应用
## 6.1 停止信号
## 6.2 任务定时
## 6.3 解耦生产方和消费方
## 6.4 控制并发数
