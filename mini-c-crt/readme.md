# 程序员的自我修养

## c runtime

# 运行方法
```
linux x86_64环境下 gcc

$ make
rm -rf ./*.o
rm -rf ./test
gcc -c -ggdb -fno-builtin -nostdlib entry.c stdio.c string.c stdlib.c
gcc -c -ggdb -fno-builtin -nostdlib ./test.c
ld -static -e crt_entry ./entry.o ./string.o ./stdio.o ./stdlib.o ./test.o -o ./test

$ ./test
15
123Hello World!
9000

//清理产出
$ make clean
rm -rf ./*.o
rm -rf ./test

```
