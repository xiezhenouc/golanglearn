# php connect vs pconnect

## 背景

>php 连接 redis 有两种方式 connect 和 pconnect，用一个例子，说明这两者的区别

## 测试
>nginx + php 
>nginx将一个路由反向代理到php文件上

```
location ~* ^/(redisconnect) {
    root  /home/work/tmp/mytest;
    rewrite ^/redisconnect$  /index.php break;

    #socket文件所在路径
    fastcgi_pass unix:/home/work/php/var/php-fastcgi.socket;
    #fastcgi配置
    include inc/fastcgi.conf.inc;
}
```

>index.php代码

```php
<?php
$redis = new Redis();
$redis->pconnect('127.0.0.1', 8379);

sleep(10);

echo 'hi';
?>
```

>查看redis情况

```
>redis-cli -h 127.0.0.1 -p 8379 info | grep connected_clients
connected_clients:1
```

>请求 http://127.0.0.1:80/redisconnect 接口，请求结束，还是2个

>但当kill fpm进程的时候，连接明显就掉了，只剩下1个


```
>redis-cli -h 127.0.0.1 -p 8379 info | grep connected_clients
connected_clients:2
```

>将pconnect换成connect，请求时候多一个，变为2个，请求结束，恢复原状，还是1个

```php
<?php
$redis = new Redis();
$redis->cconnect('127.0.0.1', 8379);

sleep(10);

echo 'hi';
?>
```

## 总结

>php的pconnect是长连接，保持对后端服务的tcp连接，并不释放，由php-fpm进程保持。


>connect是使用结束之后会释放

>所以使用pconnect代替connect，可以减少频繁建立redis连接的消耗。

## 补充文档
>pconnect文档说明 https://github.com/phpredis/phpredis#pconnect-popen

```
pconnect, popen
Description: Connects to a Redis instance or reuse a connection already established with pconnect/popen.
（复用连接）

The connection will not be closed on end of request until the php process ends. So be prepared for too many open FD's errors (specially on redis server side) when using persistent connections on many servers connecting to one redis server.
（除非php进程关闭，否则连接不会被断掉；有可能发生FD过多错误）

Also more than one persistent connection can be made identified by either host + port + timeout or host + persistent_id or unix socket + timeout.

Starting from version 4.2.1, it became possible to use connection pooling by setting INI variable redis.pconnect.pooling_enabled to 1.
（配置）

This feature is not available in threaded versions. pconnect and popen then working like their non persistent equivalents.
```

## 补充源代码
>可以明显看到connect和pconnect本质上调用的是同一个函数redis_connect，注意关键字persistent，这个代表是否复用连接
>
>https://github.com/phpredis/phpredis/blob/develop/redis.c

```c
PHP_METHOD(Redis, connect)
{
    if (redis_connect(INTERNAL_FUNCTION_PARAM_PASSTHRU, 0) == FAILURE) {
        RETURN_FALSE;
    } else {
        RETURN_TRUE;
    }
}
/* }}} */

/* {{{ proto boolean Redis::pconnect(string host, int port [, double timeout])
 */
PHP_METHOD(Redis, pconnect)
{
    if (redis_connect(INTERNAL_FUNCTION_PARAM_PASSTHRU, 1) == FAILURE) {
        RETURN_FALSE;
    } else {
        RETURN_TRUE;
    }
}
/* }}} */
```
```
PHP_REDIS_API int
redis_connect(INTERNAL_FUNCTION_PARAMETERS, int persistent)
{
...
}
```
