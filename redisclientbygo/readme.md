# redisbygo

## 1 背景
>熟悉redis协议，实现一个Redis Client demo。底层还是TCP连接
>
>Redis全称为:Remote Dictionary Server(远程数据服务)
>
>Redis 具体的通信协议见 https://redis.io/topics/protocol，中文版 http://redisdoc.com/topic/protocol.html

## 2 介绍
>redis/conn.go是具体实现

## 3 运行
>可以参照main.go进行redis的操作

```
>go run main.go
value is xiezhen
```

## 4 测试用例
>执行测试用例之前，请先将redis server ip进行替换，在dialDefaultServer()函数中

```
>cd redis
>go test . -v
=== RUN   TestWrite
--- PASS: TestWrite (0.00s)
=== RUN   TestRead
--- PASS: TestRead (0.00s)
=== RUN   TestCMD
--- PASS: TestCMD (0.12s)
PASS
ok  	github.com/xiezhenouc/golanglearn/redisclientbygo/redis	0.135s
```
