package websocket

import "io"

type Message interface {
	Get(c *Cxn, r io.Reader)
}
