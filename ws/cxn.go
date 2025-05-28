package ws

import (
	"context"
	"crypto/sha1"
	"encoding/base64"
	"net"

	"github.com/willmroliver/goagain/container"
)

const (
	ProtocolGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
)

type CxnWS struct {
	*net.TCPConn
	CxnID  uint
	Server *Server
	Buf    *container.Ring[byte]
}

func (c *Cxn) Close() error {
	delete(c.Server.Cxns, c.CxnID)
	return c.TCPConn.Close()
}

func (c *Cxn) Talk(ctx context.Context) {
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
		(c.Server.Conf.Path == "" || h.URI != c.Server.Conf.Path) &&
		h.Headers["Upgrade"] == "websocket" &&
		h.Headers["Connection"] == "Upgrade" &&
		h.Headers["Sec-WebSocket-Version"] == "13" &&
		len(h.Headers["Sec-WebSocket-Key"]) == 24

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
