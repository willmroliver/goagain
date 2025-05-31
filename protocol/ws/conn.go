package ws

import (
	"crypto/sha1"
	"encoding/base64"
	"net"

	"github.com/willmroliver/wsgo/core"
	"github.com/willmroliver/wsgo/protocol/http1"
)

const (
	ProtocolGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
)

type Conn struct {
	*net.TCPConn
	ConnID uint
	Server core.Server

	buf  core.Buf
	open bool
}

func (c *Conn) Close() error {
	c.Server.Close(c)
	CloseFrame.Encode(c)
	return c.TCPConn.Close()
}

func (c *Conn) Buf() core.Buf {
	return c.buf
}

func (c *Conn) Open() bool {
	return c.open
}

func (c *Conn) Handshake() (err error) {
	h := http1.NewMessage()
	if err = h.Decode(c); err != nil {
		c.Close()
		return
	}

	uri := ""
	s, ok := c.Server.(*Server)
	if s != nil && ok {
		uri = s.Conf.Path
	}

	valid := h.Method == "GET" &&
		h.Protocol == "HTTP/1.1" &&
		(uri == "" || h.URI == uri) &&
		h.Headers["Upgrade"] == "websocket" &&
		h.Headers["Connection"] == "Upgrade" &&
		h.Headers["Sec-WebSocket-Version"] == "13" &&
		len(h.Headers["Sec-WebSocket-Key"]) == 24

	if !valid {
		err = core.ErrBadHandshake
		return
	}

	key := h.Headers["Sec-Websocket-Key"]
	checksum := sha1.Sum([]byte(key + ProtocolGUID))

	h.ParseStatusLine("HTTP/1.1 101 Switching Protocols")
	h.Headers = map[string]string{
		"Upgrade":    "websocket",
		"Connection": "Upgrade",
		"Sec-Websocket-Accept": base64.StdEncoding.
			EncodeToString(checksum[:]),
	}

	err = h.Encode(c)
	c.open = err == nil
	return
}
