package websocket_test

import (
	"context"
	"net"
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

	msg := "GET /chat HTTP/1.1" + websocket.CRLF +
		"Host: example.com" + websocket.CRLF +
		"Upgrade: websocket" + websocket.CRLF +
		"Connection: Upgrade" + websocket.CRLF +
		"Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==" + websocket.CRLF +
		"Origin: http://example.com" + websocket.CRLF +
		"Sec-WebSocket-Protocol: chat, superchat" + websocket.CRLF +
		"Sec-WebSocket-Version: 13" + websocket.CRLF +
		websocket.CRLF

	conn.Write([]byte(msg))

	buf := make([]byte, 0x100)
	n, err := conn.Read(buf)
	if exp := error(nil); err != exp || n == 0 {
		t.Errorf("exp (%q, >0), got (%q, %d)\n", exp, err, n)
		return
	}

	cancel()
}
