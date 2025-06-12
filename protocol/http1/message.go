package http1

import (
	"iter"
	"strings"

	"github.com/willmroliver/wsgo/core"
)

const (
	CRLF      string = "\r\n"
	DelimHTTP string = CRLF + CRLF
)

// Message offers just enough HTTP/1.x text-stream
// parsing support for the WebSocket handshake
type Message struct {
	Method, URI, Protocol  string
	StatusCode, StatusText string
	Headers                map[string]string
	HeaderParsed           bool
}

func NewMessage() *Message {
	return &Message{Headers: make(map[string]string)}
}

func (m *Message) Decode(c core.Conn) error {
	m.Headers = make(map[string]string)
	m.HeaderParsed = false

	i := -1

	for i == -1 {
		if c.Buf().Full() {
			return core.ErrBadHeader
		}

		c.Buf().Fill()

		i = c.Buf().IndexOf([]byte(DelimHTTP))
	}

	bytes := make([]byte, i)
	_, err := c.Buf().Read(bytes)
	if err != nil {
		return err
	}

	it := strings.SplitSeq(string(bytes), CRLF)
	next, _ := iter.Pull(it)
	line, ok := next()

	if !ok || len(line) < 4 {
		return core.ErrBadHeader
	}

	if line[:4] == "HTTP" {
		m.Method, m.URI, m.Protocol = "", "", ""

		if !m.ParseStatusLine(line) {
			return core.ErrBadHeader
		}
	} else if !m.ParseRequestLine(line) {
		m.StatusCode, m.StatusText = "", ""

		return core.ErrBadHeader
	}

	for {
		line, ok = next()
		if !ok {
			break
		}
		if line == "" {
			continue
		}

		i := strings.IndexByte(line, ':')
		if i < 1 {
			return core.ErrBadHeader
		}

		m.Headers[line[:i]] = line[i+2:]
	}

	m.HeaderParsed = true
	return nil
}

func (m *Message) Encode(c core.Conn) (err error) {
	var b strings.Builder

	if m.Method != "" {
		b.WriteString(
			m.Method + " " +
				m.URI + " " +
				m.Protocol + CRLF,
		)
	} else {
		b.WriteString(
			m.Protocol + " " +
				m.StatusCode + " " +
				m.StatusText + CRLF,
		)
	}

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
	m.StatusCode, m.StatusText = "", ""
	return true
}

func (m *Message) ParseStatusLine(s string) bool {
	n := len(s)
	var i, j int
	if i = strings.IndexByte(s, ' '); i < 1 || i == n-1 {
		return false
	}
	if j = strings.IndexByte(s[i+1:], ' ') + i + 1; j <= i || j == n-1 {
		return false
	}

	m.Protocol, m.StatusCode, m.StatusText = s[:i], s[i+1:j], s[j+1:]
	m.Method, m.URI = "", ""
	return true
}
