# redis pool client 
## 1 内容
>实现了redis client和简单的redis pool，pool的内部实现是双向链表

## 2 文件说明
> go run main.go 执行简单例子

|____main.go pool的简单使用
|____Readme.md
|____redis 
| |____conn.go        // conn实现
| |____conn_test.go
| |____pool.go        // pool实现
| |____pool_test.go   
| |____redis.go       // Conn接口

## 3 测试用例运行

```
cd redis
go test . -v

=== RUN   TestWrite
--- PASS: TestWrite (0.00s)
=== RUN   TestRead
--- PASS: TestRead (0.00s)
=== RUN   TestCMD
--- PASS: TestCMD (0.39s)
=== RUN   TestPoolReuse
--- PASS: TestPoolReuse (0.23s)
=== RUN   TestMaxIdle
--- PASS: TestMaxIdle (0.43s)
=== RUN   TestPoolClose
--- PASS: TestPoolClose (0.03s)
=== RUN   TestPoolTimeout
--- PASS: TestPoolTimeout (0.15s)
=== RUN   TestPoolMaxActive
--- PASS: TestPoolMaxActive (0.02s)
=== RUN   TestWaitPool
--- PASS: TestWaitPool (1.05s)
=== RUN   TestWaitPoolClose
--- PASS: TestWaitPoolClose (1.16s)
	pool_test.go:306: redispoolclient by go: pool is closed
	pool_test.go:306: redispoolclient by go: pool is closed
	pool_test.go:306: redispoolclient by go: pool is closed
	pool_test.go:306: redispoolclient by go: pool is closed
	pool_test.go:306: redispoolclient by go: pool is closed
	pool_test.go:306: redispoolclient by go: pool is closed
	pool_test.go:306: redispoolclient by go: pool is closed
	pool_test.go:306: redispoolclient by go: pool is closed
	pool_test.go:306: redispoolclient by go: pool is closed
	pool_test.go:306: redispoolclient by go: pool is closed
PASS
ok  	github.com/xiezhenouc/golanglearn/redispoolclientbygo/redis	3.474s
```