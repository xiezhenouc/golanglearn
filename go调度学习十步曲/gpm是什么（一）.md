# gpm是什么

```
g p m是Go调度器的三个核心组件，各司其职，又相互配合，Go调度器得以高效运转，这也是Go天然支持高并发的内在动力。

```

## 1 G是什么
```golang
goroutine的首字母即为G。保存goroutine的状态信息和cpu一些寄存器的值，比如IP寄存器，轮到本goroutine执行时，CPU知道从哪一条指令开始执行

当goroutine被调离CPU时，调度器负责把CPU寄存器的值保存在g对象的成员变量中

当goroutine被调度起来运行时，调度器又负责把g对象的成员变量中保存的寄存器的值再恢复到CPU的寄存器中

g源码

基础机构之stack
// Stack describes a Go execution stack.
// The bounds of the stack are exactly [lo, hi),
// with no implicit data structures on either side.
// 栈的范围 [low, high)
type stack struct {
    // 低地址，栈顶
    lo uintptr
    // 高地址，栈底
	hi uintptr
}

基础结构之buf
// goroutine运行时，光有栈不行，还得有pc  sp等寄存器
type gobuf struct {
	// The offsets of sp, pc, and g are known to (hard-coded in) libmach.
	//
	// ctxt is unusual with respect to GC: it may be a
	// heap-allocated funcval so write require a write barrier,
	// but gobuf needs to be cleared from assembly. We take
	// advantage of the fact that the only path that uses a
	// non-nil ctxt is morestack. As a result, gogo is the only
	// place where it may not already be nil, so gogo uses an
	// explicit write barrier. Everywhere else that resets the
    // gobuf asserts that ctxt is already nil.
    // rsp寄存器的值
    sp   uintptr
    // rip寄存器的值
    pc   uintptr
    // 指向goroutine
    g    guintptr
    // ctxt对gc来说时不同寻常的。可能是一个堆分配函数，所以write需要写屏障
    // 但是gobuf需要从程序集中清除。
    // 我们达成这样一个共识，非nil的ctxt只有在more stack时使用。其余情况均为nil。
    ctxt unsafe.Pointer // this has to be a pointer so that gc scans it
    // 保存系统调用的返回值
	ret  sys.Uintreg
	lr   uintptr
	bp   uintptr // for GOEXPERIMENT=framepointer
}

type g struct {
	// Stack parameters.
	// stack describes the actual stack memory: [stack.lo, stack.hi).
	// stackguard0 is the stack pointer compared in the Go stack growth prologue.
	// It is stack.lo+StackGuard normally, but can be StackPreempt to trigger a preemption.
	// stackguard1 is the stack pointer compared in the C stack growth prologue.
	// It is stack.lo+StackGuard on g0 and gsignal stacks.
	// It is ~0 on other goroutine stacks, to trigger a call to morestackc (and crash).
    // goroutine使用的栈
    stack       stack   // offset known to runtime/cgo
    // stackguard0 go 栈扩张 比较的栈指针 抢占标识
    stackguard0 uintptr // offset known to liblink
    // stackguard1 c 栈扩张 比较的栈指针 抢占标识
	stackguard1 uintptr // offset known to liblink

	_panic         *_panic // innermost panic - offset known to liblink
    _defer         *_defer // innermost defer
    // 当前与g绑定的m
    m              *m      // current m; offset known to arm liblink
    // goroutine的运行现场
	sched          gobuf
	syscallsp      uintptr        // if status==Gsyscall, syscallsp = sched.sp to use during gc
	syscallpc      uintptr        // if status==Gsyscall, syscallpc = sched.pc to use during gc
	stktopsp       uintptr        // expected sp at top of stack, to check in traceback
    // wakeup时传入的现场
    param          unsafe.Pointer // passed parameter on wakeup
	atomicstatus   uint32
	stackLock      uint32 // sigprof/scang lock; TODO: fold in to atomicstatus
    goid           int64
    // g 被阻塞之后的近似时间
    waitsince      int64  // approx time when the g become blocked
    // g 被阻塞的原因
    waitreason     string // if status==Gwaiting
    // 指向全局队列的下一个g
    schedlink      guintptr
    // 抢占调度标识，为true时，stackguard0 = stackpreempt
	preempt        bool     // preemption signal, duplicates stackguard0 = stackpreempt
	paniconfault   bool     // panic (instead of crash) on unexpected fault address
	preemptscan    bool     // preempted g does scan for gc
	gcscandone     bool     // g has scanned stack; protected by _Gscan bit in status
	gcscanvalid    bool     // false at start of gc cycle, true if G has not run since last scan; TODO: remove?
	throwsplit     bool     // must not split stack
	raceignore     int8     // ignore race detection events
    sysblocktraced bool     // StartTrace has emitted EvGoInSyscall about this goroutine
    // syscall返回之后的cpu ticks，用了做tracing
	sysexitticks   int64    // cputicks when syscall has returned (for tracing)
	traceseq       uint64   // trace event sequencer
    tracelastp     puintptr // last P emitted an event for this goroutine
    // 如果调用了LockOsThread，那么这个g会绑定到某个m上
	lockedm        *m
	sig            uint32
	writebuf       []byte
	sigcode0       uintptr
	sigcode1       uintptr
    sigpc          uintptr
    // 创建该goroutine语句的指令地址
    gopc           uintptr // pc of go statement that created this goroutine
    // goroutine函数的指令地址
	startpc        uintptr // pc of goroutine function
	racectx        uintptr
	waiting        *sudog         // sudog structures this g is waiting on (that have a valid elem ptr); in lock order
	cgoCtxt        []uintptr      // cgo traceback context
    labels         unsafe.Pointer // profiler labels
    // time.Sleep缓存的定时器
	timer          *timer         // cached timer for time.Sleep

	// Per-G GC state

	// gcAssistBytes is this G's GC assist credit in terms of
	// bytes allocated. If this is positive, then the G has credit
	// to allocate gcAssistBytes bytes without assisting. If this
	// is negative, then the G must correct this by performing
	// scan work. We track this in bytes to make it fast to update
	// and check for debt in the malloc hot path. The assist ratio
	// determines how this corresponds to scan work debt.
	gcAssistBytes int64
}
```

