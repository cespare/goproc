package main

import "fmt"

type CircBuf struct {
	buf  []int64
	size int // equal to len(buf)
	full bool
	// Cur points to the *next* unused position (so that when cur == 0 and full == false, there are 0 elements).
	cur int
}

func NewCircBuf(size int) *CircBuf {
	return &CircBuf{
		buf:  make([]int64, size),
		size: size,
	}
}

func (b *CircBuf) Full() bool { return b.full }

func (b *CircBuf) Len() int {
	if b.full {
		return b.size
	}
	return b.cur
}

func (b *CircBuf) Append(v int64) {
	b.buf[b.cur] = v
	b.cur = (b.cur + 1) % b.size
	if b.cur == 0 {
		b.full = true
	}
}
func (b *CircBuf) Delta() int64 {
	first := 0
	if b.full {
		first = b.cur % b.size
	}
	last := b.cur - 1
	if last < 0 {
		last = b.size - 1
	}
	return b.buf[last] - b.buf[first]
}

func (b *CircBuf) String() string {
	return fmt.Sprintf("%#v", b)
}
