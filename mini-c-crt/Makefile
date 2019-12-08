.PHONY: test clean

all: clean test

test:
	gcc -c -ggdb -fno-builtin -nostdlib entry.c stdio.c string.c stdlib.c
	gcc -c -ggdb -fno-builtin -nostdlib ./test.c 
	ld -static -e crt_entry ./entry.o ./string.o ./stdio.o ./stdlib.o ./test.o -o ./test

clean:
	rm -rf ./*.o
	rm -rf ./test