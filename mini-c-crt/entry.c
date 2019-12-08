#include "mini_c_crt.h"

extern int main(int argc, char** argv);
void _exit(int);

// 异常处理函数
void die(char *msg)
{
    puts(msg);
    _exit(1);
}

/*
void die(char *msg)
{
    puts(msg);
    _exit(1);
}
*/

void crt_entry(void)
{
    int ret;
    int argc;
    char** argv;

    char * rbp_reg = 0;
    asm volatile("movq %%rbp, %0 \n":"=m"(rbp_reg));
    
    // argc argv
    argc = *(int *)(rbp_reg + 8);
    argv = (char **) (rbp_reg + 16);


    // heap初始化，链表简单实现
    if(!crt_heap_init()) {
        die("heap init failed!");
    }

    // io初始化
    if(!crt_io_init()) {
        die("IO init failed!");
    }

    // 用户main开始执行的地方
    ret = main(argc, argv);
    _exit(ret);
}

void _exit(int status)
{
    // 系统调用
    __asm__("movq $60, %%rax \n\t"
            "movq %0, %%rdi \n\t"
            "syscall \n\t"
            "hlt \n\t"/* Crash if somehow `exit' does return.	 */
            :: "g" (status)); /* input */
}
