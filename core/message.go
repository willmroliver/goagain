package core

import (
	"errors"
)

var ErrBadHeader = errors.New("bad header")

// Message acts as a codec for some kind of transmittable unit,
// such as a text-based message, an encoded HTTP/1.1 chunk or
// a WebSocket binary-encoded frame
type Message interface {
	Decode(Conn) error
	Encode(Conn) error
}
