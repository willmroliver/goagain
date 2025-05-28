package ws

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"math"
)

const (
	FrameOpcodeCont   = 0x0
	FrameOpcodeText   = 0x1
	FrameOpcodeBinary = 0x2
	FrameOpcodeClose  = 0x8
	FrameOpcodePing   = 0x9
	FrameOpcodePong   = 0xA
)

var ErrBadFrame = errors.New("malformed WebSocket frame")

// Message serializes and parses individual
// WebSocket wire-format frames
type Message struct {
	Payload    []byte
	Final      bool
	Opcode     byte
	Masked     bool
	MaskingKey [4]byte
}

// Encode serializes a WebSocket Protocol frame in accordance with
// [RFC6455] and ABNF [RFC5234].
func (f *Message) Encode() (data []byte, err error) {
	// FIN + RSV1-3 + opcode + MASK + Payload len
	n := 2

	// Extended payload len
	pl, epl := len(f.Payload), 0

	if 125 < pl && pl <= math.MaxUint16 {
		epl = 2
	} else if math.MaxUint16 < pl {
		epl = 8
	}

	n += epl + pl

	// Masking key
	if f.Masked {
		n += 4
	}

	buf := new(bytes.Buffer)
	buf.Grow(n)

	var b [2]byte
	if f.Final {
		b[0] = 1
	}

	b[0] = (b[0] << 7) | f.Opcode
	if f.Masked {
		b[1] = (1 << 7)
	}

	switch {
	case epl == 0:
		b[1] |= byte(pl)
		buf.Write(b[:])
	case epl == 2:
		b[1] |= 126
		buf.Write(b[:])
		err = binary.Write(buf, binary.BigEndian, uint16(pl))
	case epl == 8:
		b[1] |= 127
		buf.Write(b[:])
		err = binary.Write(buf, binary.BigEndian, uint64(pl))
	default:
		panic("invalid payload len!")
	}

	if err != nil {
		return
	}

	if f.Masked {
		buf.Write(f.MaskingKey[:])
	}

	buf.Write(f.Payload)
	data = buf.Bytes()

	return
}

// Decode parses WebSocket Protocol frame bytes in accordance with
// [RFC6455] and ABNF [RFC5234].
//
// f.Payload is copied from data, so mutations to data
// after decoding do not affect f after Decode completes.
//
// If f.Payload is masked, Decode sets f.Masked and does not ApplyMask
func (f *Message) Decode(data []byte) (err error) {
	n := len(data)
	if n < 2 {
		err = ErrBadFrame
		return
	}

	pstart := 2

	f.Final = data[0]>>7 == 1
	f.Opcode = data[0] & 0x7

	pl := int(data[1] & 0x7)
	if 125 < pl && pl <= math.MaxUint16 {
		pstart += 2
		_, err = binary.Decode(data[2:4], binary.BigEndian, &pl)
	} else if math.MaxUint16 < pl {
		pstart += 8
		// @todo - proper handling of payloads > 2^32?
		_, err = binary.Decode(data[2:10], binary.BigEndian, &pl)
	}

	if err != nil {
		return
	}

	if n < pstart+pl {
		err = ErrBadFrame
		return
	}

	if data[1]&0x8 != 0 {
		f.Masked = true
		pstart += 4
		copy(f.MaskingKey[:], data[pstart-4:pstart])
	}

	f.Payload = make([]byte, pl)
	copy(f.Payload, data[pstart:pl])

	return
}

// NewMaskingKey generates a 32-bit cryptographically
// secure key for masking and unmasking client frames
func (f *Message) NewMaskingKey() {
	rand.Read(f.MaskingKey[:])
}

// ApplyMask will mask an unmasked payload, or unmask
// a masked payload, as the WebSocket Protocol masking
// algrithm is its own inverse
func (f *Message) ApplyMask() {
	for i := range f.Payload {
		f.Payload[i] ^= f.MaskingKey[i%4]
	}

	f.Masked = !f.Masked
}
