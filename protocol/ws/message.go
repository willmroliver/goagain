package ws

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"math"
	"slices"
	"strings"
	"unsafe"

	"github.com/willmroliver/wsgo/core"
)

const (
	FrameOpcodeCont   = 0x0
	FrameOpcodeText   = 0x1
	FrameOpcodeBinary = 0x2
	FrameOpcodeClose  = 0x8
	FrameOpcodePing   = 0x9
	FrameOpcodePong   = 0xA

	StatusCodeNormalClosure    = 1000
	StatusCodeGoingAway        = 1001
	StatusCodeProtocolError    = 1002
	StatusCodeBadDataType      = 1003
	StatusCodeInconsistentData = 1007
	StatusCodePolicyViolated   = 1008
	StatusCodeMessageTooBig    = 1009
	StatusCodeNeedExtension    = 1010
	StatusCodeUnexpectedCond   = 1011
)

var (
	ErrBadFrame = errors.New("malformed WebSocket frame")

	CloseFrame = NewCloseFrame(0, "")
	PingFrame  = NewPingFrame()
	PongFrame  = NewPongFrame()
)

// Message serializes and parses individual
// WebSocket wire-format frames
type Message struct {
	Payload    []byte
	Final      bool
	Opcode     byte
	Masked     bool
	MaskingKey [4]byte
}

// Encode writes a serialized WebSocket Protocol frame
// to the underlying Conn
func (f *Message) Encode(c core.Conn) error {
	data, err := f.EncodeBytes()
	if err != nil {
		return err
	}

	_, err = c.Write(data)
	return err
}

// EncodeBytes serializes a WebSocket Protocol frame in accordance with
// [RFC6455] and ABNF [RFC5234].
func (f *Message) EncodeBytes() (data []byte, err error) {
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

	if _, err = buf.Write(f.Payload); err != nil {
		return
	}

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
func (f *Message) Decode(c core.Conn) (err error) {
	data, read, target := make([]byte, 2, 16), 0, 2
	var n, pl, mstart, pstart int

	for ; read < target; err = c.Buf().Fill() {
		if err != nil {
			return
		}
		if c.Buf().Available() < target {
			continue
		}

		n, err = c.Buf().Read(data[read : target-read])
		if err != nil {
			return
		}

		if read == 0 {
			f.Final = data[0]>>7 == 1
			f.Opcode = data[0] & 0x7

			pl = int(data[1] & 0x7)

			if 125 < pl && pl <= math.MaxUint16 {
				target += 2
			} else if math.MaxUint16 < pl {
				target += 8
			}

			if data[1]&0x8 != 0 {
				mstart = target
				f.Masked = true
				target += 4
			}

			pstart = target
			if pl < 126 {
				target += pl
			}
		} else if read == 2 {
			switch pl {
			case 126:
				_, err = binary.Decode(data[2:4], binary.BigEndian, &pl)
				target += pl
			case 127:
				_, err = binary.Decode(data[2:10], binary.BigEndian, &pl)
				target += pl
			}

			if err != nil {
				return
			}

			if f.Masked {
				copy(f.MaskingKey[:], data[mstart:mstart+4])
			}
		}

		read += n
	}

	copy(f.Payload, data[pstart:pstart+pl])
	return
}

// NewMaskingKey generates a 32-bit cryptographically
// secure key for masking and unmasking client frames
func (f *Message) NewMaskingKey() {
	rand.Read(f.MaskingKey[:])
}

// ApplyMask will mask an unmasked payload, or unmask
// a masked payload, as the WebSocket Protocol masking
// algorithm is its own inverse
func (f *Message) ApplyMask() {
	for i := range f.Payload {
		f.Payload[i] ^= f.MaskingKey[i%4]
	}

	f.Masked = !f.Masked
}

// UnsafeMask bypasses Go type-safety to perform the XOR
// operations in 64-bit chunks: 6.5x faster than ApplyMask
func (f *Message) UnsafeMask() {
	n := len(f.Payload)
	bytes := unsafe.SliceData(slices.Repeat(f.MaskingKey[:], 2))
	key64 := *(*uint64)(unsafe.Pointer(bytes))

	var payload64 *uint64
	var i int

	for ; i+8 <= n; i += 8 {
		bytes = unsafe.SliceData(f.Payload[i : i+8])
		payload64 = (*uint64)(unsafe.Pointer(bytes))
		*payload64 ^= key64
	}

	for ; i < n; i++ {
		f.Payload[i] ^= f.MaskingKey[i%4]
	}
}

func NewCloseFrame(status uint16, reason string) []byte {
	m := &Message{
		Opcode: FrameOpcodeClose,
	}

	if status != 0 {
		var b strings.Builder
		binary.Write(&b, binary.BigEndian, status)
		b.WriteString(reason)
		m.Payload = []byte(b.String())
	}

	return newControlFrame(m)
}

func NewPingFrame() []byte {
	return newControlFrame(&Message{
		Opcode: FrameOpcodePing,
	})
}

func NewPongFrame() []byte {
	return newControlFrame(&Message{
		Opcode: FrameOpcodePong,
	})
}

func newControlFrame(m *Message) []byte {
	data, err := m.EncodeBytes()
	if err != nil || len(data) > 125 {
		data = nil
	}

	return data
}