## 2 M是什么
```golang
machine的首字母，即为M。代表一个工作线程 or 系统线程。G需要调度到M上才能运行，M是真正工作的人。
M保存了M自身使用的栈信息，当前正在M上执行的G信息，与之绑定的P信息。

当M没有工作可做的时候，在休眠之前，会"自旋"地来找活干。
检查全局队列
查看netdisk poller
试图执行gc
或者"偷"工作

// m代表工作线程
type m struct {
    // 记录工作线程使用的栈信息，在执行调度代码时需要使用
    // 执行用户goroutine代码时，使用用户自己的goroutine栈，因此调度会发生栈的切换
	g0      *g     // goroutine with scheduling stack
	morebuf gobuf  // gobuf arg to morestack
	divmod  uint32 // div/mod denominator for arm - known to liblink

	// Fields not known to debuggers.
	procid        uint64     // for debuggers, but offset not hard-coded
	gsignal       *g         // signal-handling g
    sigmask       sigset     // storage for saved signal mask
    // 通过tls结构体实现m与工作线程的绑定
    // 这里是线程本地存储
	tls           [6]uintptr // thread-local storage (for x86 extern register)
    mstartfn      func()
    // 指向正在运行的goroutine对象
	curg          *g       // current running goroutine
    caughtsig     guintptr // goroutine running during fatal signal
    // 当前工作线程绑定的p
	p             puintptr // attached p for executing go code (nil if not executing go code)
	nextp         puintptr
	id            int32
	mallocing     int32
    throwing      int32
    // 如果preemptoff!= ""，保持当前goroutine始终在此m上运行
	preemptoff    string // if != "", keep curg running on this m
	locks         int32
	softfloat     int32
	dying         int32
	profilehz     int32
    helpgc        int32
    // 为true时表示当前m处于自旋状态，正在从其他线程偷工作
    spinning      bool // m is out of work and is actively looking for work
    // m正阻塞在 note上
    blocked       bool // m is blocked on a note
    // m正在执行写屏障
	inwb          bool // m is executing a write barrier
	newSigstack   bool // minit on C thread called sigaltstack
    printlock     int8
    // 正在执行cgo调用
	incgo         bool // m is executing a cgo call
    fastrand      uint32
    // cgo调用总计数
	ncgocall      uint64      // number of cgo calls in total
	ncgo          int32       // number of cgo calls currently in progress
	cgoCallersUse uint32      // if non-zero, cgoCallers in use temporarily
    cgoCallers    *cgoCallers // cgo traceback if crashing in cgo call
    // 没有goroutine需要运行时，工作线程睡眠在这个park成员上
    // 其他线程通过这个park唤醒该工作线程
    park          note
    // 记录所有工作线程的链表
	alllink       *m // on allm
	schedlink     muintptr
	mcache        *mcache
	lockedg       *g
	createstack   [32]uintptr // stack that created this thread.
	freglo        [16]uint32  // d[i] lsb and f[i]
	freghi        [16]uint32  // d[i] msb and f[i+16]
	fflag         uint32      // floating point compare flags
    locked        uint32      // tracking for lockosthread
    // 正在等待锁的下一个m
	nextwaitm     uintptr     // next m waiting for lock
	needextram    bool
	traceback     uint8
	waitunlockf   unsafe.Pointer // todo go func(*g, unsafe.pointer) bool
	waitlock      unsafe.Pointer
	waittraceev   byte
	waittraceskip int
	startingtrace bool
    syscalltick   uint32
    // 工作线程ID
	thread        uintptr // thread handle

	// these are here because they are too large to be on the stack
	// of low-level NOSPLIT functions.
	libcall   libcall
	libcallpc uintptr // for cpu profiler
	libcallsp uintptr
	libcallg  guintptr
	syscall   libcall // stores syscall parameters on windows

	mOS
}

```

