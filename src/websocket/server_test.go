package websocket_test

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/willmroliver/goagain/src/websocket"
)

func TestServerRun(t *testing.T) {
	s, err := websocket.NewServer(9999)
	if err != nil {
		t.Errorf("exp nil, got %q\n", err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	go s.Run(&ctx, &cancel)

	conn, err := net.Dial("tcp", ":9999")
	if err != nil {
		t.Errorf("exp nil, got %q\n", err)
		return
	}
	time.Sleep(time.Millisecond)
	if exp, got := 1, len(s.Cxns); exp != got {
		t.Errorf("exp %d, got %d\n", exp, got)
		return
	}

	var b strings.Builder

	b.WriteString("GET /chat HTTP/1.1")
	b.WriteString(websocket.CRLF)
	b.WriteString("Host: example.com")
	b.WriteString(websocket.CRLF)
	b.WriteString("Upgrade: websocket")
	b.WriteString(websocket.CRLF)
	b.WriteString("Connection: Upgrade")
	b.WriteString(websocket.CRLF)
	b.WriteString("Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==")
	b.WriteString(websocket.CRLF)
	b.WriteString("Origin: http://example.com")
	b.WriteString(websocket.CRLF)
	b.WriteString("Sec-WebSocket-Protocol: chat, superchat")
	b.WriteString(websocket.CRLF)
	b.WriteString("Sec-WebSocket-Version: 13")
	b.WriteString(websocket.CRLF)
	b.WriteString(websocket.CRLF)

	conn.Write([]byte(b.String()))

	buf := make([]byte, 0x100)
	n, err := conn.Read(buf)
	if exp := error(nil); err != exp || n == 0 {
		t.Errorf("exp (%q, >0), got (%q, %d)\n", exp, err, n)
		return
	}

	cancel()
}
