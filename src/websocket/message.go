package websocket

import (
	"errors"
	"io"
	"strings"
)

const (
	CRLF      string = "\r\n"
	DelimHTTP string = CRLF + CRLF
)

var ErrBadHeader = errors.New("bad header")

type Message interface {
	Receive(*Cxn) error
	Send(*Cxn) error
}

// MessageHTTP offers just enough to parse a client handshake
type MessageHTTP struct {
	Method, URI, Protocol, StatusLine string
	Headers                           map[string]string
	HeaderParsed                      bool
}

func NewMessageHTTP() *MessageHTTP {
	return &MessageHTTP{Headers: make(map[string]string)}
}

func (m *MessageHTTP) Receive(c *Cxn) error {
	m.Method, m.URI, m.Protocol = "", "", ""
	m.Headers = make(map[string]string)
	m.HeaderParsed = false

	for !c.Buf.HasSuffix([]byte(DelimHTTP)) {
		if c.Buf.Full() {
			return ErrBadHeader
		}

		io.Copy(c.Buf, c)
	}

	b := &strings.Builder{}
	io.Copy(b, c.Buf)

	for line := range strings.SplitSeq(b.String(), CRLF) {
		if m.Method == "" {
			if !m.parseRequestLine(line) {
				return ErrBadHeader
			}
			continue
		}

		if line == "" {
			continue
		}

		i := strings.IndexByte(line, ':')
		if i < 1 {
			return ErrBadHeader
		}

		m.Headers[line[:i]] = line[i+2:]
	}

	return nil
}

func (m *MessageHTTP) Send(c *Cxn) error {
	data := m.Protocol + " " + m.StatusLine + CRLF

	for k, v := range m.Headers {
		data += k + ": " + v + CRLF
	}

	data += CRLF

	_, err := c.Write([]byte(data))

	return err
}

func (m *MessageHTTP) parseRequestLine(s string) bool {
	parts := strings.Split(s, " ")
	if len(parts) != 3 {
		return false
	}

	m.Method, m.URI, m.Protocol = parts[0], parts[1], parts[2]
	return true
}