## 3 P是什么 
```golang
取processor的首字母，为M的执行提供"上下文"，保存M执行G的一些资源，例如本地可运行G队列，memory cache等

一个M只有绑定P才能执行goroutine，当M被阻塞时，整个P会被传递给其他M，或者说整个P被接管

// p保存go运行时所必须的资源
type p struct {
	lock mutex

    // 在allp中的索引
	id          int32
	status      uint32 // one of pidle/prunning/...
    link        puintptr
    // 每次scheduler call后加一
    schedtick   uint32     // incremented on every scheduler call
    // 每次system call后加一
    syscalltick uint32     // incremented on every system call
    // sysmon线程记录被监控p的系统调用时间和运行时间
    sysmontick  sysmontick // last tick observed by sysmon
    // 指向绑定的m，如果是idle的话，此值nil
	m           muintptr   // back-link to associated m (nil if idle)
	mcache      *mcache
	racectx     uintptr

	deferpool    [5][]*_defer // pool of available defer structs of different sizes (see panic.go)
	deferpoolbuf [5][32]*_defer

	// Cache of goroutine ids, amortizes accesses to runtime·sched.goidgen.
	goidcache    uint64
	goidcacheend uint64

    // Queue of runnable goroutines. Accessed without lock.
    // local runnable queue 不用通过锁即可访问
    // 队首
    runqhead uint32
    // 队尾
    runqtail uint32
    // 使用循环数组实现循环队列
	runq     [256]guintptr
	// runnext, if non-nil, is a runnable G that was ready'd by
	// the current G and should be run next instead of what's in
	// runq if there's time remaining in the running G's time
	// slice. It will inherit the time left in the current time
	// slice. If a set of goroutines is locked in a
	// communicate-and-wait pattern, this schedules that set as a
	// unit and eliminates the (potentially large) scheduling
	// latency that otherwise arises from adding the ready'd
    // goroutines to the end of the run queue.
    // runnext 非空时，代表的是一个 runnable 状态的 G，
    // 这个 G 被 当前 G 修改为 ready 状态，相比 runq 中的 G 有更高的优先级。
    // 如果当前 G 还有剩余的可用时间，那么就应该运行这个 G
    // 运行之后，该 G 会继承当前 G 的剩余时间
	runnext guintptr

    // Available G's (status == Gdead)
    // 空闲的g
	gfree    *g
	gfreecnt int32

	sudogcache []*sudog
	sudogbuf   [128]*sudog

	tracebuf traceBufPtr

	// traceSweep indicates the sweep events should be traced.
	// This is used to defer the sweep start event until a span
	// has actually been swept.
	traceSweep bool
	// traceSwept and traceReclaimed track the number of bytes
	// swept and reclaimed by sweeping in the current sweep loop.
	traceSwept, traceReclaimed uintptr

	palloc persistentAlloc // per-P to avoid mutex

	// Per-P GC state
	gcAssistTime     int64 // Nanoseconds in assistAlloc
	gcBgMarkWorker   guintptr
	gcMarkWorkerMode gcMarkWorkerMode

	// gcw is this P's GC work buffer cache. The work buffer is
	// filled by write barriers, drained by mutator assists, and
	// disposed on certain GC state transitions.
	gcw gcWork

	runSafePointFn uint32 // if 1, run sched.safePointFn at next safe point

	pad [sys.CacheLineSize]byte
}

```

## 总结

```
GPM三足鼎立 共同成就go scheduler

G需要在M上运行，M依赖P提供的资源，P则持有待运行的G。

M从与它绑定的P的本地队列获取可运行的G，也会从network poller获取可运行的G，还会从其他P偷G。

```
![三足鼎立](https://github.com/xiezhenouc/golanglearn/blob/master/%E5%9B%BE%E7%89%87%E8%AF%B4%E6%98%8E/三足鼎立.png)
