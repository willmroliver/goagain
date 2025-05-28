package core

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

type Frame interface {
	Decode(Cxn) error
	Encode(Cxn) error
}

// HTTPMessage offers just enough to parse a client handshake
type HTTPFrame struct {
	Method, URI, Protocol, StatusLine string
	Headers                           map[string]string
	HeaderParsed                      bool
}

func NewHTTPFrame() *HTTPFrame {
	return &HTTPFrame{Headers: make(map[string]string)}
}

func (m *HTTPFrame) Decode(c Cxn) error {
	m.Method, m.URI, m.Protocol = "", "", ""
	m.Headers = make(map[string]string)
	m.HeaderParsed = false

	for i := -1; i == -1; i = c.Buf().IndexOf([]byte(DelimHTTP)) {
		if c.Buf().Full() {
			return ErrBadHeader
		}

		c.Buf().Fill(c)
	}

	b := &strings.Builder{}
	io.Copy(b, c.Buf())

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

func (m *HTTPFrame) Send(c *Cxn) (err error) {
	var b strings.Builder

	b.WriteString(m.Protocol + " " + m.StatusLine + CRLF)

	for k, v := range m.Headers {
		b.WriteString(k + ": " + v + CRLF)
	}

	b.WriteString(CRLF)

	_, err = c.Write([]byte(b.String()))
	return
}

func (m *HTTPFrame) parseRequestLine(s string) bool {
	parts := strings.Split(s, " ")
	if len(parts) != 3 {
		return false
	}

	m.Method, m.URI, m.Protocol = parts[0], parts[1], parts[2]
	return true
}
