package container

import (
	"errors"
	"fmt"
	"io"
)

type Ring[T any] struct {
	buf          []T
	size         uint
	imask, lmask uint
	start, end   uint
}

func NewRing[T any](size uint) (r *Ring[T]) {
	if size&(size-1) != 0 || size == 0 {
		size = 0x1000
	}

	r = &Ring[T]{
		buf:   make([]T, size),
		size:  size,
		imask: size - 1,
		lmask: 2*size - 1,
	}

	return
}

func (r *Ring[T]) Empty() bool {
	return r.start == r.end
}

func (r *Ring[T]) Full() bool {
	return (r.end-r.start)&(r.lmask) == r.size
}

func (r *Ring[T]) Cap() uint {
	return r.size
}

func (r *Ring[T]) Size() uint {
	return (r.end - r.start) & (r.lmask)
}

func (r *Ring[T]) Push(val T) bool {
	if r.Full() {
		return false
	}

	r.buf[r.end&r.imask] = val
	r.end = (r.end + 1) & r.lmask

	return true
}

func (r *Ring[T]) Pop(val *T) bool {
	if r.Empty() {
		return false
	}

	*val = r.buf[r.start&r.imask]
	r.start = (r.start + 1) & r.lmask

	return true
}

func (r *Ring[T]) Debug() string {
	return fmt.Sprintf("%+v\n", r)
}

// WriteFunc partially exposes the underlying memory to a callback
// in contiguous slices: when the write operation must wrap around
// the underlying slice bounds, the callback will be applied twice.
//
// Unlike the io.ReadWriter interfaces, w func() must return a
// non-nil error to indicate the source data has NOT been exhausted
func (r *Ring[T]) WriteFunc(
	w func([]T, any) (int, error),
	arg any,
) (n int, err error) {
	from, to, wrap := r.end&r.imask, r.start&r.imask, false
	if to <= from {
		to = r.size
		wrap = true
	}

	n, err = w(r.buf[from:to], arg)
	r.end = (r.end + uint(n)) & r.lmask
	if err == nil || !wrap {
		return
	}

	hold, to := n, r.start&r.imask
	n, err = w(r.buf[:to], arg)
	r.end = (r.end + uint(n)) & r.lmask
	return n + hold, err
}

func (r *Ring[byte]) Write(p []byte) (n int, err error) {
	w := func(dest []byte, arg any) (n int, err error) {
		src := arg.(*[]byte)

		var m int

		if m, n = len(dest), len(*src); m < n {
			err = io.EOF
			n = m
		}

		if n == 0 {
			err = errors.New("Buffer full")
			return
		}

		copy(dest[:n], (*src)[:n])
		*src = (*src)[n:]

		return
	}

	q := p
	n, err = r.WriteFunc(w, &q)
	return
}

func (r *Ring[byte]) Read(p []byte) (n int, err error) {
	if r.Empty() {
		err = io.EOF
		return
	}

	for ; n < len(p); n++ {
		if !r.Pop(&p[n]) {
			return
		}
	}

	return
}
