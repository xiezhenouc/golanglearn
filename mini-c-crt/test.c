#include "mini_c_crt.h"

static const char *str = "123Hello World!";

void test_itoa()
{
    int len = strlen(str);
    char len_str[20];
    itoa(len, len_str);
    puts(len_str);
    puts(str);
}

void test_malloc()
{
    int *p_int = (int *) malloc(sizeof(int));
    *p_int = 9000;
    char len_str[10];
    itoa(*p_int, len_str);
    puts(len_str);
    free(p_int);
}

int main(int argc, char * argv[])
{
    test_itoa();
    test_malloc();
    return 42;
}
