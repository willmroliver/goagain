package ws_test

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/willmroliver/wsgo/protocol/ws"
	"github.com/willmroliver/wsgo/test"
)

func TestEncode(t *testing.T) {
	r := strings.NewReader("")
	w := new(bytes.Buffer)
	conn := test.NewConn(r, w)

	t.Run("Simple frame", func(t *testing.T) {
		w.Reset()

		f := &ws.Message{
			Payload: []byte{1, 1, 2, 2, 3, 3, 4, 4},
			Final:   true,
			Opcode:  ws.FrameOpcodeBinary,
		}

		exp := []byte{(1 << 7) | f.Opcode, 8}
		exp = append(exp, f.Payload...)

		if err := f.Encode(conn); err != nil {
			t.Error(err)
			return
		}

		if got := w.Bytes(); !slices.Equal(exp, got) {
			t.Errorf("exp %+v, got %+v", exp, got)
		}
	})

	t.Run("Extended payload", func(t *testing.T) {
		w.Reset()

		f := &ws.Message{
			Payload: bytes.Repeat([]byte{1, 2, 3, 4, 1, 2, 3, 4}, 256),
			Final:   false,
			Opcode:  ws.FrameOpcodeText,
		}

		epl := make([]byte, 2)
		binary.Encode(epl, binary.BigEndian, uint16(len(f.Payload)))

		exp := []byte{f.Opcode, 126}
		exp = append(exp, epl...)
		exp = append(exp, f.Payload...)

		if err := f.Encode(conn); err != nil {
			t.Error(err)
			return
		}

		if got := w.Bytes(); !slices.Equal(exp, got) {
			t.Errorf(
				"2 byte EPL - exp %+v, got %+v",
				exp[:min(len(exp), 100)],
				got[:min(len(got), 100)],
			)
			return
		}

		w.Reset()

		f.Payload = bytes.Repeat([]byte{1}, math.MaxUint16*2)
		epl = make([]byte, 8)
		binary.Encode(epl, binary.BigEndian, uint64(len(f.Payload)))

		exp = []byte{f.Opcode, 127}
		exp = append(exp, epl...)
		exp = append(exp, f.Payload...)

		if err := f.Encode(conn); err != nil {
			t.Error(err)
			return
		}

		if got := w.Bytes(); !slices.Equal(exp, got) {
			t.Errorf(
				"8-byte EPL - exp %+v, got %+v",
				exp[:min(len(exp), 100)],
				got[:min(len(got), 100)],
			)
			return
		}
	})

	t.Run("Masked payload", func(t *testing.T) {
		w.Reset()

		payload := []byte{1, 1, 0, 0, 2, 2, 4, 4}

		f := &ws.Message{
			Payload:    payload,
			Opcode:     ws.FrameOpcodeCont,
			MaskingKey: [4]byte{1, 0, 2, 0},
		}

		f.ApplyMask()

		exp := []byte{
			f.Opcode,
			(1 << 7) | byte(len(payload)),
		}
		exp = append(exp, f.MaskingKey[:]...)
		exp = append(exp, []byte{
			1 ^ 1, 1 ^ 0, 0 ^ 2, 0 ^ 0,
			2 ^ 1, 2 ^ 0, 4 ^ 2, 4 ^ 0,
		}...)

		if err := f.Encode(conn); err != nil {
			t.Error(err)
			return
		}

		if got := w.Bytes(); !slices.Equal(exp, got) {
			t.Errorf("exp %+v, got %+v", exp, got)
			return
		}
	})
}

func BenchmarkEncode(t *testing.B) {
	r := strings.NewReader("")
	w := new(bytes.Buffer)
	conn := test.NewConn(r, w)

	f := &ws.Message{
		Payload: slices.Repeat([]byte{1, 2, 3, 4}, 0x100),
		Opcode:  ws.FrameOpcodeCont,
	}

	f.NewMaskingKey()
	f.ApplyMask()

	for t.Loop() {
		f.Encode(conn)
		w.Reset()
	}
}

