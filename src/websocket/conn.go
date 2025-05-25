package websocket

import (
	"net"

	"github.com/willmroliver/goagain/src/container"
)

type Cxn struct {
	*net.TCPConn
	server *Server
	buf    *container.Ring[byte]
}

func NewCxn(s *Server, c *net.TCPConn) *Cxn {
	return &Cxn{
		c,
		s,
		container.NewRing[byte](s.Conf.CxnBufSize),
	}
}

func (c *Cxn) Talk() {
	if err := c.Handshake(); err != nil {
		return
	}
}

func (c *Cxn) Handshake() (err error) {
	return
}
