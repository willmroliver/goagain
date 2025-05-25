package websocket

import "io"

const (
	DelimHTTP string = "\r\n\r\n"
)

type Message interface {
	Next(c *Cxn) error
	Ready() bool
	Get() []byte
}

type MessageHTTP struct {
	Headers      map[string]string
	HeaderParsed bool
	HasBody      bool
}

func NewMessageHTTP() *MessageHTTP {
	return &MessageHTTP{Headers: make(map[string]string)}
}

func (m *MessageHTTP) Next(c *Cxn) (err error) {
	m.Headers = make(map[string]string)
	m.HeaderParsed, m.HasBody = false, false

	for !m.HeaderParsed {
		io.Copy(c.buf, c)
		if c.buf.HasSuffix([]byte(DelimHTTP)) {
			m.HeaderParsed = true
		}
	}

	return
}
