#include "mini_c_crt.h"

// heap块是否被使用的标志
#define HEAP_BLOCK_FREE  0xABABABABABABABAB // 8 bytes
#define HEAP_BLOCK_USED  0xCDCDCDCDCDCDCDCD

// header头
typedef struct _heap_header
{
    unsigned long long type; // 是否被使用
    size_t size; // 大小
    struct _heap_header* next; // 双向链表
    struct _heap_header* prev; // 双向链表
}heap_header;

// 基地址+offset
#define ADDR_ADD(a, o) (((char *)(a)) + o)
// header大小
#define HEADER_SIZE (sizeof(heap_header))

// 全局变量，所有header管理的起点
heap_header *list_head = NULL;

// 系统调用
static long brk(void *addr) 
{
    long ret = 0;
    __asm__ volatile(
        "movq $12, %%rax \n\t"
        "movq %1, %%rdi \n\t"
        "syscall \n\t"
        "movq %%rax, %0 \n\t"
        : "=r" (ret)
        : "m" (addr)
        );
}


int crt_heap_init()
{
    void *base = NULL;
    heap_header *header = NULL;
    // 32 MB heap size
    size_t heap_size = 1024 * 1024 * 32;

    // 创建heap空间 (base, end)
    base = (void*) brk(0);
    void *end = ADDR_ADD(base, heap_size);
    end = (void *) brk(end);
    
    if (!end) {
        return 0;
    }
    header = (heap_header*) base;

    // 链表的第一个节点
    header->size = heap_size;
    header->type = HEAP_BLOCK_FREE;
    header->next = NULL;
    header->prev = NULL;

    list_head = header;
    return 1;
}

void *malloc(size_t size) 
{
    heap_header *header;
    if (size == 0) 
        return NULL;

    // 从第一个节点开始找
    header = list_head;
    // find a FREE MEM
    while(header != NULL) {
        // 当前块已经被使用
        if (header->type == HEAP_BLOCK_USED) {
            header = header->next;
            continue;
        }

        // 空间不够的异常情况处理
        if ((header->size > size + HEADER_SIZE) &&
            (header->size <= size + HEADER_SIZE *2))
        {
            header->type = HEAP_BLOCK_USED;
        }

        // 空间足够
        if (header->size > size + HEADER_SIZE *2) {
            // 创建一个新的header next，它的地址开始地点是当前的节点开始地址+header头大小+申请的空间大小
            heap_header *next = (heap_header*)ADDR_ADD(header, size+HEADER_SIZE);
            // 和当前的header绑定关系
            next->prev = header;
            // 与header之后的节点绑定关系
            next->next = header->next;
            // 标志是未使用
            next->type = HEAP_BLOCK_FREE;
            // 大小是剩余空间的大小（未使用）
            next->size = header->size - (size + HEADER_SIZE);
            // 当前的header节点就被征用了
            header->next = next;
            header->size = size + HEADER_SIZE;
            header->type = HEAP_BLOCK_USED;
            // 当前节点被征用，返回header+头部大小=真正的空余的地址开始位置
            return ADDR_ADD(header, HEADER_SIZE);
        }
        header = header->next;
    }
    return NULL;
}

void free(void *ptr) 
{
    heap_header *header = (heap_header *)ADDR_ADD(ptr, -HEADER_SIZE);
    if (header->type != HEAP_BLOCK_USED) {
        return;
    }

    /// 当前块的状态置为free
    header->type = HEAP_BLOCK_FREE;
    
    //merge
    // 和前面的free块合并
    if (header->prev != NULL && header->prev->type == HEAP_BLOCK_FREE) {
        header->prev->next = header->next;
        if (header->next != NULL)
            header->next->prev = header->prev;
        header->prev->size += header->size;

        header = header->prev;
    }
    
    // 和后面的free块合并
    if (header->next != NULL && header->next->type == HEAP_BLOCK_FREE) {
        header->size += header->next->size;
        header->next = header->next->next;
    }
}
