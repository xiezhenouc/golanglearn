#include "mini_c_crt.h"

int crt_io_init()
{
    return 1;
}

// 系统调用
int write(int fd, const void *buffer, size_t size)
{
    int ret = 0;
    __asm__ volatile ("movq $1, %%rax \n\t"
                      "movq %1, %%rdi \n\t"
                      "movq %2, %%rsi \n\t"
                      "movq %3, %%rdx \n\t"
                      "syscall \n\t"
                      :"=m"(ret)
                      :"m"(fd), "m"(buffer), "m"(size));
    return ret;
}

void putchar(char c)
{
    write(1, &c, 1);
}

void puts(const char *str)
{
    size_t len = strlen(str);
    write(1, str, len);
    putchar('\n');
}