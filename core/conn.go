package core

import (
	"errors"
	"net"
)

var ErrBadHandshake = errors.New("bad handshake")

type Conn interface {
	net.Conn
	Handshake() error
	Open() bool
	Buf() Buf
}
