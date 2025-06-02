package ws_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/willmroliver/wsgo/protocol/ws"
)

func runTestServer(port int) (*ws.Server, context.CancelFunc) {
	s, err := ws.NewServer(port)
	if err != nil {
		return nil, nil
	}

	ctx, cancel := context.WithCancel(context.Background())

	go s.Run(ctx)

	return s, cancel
}

func TestHandshake(t *testing.T) {
	const PORT = 9001

	s, cancel := runTestServer(PORT)
	defer cancel()

	c, err := ws.NewClientConn(fmt.Sprintf(":%d", PORT), "")
	if err != nil {
		t.Error(err)
		return
	}

	time.Sleep(time.Millisecond)

	if exp, got := 1, len(s.Conns); exp != got {
		t.Errorf("exp %d conns, got %d\n", exp, got)
		return
	}

	if err := c.Handshake(); err != nil {
		t.Error(err)
		return
	}
}
