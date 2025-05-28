package core

import (
	"errors"
	"io"
)

var ErrBadHandshake = errors.New("bad handshake")

type Cxn interface {
	io.ReadWriteCloser
	Handshake() error
	Open() bool
	Buf() Buf
}
