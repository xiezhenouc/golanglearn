# mysql 原理学习

## 1 背景
>1 从事互联网工作以来，发现MySQL的应用的领域真是非常多。如新浪博客/百度百科等，都用MySQL进行存储，从这个角度来讲，深入理解MySQL显得非常重要。
>
>2 平时的工作用到MySQL的地方也很多，但由于有专业的DBA人员来维护，所以关于性能自己并不是关心的非常多，因此自己对于MySQL并不是非常了解。
>
>3 MySQL本质上是存储数据的软件，对于存储&系统这个领域，自己了解较少。

>参考 http://blog.codinglabs.org/articles/theory-of-mysql-index.html

>参考 https://tech.meituan.com/2014/06/30/mysql-index.html

## 2 索引
>一般的应用系统，读写比例是10:1左右，因此提升读性能很重要。

>查询算法，包括 顺序查询 O(n)、二分查询、二叉树查找等。每种查找算法都只能应用于特定的数据结构上，二分查找要求数据有序，二叉树查找的数据必须要在二叉查找树上。

>但是，数据本身的组织结构不可能完全满足各种数据结构。比如一行数据有三列，要求每一列都是顺序存储的，是不现实的。

>所以，数据库系统还维护着满足特定查找算法的数据结构，这些数据结构以某种方式引用（指向）数据，这样就可以在这些数据结构上实现高级查找算法，这种数据结构叫做索引。

>但是，实际数据库系统中，用二叉树或者变种红黑树作为索引的几乎没有，原因是一个是深度，深度越深，I/O越多，导致读的性能下降严重。

## 3 B树
>每一个节点的内容为[key, data]，key即为索引列的某一行在此列的元素值，data是此行记录。

