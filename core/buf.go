package core

import (
	"io"

	"github.com/willmroliver/wsgo/container"
)

type Buf interface {
	io.ReadWriter
	Fill() error
	Full() bool
	Available() int
	Reset(io.Reader)
	IndexOf([]byte) int
}

type RingBuf struct {
	*container.Ring[byte]
	r io.Reader
}

func NewRingBuf(size uint, r io.Reader) *RingBuf {
	return &RingBuf{
		container.NewRing[byte](size),
		r,
	}
}

func (r *RingBuf) Fill() (err error) {
	_, err = io.Copy(r, r.r)
	return
}

func (r *RingBuf) Available() int {
	return int(r.Size())
}

func (r *RingBuf) Reset(s io.Reader) {
	r.Clear()
	r.r = s
}
