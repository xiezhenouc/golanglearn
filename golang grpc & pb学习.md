# golang grpc 学习

## 1 依赖的基础环境
>需要依赖protobuf，访问https://github.com/protocolbuffers/protobuf/releases  找到合适版本
>解压，执行bin目录下的protoc二进制，查看是否可以用
>如果可用，执行`ln protoc /usr/bin/protoc`，这样在任意目录均可用了

```
go get google.golang.org/grpc
go get google.golang.org/genproto
go get github.com/golang/protobuf
go install github.com/golang/protobuf/protoc-gen-go
```

>中间还可能依赖`google.golang.org`下面的包,如果被墙，请查询对应的github.com的mirror，然后放到对应的$GOPATH目录下即可

## 2 Protoltbuf文件生成
>在$GOPATH下建立一个myprotobuf的文件夹，在该文件夹下新建myprotobuf.proto文件，内容如下

```
syntax = "proto3";

package myprotobuf;

// 某个服务
service Ocean {
    // 发送一个hi到对方
    rpc SayHi (HiRequest) returns (HiReply) {}
}

// 请求数据
message HiRequest {
    int32 reqid = 1;
    string reqname = 2;
    string reqmsg = 3;
}

// 应答数据
message HiReply {
    int32 replyid = 1;
    string replyname = 2;
    string replymsg = 3;
}
```

>在该目录下执行`protoc --go_out=plugins=grpc:. *.proto`，（注意，需要已经安装完毕 protoc-gen-go）
>会看到下面生成myprotobuf.pb.go文件。

## 3 client
>在自己的工程目录下新建client文件夹，在该文件夹下新建client.go文件，内容如下

```
package main

import (
	"context"
	"log"
	"strconv"

	"google.golang.org/grpc"
	pb "myprotobuf"
)

const (
	address = "localhost:50051"
)

func main() {
	// 1 发起tcp连接
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	// 2 基于tcp连接 创建一个客户端
	c := pb.NewOceanClient(conn)

	// 3 创建一个context
	ctx := context.Background()

	i := new(int64)
	for {
		// 4 请求数据封装
		name := strconv.FormatInt(*i, 10)
		req := &pb.HiRequest{
			Reqid:   int32(*i),
			Reqname: name + " name",
			Reqmsg:  name + " msg",
		}

		// 5 调用远程方法
		r, err := c.SayHi(ctx, req)
		if err != nil {
			log.Fatalf("could not greet: %v", err)
		}

		// 6 打印返回结果
		log.Printf("Greeting: %v", r)
		*i += 1
	}
}
```

## 4 server
>在自己的工程目录下新建server文件夹，在该文件夹下新建server.go文件，内容如下

```
package main

import (
	"context"
	"log"
	"net"

	"google.golang.org/grpc"
	pb "myprotobuf"
)

const (
	port = ":50051"
)

type server struct{}

// 实现pb.go中定义的接口
func (s *server) SayHi(ctx context.Context, in *pb.HiRequest) (*pb.HiReply, error) {
	log.Printf("Received: %v", in.Reqid)
	log.Printf("Received: %v", in.Reqname)
	log.Printf("Received: %v", in.Reqmsg)
	reply := &pb.HiReply{
		Replyid:   in.Reqid,
		Replyname: "replyname",
		Replymsg:  "replymsg",
	}
	return reply, nil
}

func main() {
	// 1 开启tcp server
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// 2 new server
	s := grpc.NewServer()

	// 3 绑定定义的结构体
	pb.RegisterOceanServer(s, &server{})

	// 4 启动tcp server，监听请求
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

```

## 5
> 启动server ，然后启动client，可以看到相互之间的通信

## 6 分析
>这样开发的好处是，客户端和服务端只用通过proto文件沟通即可，接下来，client和server各自开发各自的，调试成本低。
