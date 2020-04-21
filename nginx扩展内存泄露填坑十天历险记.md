# nginx扩展内存泄露填坑十天历险记

## 1 背景

互联网有很多技术流派，java php c#等倍受各大厂信赖。而php依赖的LNMP这一套框架，曾经辉煌过很长一段时间。

我所在的产品线中，就是依赖的LNMP这一套架构。但一直存在一个问题，就是内存泄露，经过各个前辈的观察和总结，发现可能是nginx。
另外还有业务的同学观察到泄露的点在某个 服务发现 nginx扩展上。
因为这个同学将大量不用的下游服务从这个扩展中摘除出来后，发现内存泄露的速度明显减慢了。

组内的大佬看到这个现象忍不了了，直接拉上我，一起准备把这个问题搞一把，节省内存!


## 2 问题定位过程

现象是nginx内存泄露，线下环境和生产环境都印证了这一点

### 2.1 首先，大佬直接将线上的一个实例内存dump出来一份，发现这块内存里面有很多ip和port。这个操作的过程我记录下，在5中。

思考了下，ip和port是服务发现拿到的结果，看着内存泄露好像确认和服务发现的nginx扩展有点相关。

然后，我们就开始看 服务发现的nginx扩展 c源码，刚开始，肯定是一头雾水。

### 2.2 我自己尝试用valgrind跑了一把，发现valgrind报的可能并不是我们想要的结果，valgrind，贴下命令

valgrind --tool=memcheck --leak-check=full -q --show-possibly-lost=no --log-file=leak.log ./bin/nginx -p . -c ./conf/nginx.conf

另外，nginx运行时daemon模式，运行完上述命令之后，给nginx发一个结束信号，才能看到结果

./bin/nginx -s stop

在当前目录的leak.log能看到对应的结果

### 2.3 源码编译改造
由于之前项目使用makefile和svn搞的，但是厂内已经禁止这种编译方式了，所以这块费了很大劲，最后，能自己编译了。

### 2.4 修改自己编的代码，逐步试错。这里写一个主要过程
观察：服务发现的扩展每20s更新下服务下面的ip和port，还有一个timeout是500s,这个值蛮重要

思路：这个服务发现的扩展本质是nginx和本地的某个端口通信，本地某个端口运行的是获取服务ip列表。

过程：和本地建立连接->发请求->收请求->发送配置请求->接收配置请求->所有工作都完成

观察：根据内存涨的时机和日志，发现应该是 收请求 和 接收配置请求 这俩造成的泄露

`修改点：尝试将receiver函数直接注释，发现不泄露了！！！`

思路：很明显，我们缩小了检查的范围，根据仔细观察，里面有一个update函数，好像很重要

`修改点：把update函数干掉，发现不泄露了！！！`

思路：路子很对呀，那就把update里面所有的可能泄露的地方都搞一遍不就成了，所以我和大佬一个字符一个字符的看。
把这个300行的函数来来回回看了n遍，然后找出来三四处我们觉得肯定泄露的点，

弯路：尝试我俩找到所有可能泄露的点，free掉这些点，这个过程持续了三四天，可是只有一丢丢的效果。
ngx_parse_url函数中有很多嵌套的新分配空间的代码，找不全。

思路：在这过程中，发现了一个pool size参数是32，好像有点用，调成3或者2，报pool不够用。
每个pool会记录一个时间，这个时间也不知道干啥的

`思路：准备弄明白，32个pool 500s pool记录 以及update_server的时机`

最终流程：每个服务会注册32个pool，这32个pool，会循环使用，通过 (i + 1) % size来循环。
500s的意思是，如果下一个pool使用之前，比较下500s，如果已经超过500s，reset pool。然后使用这个pool。调用update_server。unchange的话，再次使用这个pool。

`宝贵的流程图`

