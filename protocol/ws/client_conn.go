package ws

import (
	"errors"
	"net"
	"strings"

	"github.com/willmroliver/wsgo/core"
	"github.com/willmroliver/wsgo/protocol/http1"
)

var ErrHandshakeFailed = errors.New("server rejected handshake")

type ClientConn struct {
	*net.TCPConn
	Host, Path string

	buf  core.Buf
	open bool
}

func (c *ClientConn) Buf() core.Buf {
	return c.buf
}

func (c *ClientConn) Open() bool {
	return c.open
}

func (c *ClientConn) Close() (err error) {
	if err = CloseFrame.Encode(c); err != nil {
		return err
	}

	err = c.TCPConn.Close()
	c.open = err != nil
	return
}

// Handshake sends an HTTP/1.x request to the server to
// upgrade to a WebSocket connection.
//
// On receiving a 101 switch response, server and client
// can proceed to send messages across the open channel.
func (c *ClientConn) Handshake() (err error) {
	if c.open {
		return
	}

	h := http1.NewMessage()
	h.ParseRequestLine("GET " + c.Path + " HTTP/1.1")
	h.Headers = map[string]string{
		"Host":                   c.Host,
		"Upgrade":                "websocket",
		"Connection":             "Upgrade",
		"Sec-WebSocket-Key":      "dGhlIHNhbXBsZSBub25jZQ==",
		"Sec-WebSocket-Protocol": "chat,superchat",
		"Sec-WebSocket-Version":  "13",
	}

	if err = h.Encode(c); err != nil {
		return
	}
	if err = h.Decode(c); err != nil {
		return
	}
	if h.StatusCode != "101" {
		err = ErrHandshakeFailed
		return
	}

	c.open = true
	return
}

func NewClientConn(address, path string) (c *ClientConn, err error) {
	addr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return
	}
	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		return
	}

	host := ""
	if i := strings.IndexByte(address, ':'); i > 0 {
		host = address[:i]
	}

	c = &ClientConn{
		conn,
		host,
		path,
		core.NewRingBuf(0x1000, conn),
		false,
	}

	return
}