>举例说明，key为数量，data为其他元素值，如[5, (香蕉，21）]

name | 价格 |  数量  
-|-|-
香蕉 | 21 | 5 |
苹果 | 31 | 6 |
草莓 | 41 | 7 |

>示意图如图所示

![B tree说明](https://raw.githubusercontent.com/xiezhenouc/golanglearn/master/%E5%9B%BE%E7%89%87%E8%AF%B4%E6%98%8E/B树示意图.png)

>搜索的伪代码如下

```
BTree_Search(node, key) {
    if(node == null) return null;
    // 遍历node的每一个的key，i 0->n
    foreach(node.key)
    {
        // 找到了，直接返回
        if(node.key[i] == key) return node.data[i];
        
        // 如果当前的这个key[i]值 > 给定的key，说明在左边
        if(node.key[i] > key) return BTree_Search(point[i]->node);

        // 如果当前的这个key[i]值 > 给定的key，说明在右边，继续往右边找
    }
    return BTree_Search(point[i+1]->node);
}
data = BTree_Search(root, my_key);
```

>插入删除新的数据记录会破坏B-Tree的性质，因此在插入删除时，需要对树进行一个分裂、合并、转移等操作以保持B-Tree性质

## 4 B+树

>B Tree有许多变种，其中最常见的是B+Tree，例如MySQL就普遍使用B+Tree实现其索引结构。

>B+ Tree不同于 B Tree点

>1 内节点不存储data，只存储key

>2 叶子节点不存储指针。

>示意图

![B+ tree说明](https://raw.githubusercontent.com/xiezhenouc/golanglearn/master/%E5%9B%BE%E7%89%87%E8%AF%B4%E6%98%8E/B加树示意图.png)

>从图中可以看到叶节点大小和内节点大小可能不同

>B+ Tree比B Tree更适合外存储索引结构

>带有顺序访问指针的B+Tree，提升区间搜索的性能

![顺序访问 B+ tree说明](https://raw.githubusercontent.com/xiezhenouc/golanglearn/master/%E5%9B%BE%E7%89%87%E8%AF%B4%E6%98%8E/顺序访问B加树示意图.png)

## 5 为什么要用B Tree 和 B+ Tree，不用二叉搜索树or红黑树
>数据库存储的数据如果无法完整放到内存的时候，需要放到磁盘上

>访问速度对比 CPU > 内存 > 磁盘。


```
// from google 工程师Jeff Dean 分布式系统的ppt文档
L1 cache reference 读取CPU的一级缓存	0.5 ns
Branch mispredict(转移、分支预测)	5 ns
L2 cache reference 读取CPU的二级缓存	7 ns
Mutex lock/unlock 互斥锁\解锁	100 ns
Main memory reference 读取内存数据	100 ns
Compress 1K bytes with Zippy 1k字节压缩	10,000 ns
Send 2K bytes over 1 Gbps network 在1Gbps的网络上发送2k字节	20,000 ns
Read 1 MB sequentially from memory 从内存顺序读取1MB	250,000 ns
Round trip within same datacenter 从一个数据中心往返一次，ping一下	500,000 ns
Disk seek  磁盘搜索	10,000,000 ns 
Read 1 MB sequentially from network 从网络上顺序读取1兆的数据	10,000,000 ns
Read 1 MB sequentially from disk 从磁盘里面读出1MB	30,000,000 ns 
Send packet CA->Netherlands->CA 一个包的一次远程访问	150,000,000 ns
```

>很明显可以看到 磁盘IO的成本非常高，访问磁盘的成本（10,000,000 ns）大概是访问内存（100 ns）的十万倍左右

### 5.1 磁盘IO与预读
>磁盘读取数据靠的是机械运动，每次读取数据的时间=寻道时间+旋转时间+传输时间三个部分.

>寻道时间磁臂移动到指定磁道所需要的时间，主流磁盘一般在5ms以下

>旋转延迟就是我们经常听到的磁盘转速，如一个磁盘7200转，表示每分钟能转7200次，一秒钟转120次，旋转延迟是1/120/2=4.17ms。说明除以2的原因是因为磁盘是圆形，最远转一半圆即可，所以需要除以2.

>传输时间是指从磁盘读出数据或者将数据写入磁盘的过程，一般零点几毫秒，暂时忽略不计。

>所以，一次读取数据的时间=5ms+4.17ms=9.17ms。大约9ms。看起来好像还可以。但实际上一条CPU指令的执行时间是1ns，一秒钟能执行的指令数目是上亿，执行一条IO操作的时间可以执行几十万+CPU指令。

>数据库有成千上万条数据，每次都9ms，性能太差了。

>考虑到磁盘IO是非常高昂的操作，因此操作系统做了一些优化，当发生一次IO的时候，不光把当前磁盘地址的数据，还把相邻的磁盘地址的数据都加载到内存中，这个的理论依据是局部性原理。

>每次IO读取的数据我们称之为一页，具体的一页大小跟操作系统相关，可以通过`getconf PAGE_SIZE`命令查看操作系统页的大小，一般为4096，即为4k

>预读的长度一般为页的整倍数。主存和磁盘以页为单位交换数据。当程序要读取的数据不在主存中时，会触发一个缺页异常，此时系统会向磁盘发出读盘信号，磁盘会找到数据的起始位置并向后连续读取一页或几页载入内存中，然后异常返回，程序继续运行。

### 5.2 B Tree索引的性能分析
>从上文可知，查询一次需要访问h个节点，h即为树的高度。数据库系统的设计者巧妙利用了磁盘预读原理，将一个节点的大小设置为一个页，这样每个节点一次IO就可以完全载入

>其他细节:

>每次新建节点时，直接申请一个页的空间，这样就保证一个节点物理上也存储在一个页里，加之计算机存储分配都是按页对齐的，就实现了一个node只需一次I/O。

>B-Tree中一次检索最多需要h-1次I/O（根节点常驻内存），渐进复杂度为O(h)=O(logdN)。一般实际应用中，出度d是非常大的数字，通常超过100，因此h非常小（通常不超过3）。（为啥是100，因为一页是4KB，如果一个索引的大小是4个字节，还要包含指针4个or8个字节，因此大概可以包含500个，所以大概是百级）

>综上所述，用B-Tree作为索引结构效率是非常高的。

>而红黑树这种结构，h明显要深的多。由于逻辑上很近的节点（父子）物理上可能很远，无法利用局部性，所以红黑树的I/O渐进复杂度也为O(h)，效率明显比B-Tree差很多。

## 6 MySQL索引实现

### 6.1 MyISAM
>MyISAM引擎使用B+Tree作为索引结构，叶节点的data域存放的是数据记录的地址。下图是MyISAM索引的原理图

![MyISAM](https://raw.githubusercontent.com/xiezhenouc/golanglearn/master/%E5%9B%BE%E7%89%87%E8%AF%B4%E6%98%8E/MyISAM.png)

>注意：索引文件和数据文件是分离的

### 6.2 InnoDB

>虽然InnoDB也使用B+Tree作为索引结构，但具体实现方式却不同

> 1 InnoDB的数据文件本身就是索引文件。在InnoDB中，表数据文件本身就是按B+Tree组织的一个索引结构。

![InnoDB主键索引](https://raw.githubusercontent.com/xiezhenouc/golanglearn/master/%E5%9B%BE%E7%89%87%E8%AF%B4%E6%98%8E/InnoDB主键索引.png)


> 这棵树的叶节点data域保存了完整的数据记录。这个索引的key是数据表的主键，因此InnoDB表数据文件本身就是主索引。

> 2 InnoDB的辅助索引data域存储相应记录主键的值而不是地址。换句话说，InnoDB的所有辅助索引都引用主键作为data域。

![InnoDB辅助索引](https://raw.githubusercontent.com/xiezhenouc/golanglearn/master/%E5%9B%BE%E7%89%87%E8%AF%B4%E6%98%8E/InnoDB辅助索引.png)

>辅助索引搜索需要检索两遍索引：首先检索辅助索引获得主键，然后用主键到主索引中检索获得记录。

## 7 索引使用策略及优化

### 7.1 最左前缀原理
>联合索引(a, b, c)，则从最左边开始匹配，mysql会一直向右匹配直到遇到范围查询(>、<、between、like)就停止匹配。

>结构
![联合索引](https://raw.githubusercontent.com/xiezhenouc/golanglearn/master/%E5%9B%BE%E7%89%87%E8%AF%B4%E6%98%8E/联合索引.png)

>只有一棵索引树

### 7.2 索引选择性与前缀索引
>索引的选择性（Selectivity），是指不重复的索引值（也叫基数，Cardinality）与表记录数（#T）的比值，显然选择性的取值范围为(0, 1]，选择性越高的索引价值越大。

### 7.3 优化
#### 7.3.1 查询优化神器 - explain命令
>0.先运行看看是否真的很慢，注意设置SQL_NO_CACHE 

>1.where条件单表查，锁定最小返回记录表。这句话的意思是把查询语句的where都应用到表中返回的记录数最小的表开始查起，单表每个字段分别查询，看哪个字段的区分度最高 

>2.explain查看执行计划，是否与1预期一致（从锁定记录较少的表开始查询） 

>3.order by limit 形式的sql语句让排序的表优先查 

>4.了解业务方使用场景 

>5.加索引时参照建索引的几大原则 

>6.观察结果，不符合预期继续从0分析

#### 7.3.2 mysqldumpslow
>1.配置慢查询

```
mysql> show variables like "%query%" ;
```
>slow_query_log ：是否开启慢查询，ON 开启，OFF关闭

```
slow-query-log = on  #开启MySQL慢查询功能
slow_query_log_file = /data/mysql/127-slow.log  #设置MySQL慢查询日志路径
long_query_time = 5  #修改为记录5秒内的查询，默认不设置此参数为记录10秒内的查询
log-queries-not-using-indexes = on  #记录未使用索引的查询
```
>2.查看慢查询日志

```
#mysqldumpslow /var/lib/mysql/mysql-slow.log

主要功能是, 统计不同慢sql的
出现次数(Count),
执行最长时间(Time),
累计总耗费时间(Time),
等待锁的时间(Lock),
发送给客户端的行总数(Rows),
扫描的行总数(Rows),
用户以及sql语句本身(抽象了一下格式, 比如 limit 1, 20 用 limit N,N 表示).
```
>3 再用expain分析

```
性能从最好到最差：system、const、eq_reg、ref、range、index和ALL
```


## 8 InnoDB的主键选择与插入优化
>在使用InnoDB存储引擎时，如果没有特别的需要，请永远使用一个与业务无关的自增字段作为主键。

>如果表使用自增主键，那么每次插入新的记录，记录就会顺序添加到当前索引节点的后续位置，当一页写满，就会自动开辟一个新的页。

>这样就会形成一个紧凑的索引结构，近似顺序填满。由于每次插入时也不需要移动已有数据，因此效率很高，也不会增加很多开销在维护索引上。

>如果使用非自增主键（如果身份证号或学号等），由于每次插入主键的值近似于随机，因此每次新纪录都要被插到现有索引页得中间某个位置。此时MySQL不得不为了将新记录插到合适位置而移动数据，甚至目标页面可能已经被回写到磁盘上而从缓存中清掉，此时又要从磁盘上读回来，这增加了很多开销，同时频繁的移动、分页操作造成了大量的碎片，得到了不够紧凑的索引结构

## 9 总结

本文总结了MySQL相关的一些知识点。
任何数据库层面的优化都抵不上应用系统的优化，同样是MySQL，可以用来支撑Google/FaceBook/Taobao应用，但可能连你的个人网站都撑不住。

