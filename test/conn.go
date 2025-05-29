package test

import (
	"io"
	"net"
	"time"

	"github.com/willmroliver/wsgo/core"
)

type Conn struct {
	buf *Buf
}

func NewConn(r io.Reader, w io.Writer) (c *Conn) {
	c = &Conn{NewBuf(r, w)}
	return
}

func (c *Conn) Read(p []byte) (int, error) {
	return c.buf.Read(p)
}

func (c *Conn) Write(p []byte) (int, error) {
	return c.buf.Write(p)
}

func (c *Conn) Close() (err error) {
	return
}

func (c *Conn) LocalAddr() (a net.Addr) {
	return
}

func (c *Conn) RemoteAddr() (a net.Addr) {
	return
}

func (c *Conn) SetDeadline(t time.Time) (err error) {
	return
}

func (c *Conn) SetReadDeadline(t time.Time) (err error) {
	return
}

func (c *Conn) SetWriteDeadline(t time.Time) (err error) {
	return
}

func (c *Conn) Handshake() (err error) {
	return
}

func (c *Conn) Open() bool {
	return true
}

func (c *Conn) Buf() core.Buf {
	return c.buf
}
