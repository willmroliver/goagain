package http1

import (
	"io"
	"iter"
	"strings"

	"github.com/willmroliver/goagain/core"
)

const (
	CRLF      string = "\r\n"
	DelimHTTP string = CRLF + CRLF
)

// Message offers just enough HTTP/1.1 text-stream
// parsing support for the WebSocket handshake
type Message struct {
	Method, URI, Protocol, StatusLine string
	Headers                           map[string]string
	HeaderParsed                      bool
}

func NewMessage() *Message {
	return &Message{Headers: make(map[string]string)}
}

func (m *Message) Decode(c core.Conn) error {
	m.Method, m.URI, m.Protocol = "", "", ""
	m.Headers = make(map[string]string)
	m.HeaderParsed = false

	for i := -1; i == -1; i = c.Buf().IndexOf([]byte(DelimHTTP)) {
		if c.Buf().Full() {
			return core.ErrBadHeader
		}

		c.Buf().Fill(c)
	}

	b := &strings.Builder{}
	io.Copy(b, c.Buf())

	it := strings.SplitSeq(b.String(), CRLF)
	next, _ := iter.Pull(it)
	line, ok := next()

	if !(ok && m.ParseRequestLine(line)) {
		return core.ErrBadHeader
	}

	for line := range it {
		if line == "" {
			continue
		}

		i := strings.IndexByte(line, ':')
		if i < 1 {
			return core.ErrBadHeader
		}

		m.Headers[line[:i]] = line[i+2:]
	}

	return nil
}

func (m *Message) Encode(c core.Conn) (err error) {
	var b strings.Builder

	b.WriteString(m.Protocol + " " + m.StatusLine + CRLF)

	for k, v := range m.Headers {
		b.WriteString(k + ": " + v + CRLF)
	}

	b.WriteString(CRLF)

	_, err = c.Write([]byte(b.String()))
	return
}

func (m *Message) ParseRequestLine(s string) bool {
	parts := strings.Split(s, " ")
	if len(parts) != 3 {
		return false
	}

	m.Method, m.URI, m.Protocol = parts[0], parts[1], parts[2]
	return true
}
