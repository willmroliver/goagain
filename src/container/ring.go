package container

import (
	"errors"
	"fmt"
	"io"
)

var (
	ErrRingFull  = errors.New("Ring buffer full")
	ErrRingEmpty = errors.New("Ring buffer empty")
)

type Ring[T comparable] struct {
	buf          []T
	size         uint
	imask, lmask uint
	start, end   uint
}

func NewRing[T comparable](size uint) (r *Ring[T]) {
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

func (r *Ring[T]) Write(p []byte) (n int, err error) {
	w := func(dest []byte, arg any) (n int, err error) {
		src := arg.(*[]byte)

		var m int

		if m, n = len(dest), len(*src); m < n {
			err = io.EOF
			n = m
		}

		if n == 0 {
			err = ErrRingFull
			return
		}

		copy(dest[:n], (*src)[:n])
		*src = (*src)[n:]

		return
	}

	q := p
	n, err = any(r).(*Ring[byte]).WriteFunc(w, &q)
	return
}

func (r *Ring[T]) WriteTo(w io.Writer) (n int64, err error) {
	if r.Empty() {
		err = ErrRingEmpty
		return
	}

	br := any(r).(*Ring[byte])

	from, to, wrap := r.start&r.imask, r.end&r.imask, false
	if to <= from {
		to = r.size
		wrap = true
	}

	var m int
	m, err = w.Write(br.buf[from:to])
	r.start = (r.start + uint(m)) & r.lmask
	if err != nil || !wrap {
		return
	}

	hold, to := m, r.end&r.imask
	m, err = w.Write(br.buf[:to])
	r.start = (r.start + uint(m)) & r.lmask

	return int64(m + hold), err
}

func (r *Ring[T]) Read(p []byte) (n int, err error) {
	if r.Empty() {
		err = io.EOF
		return
	}

	br := any(r).(*Ring[byte])

	for ; n < len(p); n++ {
		if !br.Pop(&p[n]) {
			return
		}
	}

	return
}

func (r *Ring[T]) ReadFrom(src io.Reader) (n int64, err error) {
	if r.Full() {
		err = ErrRingFull
		return
	}

	br := any(r).(*Ring[byte])

	from, to, wrap := r.end&r.imask, r.start&r.imask, false
	if to <= from {
		to = r.size
		wrap = true
	}

	var m int
	m, err = src.Read(br.buf[from:to])
	r.end = (r.end + uint(m)) & r.lmask
	if err != nil || !wrap {
		return
	}

	hold, to := m, r.start&r.imask
	m, err = src.Read(br.buf[:to])
	r.end = (r.end + uint(m)) & r.lmask

	return int64(m + hold), err
}

func (r *Ring[T]) HasSuffix(s []T) bool {
	var n int
	if n = len(s); n == 0 {
		return true
	}
	if r.Empty() || r.Size() != uint(n) {
		return false
	}

	end := r.end & r.imask
	start := (end - uint(n)) & r.imask

	for i, t := range s {
		if t != r.buf[(start+uint(i))&r.imask] {
			return false
		}
	}

	return true
}
