/*
 * 程序员的自我修养
 * 学习
 */

#ifndef MINI_C_CRT_H
#define MINI_C_CRT_H

#ifndef NULL
#define NULL (0)
#endif

#define X86_64 1
typedef unsigned long size_t;
typedef int FILE;

#define EOF (-1)

#define stdin ((FILE *)0)
#define stdout ((FILE *)1)
#define stderr ((FILE *)2)

//io
int crt_io_init(void);
int read(int fd, void *buffer, size_t size);
int write(int fd, const void *buffer, size_t size);
void puts(const char *str);
void putchar(char c);
int getchar(void);

//heap
int crt_heap_init(void);
void *malloc(size_t size);
void free(void *ptr);

//string
void itoa(int n, char s[]);
size_t strlen(const char *str);

#endif