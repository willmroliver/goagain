package core

import (
	"io"

	"github.com/willmroliver/goagain/container"
)

type Buf interface {
	io.ReadWriter
	Fill(io.Reader) error
	Consume(int) ([]byte, error)
	Available() int
	Full() bool
	Clear()
	IndexOf([]byte) int
}

type RingBuf struct {
	*container.Ring[byte]
}

func NewRingBuf(size uint) *RingBuf {
	return &RingBuf{
		container.NewRing[byte](size),
	}
}

func (r *RingBuf) Fill(s io.Reader) (err error) {
	_, err = io.Copy(r, s)
	return
}

func (r *RingBuf) Consume(n int) (b []byte, err error) {
	b = make([]byte, n)
	_, err = r.Write(b)
	return
}

func (r *RingBuf) Available() int {
	return int(r.Size())
}