func TestDecode(t *testing.T) {
	r := new(bytes.Reader)
	w := new(bytes.Buffer)
	conn := test.NewConn(r, w)

	t.Run("Simple frame", func(t *testing.T) {
		f, g := ws.Message{
			Opcode:  ws.FrameOpcodeText,
			Payload: []byte("Arsenal"),
		}, ws.Message{}

		data, _ := f.EncodeBytes()
		conn.Buf().Reset(bytes.NewReader(data))

		if err := g.Decode(conn); err != nil && err != io.EOF {
			t.Error(err)
			return
		}

		if f.Opcode != g.Opcode || !slices.Equal(f.Payload, g.Payload) {
			t.Errorf("exp %+v, got %+v\n", f, g)
			return
		}
	})

	t.Run("Extended payload", func(t *testing.T) {
		f, g := ws.Message{
			Opcode:  ws.FrameOpcodeBinary,
			Payload: slices.Repeat([]byte{1, 2, 3, 4}, 0x100),
		}, ws.Message{}

		data, _ := f.EncodeBytes()
		conn.Buf().Reset(bytes.NewReader(data))

		if err := g.Decode(conn); err != nil && err != io.EOF {
			t.Error(err)
			return
		}

		if f.Opcode != g.Opcode || !slices.Equal(f.Payload, g.Payload) {
			f.Payload = f.Payload[:10]
			g.Payload = g.Payload[:min(len(g.Payload), 10)]
			t.Errorf("exp %v, got %v\n", f, g)
			return
		}
	})

	t.Run("Masked payload", func(t *testing.T) {
		f, g := ws.Message{
			Opcode:  ws.FrameOpcodeBinary,
			Payload: slices.Repeat([]byte{1, 2, 3, 4}, 0x100),
		}, ws.Message{}
		f.ApplyMask()

		data, _ := f.EncodeBytes()
		conn.Buf().Reset(bytes.NewReader(data))

		if err := g.Decode(conn); err != nil && err != io.EOF {
			t.Error(err)
			return
		}

		if f.Opcode != g.Opcode || !slices.Equal(f.Payload, g.Payload) {
			f.Payload = f.Payload[:10]
			g.Payload = g.Payload[:min(len(g.Payload), 10)]
			t.Errorf("exp %v, got %v\n", f, g)
			return
		}

		if f.Masked != g.Masked || !slices.Equal(f.MaskingKey[:], g.MaskingKey[:]) {
			t.Errorf(
				"exp (%t, %v), got (%t, %v)\n",
				f.Masked, f.MaskingKey, g.Masked, g.MaskingKey,
			)
			return
		}
	})
}

func BenchmarkDecode(t *testing.B) {
	f := ws.Message{
		Opcode:  ws.FrameOpcodeBinary,
		Payload: slices.Repeat([]byte{1, 2, 3, 4}, 0x100),
	}
	f.NewMaskingKey()
	f.ApplyMask()
	data, _ := f.EncodeBytes()

	r := bytes.NewReader(data)
	w := new(bytes.Buffer)
	conn := test.NewConn(r, w)

	for t.Loop() {
		conn.Buf().Reset(bytes.NewReader(data))
		f.Decode(conn)
	}
}

func TestApplyMask(t *testing.T) {
	payload := []byte{1, 1, 0, 0, 2, 2, 4, 4}

	var f ws.Message

	f.Payload = payload
	f.MaskingKey = [4]byte{1, 0, 2, 0}

	f.ApplyMask()
	exp := []byte{
		1 ^ 1, 1 ^ 0, 0 ^ 2, 0 ^ 0,
		2 ^ 1, 2 ^ 0, 4 ^ 2, 4 ^ 0,
	}

	if !slices.Equal(exp, f.Payload) {
		t.Errorf("exp %+v, got %+v\n", exp, f.Payload)
	}

	f.ApplyMask()

	if !slices.Equal(payload, f.Payload) {
		t.Errorf("exp %+v, got %+v", payload, f.Payload)
	}
}

func BenchmarkApplyMask(t *testing.B) {
	var f ws.Message
	f.Payload = slices.Repeat([]byte{1, 2, 3, 4, 5, 6, 7, 8}, 0x100)
	f.MaskingKey = [4]byte{1, 2, 3, 4}

	for t.Loop() {
		f.ApplyMask()
	}
}

func TestUnsafeMask(t *testing.T) {
	payload := []byte{1, 1, 0, 0, 2, 2, 4, 4}

	var f ws.Message

	f.Payload = payload
	f.MaskingKey = [4]byte{1, 0, 2, 0}

	f.UnsafeMask()
	exp := []byte{
		1 ^ 1, 1 ^ 0, 0 ^ 2, 0 ^ 0,
		2 ^ 1, 2 ^ 0, 4 ^ 2, 4 ^ 0,
	}

	if !slices.Equal(exp, f.Payload) {
		t.Errorf("exp %+v, got %+v\n", exp, f.Payload)
	}

	f.UnsafeMask()

	if !slices.Equal(payload, f.Payload) {
		t.Errorf("exp %+v, got %+v", payload, f.Payload)
	}
}

func BenchmarkUnsafeMask(t *testing.B) {
	var f ws.Message
	f.Payload = slices.Repeat([]byte{1, 2, 3, 4, 5, 6, 7, 8}, 0x100)
	f.MaskingKey = [4]byte{1, 2, 3, 4}

	for t.Loop() {
		f.UnsafeMask()
	}
}

func TestNewControlFrameFuncs(t *testing.T) {
	frames := [][]byte{
		ws.NewCloseFrame(ws.StatusCodeNormalClosure, "Normal closure"),
		ws.NewPingFrame(),
		ws.NewPongFrame(),
	}

	for _, f := range frames {
		if exp, got := 125, len(f); exp < got {
			t.Errorf("exp < %d, got %d", exp, got)
			return
		}
	}
}
