#include "mini_c_crt.h"

size_t strlen(const char *  str) 
{
    size_t cnt = 0;
    if(!str)
    {
        return 0;
    }
    for(;*str != '\0'; ++str)
    {
        cnt++;
    }
    return cnt;
}

void reverse(char s[])
{
    int i, j;
    char c;

    for(i = 0, j = strlen(s) - 1; i < j; i++, j--)
    {
        c = s[i];
        s[i] = s[j];
        s[j] = c;
    }
}

void itoa(int n, char s[])
{
    int i, sign;

    if((sign = n) < 0) 
    {
        n = -n;
    }
    i = 0;
    do{
        s[i++] = n % 10 + '0';
    }while((n /= 10) > 0);
    if (sign < 0)
    {
        s[i++] = '-';
    }
    s[i] = '\0';
    reverse(s);
}

