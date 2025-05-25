package websocket

import (
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"net"

	"github.com/willmroliver/goagain/src/container"
)

const (
	ProtocolGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
)

var ErrBadHandshake = errors.New("bad handshake")

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
	h := NewMessageHTTP()
	if err = h.Receive(c); err != nil {
		c.Close()
		return
	}

	valid := h.Method == "GET" &&
		h.Protocol == "HTTP/1.1" &&
		h.URI != c.server.Conf.Path &&
		h.Headers["Upgrade"] == "websocket" &&
		h.Headers["Connection"] == "Upgrade" &&
		h.Headers["Sec-Websocket-Version"] == "13" &&
		len(h.Headers["Sec-Websocket-Key"]) == 24

	if !valid {
		err = ErrBadHandshake
		return
	}

	key := h.Headers["Sec-Websocket-Key"]
	checksum := sha1.Sum([]byte(key + ProtocolGUID))

	h.StatusLine = "101 Switching Protocols"
	h.Headers = map[string]string{
		"Upgrade":              "websocket",
		"Connection":           "Upgrade",
		"Sec-Websocket-Accept": base64.StdEncoding.EncodeToString(checksum[:]),
	}

	err = h.Send(c)
	return
}
