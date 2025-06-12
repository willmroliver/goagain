package ws

import (
	"crypto/sha1"
	"encoding/base64"
	"errors"
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
	if c.Server != nil && c.open {
		c.Server.Close(c)
		CloseFrame.Encode(c)
	}

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

	if h.Method != "GET" {
		err = errors.New("invalid method, expecting GET")
		return
	}

	if len(h.Protocol) != 8 ||
		h.Protocol[:7] != "HTTP/1." ||
		h.Protocol[7] < '1' ||
		h.Protocol[7] > '3' {
		err = errors.New("invalid protocol, expecting HTTP/1.x")
		return
	}

	if uri != "" && h.URI != uri {
		err = errors.New("invalid URI in header")
		return
	}

	if h.Headers["Upgrade"] != "websocket" {
		err = errors.New(
			"invalid header 'Upgrade', expecting 'websocket'",
		)
		return
	}

	if h.Headers["Connection"] != "Upgrade" {
		err = errors.New(
			"invalid header 'Connection', expecting 'Upgrade'",
		)
		return
	}

	if h.Headers["Sec-WebSocket-Version"] != "13" {
		err = errors.New(
			"invalid header 'Sec-WebSocket-Version', expecting '13'",
		)
		return
	}

	key := h.Headers["Sec-Websocket-Key"]
	checksum := sha1.Sum([]byte(key + ProtocolGUID))

	h.ParseStatusLine("HTTP/1.1 101 Switching Protocols")

	h.Headers = map[string]string{
		"Upgrade":              "websocket",
		"Connection":           "Upgrade",
		"Sec-Websocket-Accept": base64.StdEncoding.EncodeToString(checksum[:]),
	}

	err = h.Encode(c)
	c.open = err == nil
	return
}
