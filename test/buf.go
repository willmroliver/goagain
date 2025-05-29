package test

import (
	"bufio"
	"bytes"
	"io"
)

type Buf struct {
	*bufio.Reader
	w io.Writer
}

func NewBuf(r io.Reader, w io.Writer) *Buf {
	return &Buf{bufio.NewReader(r), w}
}

func (r *Buf) Write(p []byte) (int, error) {
	return r.w.Write(p)
}

func (r *Buf) Fill() (err error) {
	_, err = r.Reader.Peek(1)
	return
}

func (r *Buf) Full() bool {
	return false
}

func (r *Buf) IndexOf(b []byte) int {
	s, _ := r.Peek(r.Buffered())
	return bytes.Index(s, b)
}

func (r *Buf) Available() int {
	return r.Reader.Buffered()
}
