package ws_test

import (
	"bytes"
	"encoding/binary"
	"math"
	"slices"
	"testing"

	"github.com/willmroliver/goagain/protocol/ws"
)

func TestEncode(t *testing.T) {
	t.Run("Simple frame", func(t *testing.T) {
		f := &ws.Message{
			Payload: []byte{1, 1, 2, 2, 3, 3, 4, 4},
			Final:   true,
			Opcode:  ws.FrameOpcodeBinary,
		}

		exp := []byte{(1 << 7) | f.Opcode, 8}
		exp = append(exp, f.Payload...)

		if got, _ := f.Encode(); !slices.Equal(exp, got) {
			t.Errorf("exp %+v, got %+v", exp, got)
		}
	})

	t.Run("Extended payload", func(t *testing.T) {
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

		if got, _ := f.Encode(); !slices.Equal(exp, got) {
			t.Errorf(
				"2 byte EPL - exp %+v, got %+v",
				exp[:min(len(exp), 100)],
				got[:min(len(got), 100)],
			)
			return
		}

		f.Payload = bytes.Repeat([]byte{1}, math.MaxUint16*2)
		epl = make([]byte, 8)
		binary.Encode(epl, binary.BigEndian, uint64(len(f.Payload)))

		exp = []byte{f.Opcode, 127}
		exp = append(exp, epl...)
		exp = append(exp, f.Payload...)

		if got, _ := f.Encode(); !slices.Equal(exp, got) {
			t.Errorf(
				"8-byte EPL - exp %+v, got %+v",
				exp[:min(len(exp), 100)],
				got[:min(len(got), 100)],
			)
			return
		}
	})

	t.Run("Masked payload", func(t *testing.T) {
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

		if got, _ := f.Encode(); !slices.Equal(exp, got) {
			t.Errorf("exp %+v, got %+v", exp, got)
			return
		}
	})
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
