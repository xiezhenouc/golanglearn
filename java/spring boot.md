# spring boot

## 1 背景
```
工作以来，自己使用的技术栈主要是L(linux)N(nginx)M(mysql)P(php)，之后开始接触golang。
在业务开发中，一般是基于某种语言的框架，进行业务的架构和开发。针对PHP、Golng和C++，自己用的都是公司内部的框架，在这其中，对Golang的框架是最熟悉的，自己剖析过这个业务框架的每一个环节，所以出了各种问题能够及时发现和解决。
Java在国内互联网应用非常多，生态非常好，有所耳闻。最近听到一个小伙伴讲起dubbo框架，感觉挺有意思。所以准备熟悉下Java的开发。多看看生态好的语言，会帮助自己更深刻的理解语言底层的设计。

准备的学习路线 spring boot => spring => dubbo => zookeeper ...
```

## 2 准备工作
```
操作系统 CentOS release 6.3
Java基础环境
下载页面 https://www.oracle.com/java/technologies/javase-downloads.html 
选择的版本 14.0.2 

Maven基础环境
下载页面 https://maven.apache.org/download.cgi
选择的版本 3.6.3

环境变量 /etc/profile
export JAVA_HOME=/home/work/opt/java/jdk-14.0.2
export PATH=$JAVA_HOME/bin:/home/work/opt/java/apache-maven-3.6.3/bin:$PATH
export CLASSPATH=.:$CLASSPATH

环境变量生效 source /etc/profile
验证 java javac mvn，查看版本号即可

$ java -version
java version "14.0.2" 2020-07-14
Java(TM) SE Runtime Environment (build 14.0.2+12-46)
Java HotSpot(TM) 64-Bit Server VM (build 14.0.2+12-46, mixed mode, sharing)

$ javac -version
javac 14.0.2

$ mvn -version
Apache Maven 3.6.3 (cecedd343002696d0abb50b32b541b8a6ba2883f)
Maven home: /home/work/opt/java/apache-maven-3.6.3
Java version: 14.0.2, vendor: Oracle Corporation, runtime: /home/work/.jumbo/opt/java/jdk-14.0.2
Default locale: en_US, platform encoding: UTF-8
OS name: "linux", version: "2.6.32_1-16-0-0_virtio", arch: "amd64", family: "unix"

```

## 3 maven源国内代理设置
各个语言都有管理依赖的方式，golang有go mod方式，java有mvn，还有gradle(还没研究过)
修改代理配置，待会下载依赖的时候健步如飞！！！不修改这个就是龟速！
```
修改conf/settings.xml，增加一个国内代理

    <mirror>
      <id>alimaven</id>
      <mirrorOf>central</mirrorOf>
      <name>aliyun maven</name>
      <url>http://maven.aliyun.com/nexus/content/groups/public/</url>
    </mirror>
```

## 4 spring boot

### 4.1 Spring 初始化器  https://start.spring.io/
这个就是生成一个初始化的压缩包，可以参考 https://spring.io/guides/gs/rest-service/ 中的 Starting with Spring Initializr 部分

点击Generate，即可得到压缩包。将这个压缩包放到服务器上

### 4.2 简单修改下main代码
```java
package com.example.restservice;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController // 控制器
@SpringBootApplication
public class RestServiceApplication {

    @RequestMapping("/") // 路由
    String home() {
        return "Hello Spring Boot !";
    }

	public static void main(String[] args) {
		SpringApplication.run(RestServiceApplication.class, args);
	}
}

``` 

### 4.3 修改端口

```
修改  src/main/resources/application.properties
spring默认端口是8080，但如果本机已经有了8080，则会端口冲突，在这个文件中增加
server.port=8084

这个点找了好久，感觉应该有个default提示的，这样会好很多
```

### 4.4 运行
```
执行 mvn spring-boot:run 即可

官网上 让运行  ./mvnw spring-boot:run ，但我这边一直卡住，也看不到错误日志，感觉做的不好
```

### 4.5 访问
```
访问 :8080/即可看到页面
```
