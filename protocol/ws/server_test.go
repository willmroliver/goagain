package ws_test

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/willmroliver/wsgo/protocol/ws"
)

func TestServerRun(t *testing.T) {
	const PORT = 9000

	s, err := ws.NewServer(PORT)
	if err != nil {
		t.Errorf("exp nil, got %q\n", err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	go s.Run(ctx)

	defer cancel()

	conn, err := net.Dial("tcp", fmt.Sprintf(":%d", PORT))
	if err != nil {
		t.Errorf("Dial: exp nil, got %q\n", err)
		return
	}
	time.Sleep(time.Millisecond)
	if exp, got := 1, len(s.Conns); exp != got {
		t.Errorf("len(Cxns): exp %d, got %d\n", exp, got)
		return
	}

	msg := "GET /chat HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\n" +
		"Origin: http://example.com\r\n" +
		"Sec-WebSocket-Protocol: chat, superchat\r\n" +
		"Sec-WebSocket-Version: 13\r\n" +
		"\r\n"

	conn.Write([]byte(msg))

	buf := make([]byte, 0x100)
	n, err := conn.Read(buf)
	if exp := error(nil); err != exp || n == 0 {
		t.Errorf("Read err: exp %v, got %v\n", exp, err)
		return
	}

	exp := "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-Websocket-Accept: Kfh9QIsMVZcl6xEPYxPHzW8SZ8w=\r\n" +
		"\r\n"

	if m := len(exp); n != m {
		t.Errorf("\n%s\n\n%s\n", exp, buf[:n])
		t.Errorf("Read n: exp %d, got %d\n", m, n)
		return
	}

	// @todo - proper value comparison
	//
	// go maps are unpredictably ordered so, should build some
	// http utilities for testing req/res equality
}