![nginx扩展内存泄露](https://raw.githubusercontent.com/xiezhenouc/golanglearn/master/%E5%9B%BE%E7%89%87%E8%AF%B4%E6%98%8E/nginx扩展内存泄露.png)

所以我们最后的思路，把ngx_reset_pool改为ngx_destroy_pool。因为update_servers中所有创建的空间，都是从这个pool出来的。

结果：生效了！！！

跟原作者：但是跟原作者沟通，他不接受这个解决方案，sad。
原因是，他觉得这个pool不应该被destory，可能是正在被使用中，直接destroy，可能会让进程直接退出。

```
我们的观点：有三个点
1 可以destroy，因为原来就已经reset了
2 unchange的逻辑中，destroy没有影响
3 change逻辑中，那么会用新创建的空间去替换老的空间。老的空间的东西500s后没有人再用
```

## 3 解决问题的核心关键点

1 最好能大概看到内存里面的东西，有个大方向
2 最好能有个本地开发的环境，大不了，二分查找法，快速定位到大的函数
3 观察函数的大流程，将流程吃透，每个看不懂的点可能都是解决问题的关键钥匙
4 不能瞎改，改的时候要说服自己为什么，最好讲给别人听一听。


## 4 经验教训

update_server函数内部耗费了很多时间，应该把更多精力留在大流程的探索上


## 5 学习到的知识
### 如何查看正在运行中的进程的内存信息

系统环境centos6u3

假如进程ID是5092，执行

```
$ cat /proc/5092/maps
00400000-00f69000 r-xp 00000000 fd:10 3408622                            /home/bae/local/nginx/bin/nginx
01169000-011c1000 rw-p 00b69000 fd:10 3408622                            /home/bae/local/nginx/bin/nginx
011c1000-01821000 rw-p 00000000 00:00 0
01821000-04dfa000 rw-p 00000000 00:00 0
0c180000-0c190000 r-xp 00000000 00:00 0
400a1000-400c1000 rw-p 00000000 00:00 0
4062c000-4064c000 rw-p 00000000 00:00 0
40913000-40933000 rw-p 00000000 00:00 0
40b8e000-40bae000 rw-p 00000000 00:00 0
40c13000-40c33000 rw-p 00000000 00:00 0
4126c000-4128c000 rw-p 00000000 00:00 0
412a5000-412c5000 rw-p 00000000 00:00 0
41320000-41340000 rw-p 00000000 00:00 0
41513000-41533000 rw-p 00000000 00:00 0
416cc000-416ec000 rw-p 00000000 00:00 0
41c5d000-41c7d000 rw-p 00000000 00:00 0
7faabe650000-7faabe936000 rw-p 00000000 00:00 0
7faabe936000-7faabe999000 r-xp 00000000 fd:10 4757504                    /home/opt/gcc-4.8.2.bpkg-r4/gcc-4.8.2.bpkg-r4/lib64/libssl.so.1.0.0
7faabe999000-7faabeb98000 ---p 00063000 fd:10 4757504                    /home/opt/gcc-4.8.2.bpkg-r4/gcc-4.8.2.bpkg-r4/lib64/libssl.so.1.0.0
7faabeb98000-7faabeba2000 rw-p 00062000 fd:10 4757504                    /home/opt/gcc-4.8.2.bpkg-r4/gcc-4.8.2.bpkg-r4/lib64/libssl.so.1.0.0
7faabeba2000-7faabebbe000 r-xp 00000000 fd:10 3409592                    /home/bae/local/nginx/lib/bd_sign64.so
7faabebbe000-7faabedbe000 ---p 0001c000 fd:10 3409592                    /home/bae/local/nginx/lib/bd_sign64.so
7faabedbe000-7faabedbf000 rw-p 0001c000 fd:10 3409592                    /home/bae/local/nginx/lib/bd_sign64.so
7faabedbf000-7faabeddc000 r-xp 00000000 fd:10 3409589                    /home/bae/local/nginx/lib/xxx.so
```

这些都是内存地址，这么看感觉很蒙圈呀，大佬就找到一个快速找到大块内存的方法

```
cat /proc/5092/maps >> d1.txt

```

来一个简单的python脚本，这个脚本的意思，是将内存块中的大于10m的内存地址都打印出来，当然这个10m是可以自己调整的。

```python
#encoding=utf-8

for line in open("d1.txt"):
    l = line.strip('\n').split()[0].split('-')
    d = int(l[1], 16) - int(l[0], 16)
    m = float(d)/1024/1024
    if m > 10:
        print m, line
```

结果如下所示，很明显，这个53m的很可疑

```
11.41015625 00400000-00f69000 r-xp 00000000 fd:10 3408622                            /home/bae/local/nginx/bin/nginx

53.87890625 01821000-04e02000 rw-p 00000000 00:00 0

300.0 7faac0d30000-7faad3930000 rw-s 00000000 00:04 1047728516                 /dev/zero (deleted)
```

gdb大法来了，通过gdb attach到具体的进程号上，把这块内存地址dump出来，ps：gdb版本 4.8

```
./gdb attach 5092
dump binary memory leak.log 0x01821000 0x04e02000
quit # 一定要quit，之前没quit，发现内存不涨了，原来是因为暂停了..尴尬
```

这样，在leak.log中，就能看到这50M里面存的是啥了。