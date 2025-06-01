package ws

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"io"
	"math"
	"slices"
	"strings"
	"unsafe"

	"github.com/willmroliver/wsgo/core"
)

const (
	OpcodeCont   byte = 0x0
	OpcodeText   byte = 0x1
	OpcodeBinary byte = 0x2
	OpcodeClose  byte = 0x8
	OpcodePing   byte = 0x9
	OpcodePong   byte = 0xA

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
	PingFrame  = NewMessage(OpcodePing)
	PongFrame  = NewMessage(OpcodePong)
)

type FrameHeader struct {
	FIN, MASK        bool
	RSV1, RSV2, RSV3 bool
	Opcode           byte
	PL, EPL          int
	MaskingKey       [4]byte
}

// Message serializes and parses individual
// WebSocket wire-format frames
type Message struct {
	FrameHeader
	Payload []byte
}

func NewMessage(op byte) *Message {
	return &Message{
		FrameHeader: FrameHeader{
			Opcode: op,
		},
	}
}

func (f *Message) SetPayload(p []byte) *Message {
	f.Payload, f.PL = p, len(p)
	return f
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
	var b [2]byte
	if f.FIN {
		b[0] = 1
	}

	b[0] = (b[0] << 7) | f.Opcode
	if f.MASK {
		b[1] = (1 << 7)
	}

	pl := len(f.Payload)

	switch {
	case pl > math.MaxUint16:
		b[1] |= 127
	case pl > 125:
		b[1] |= 126
	case pl >= 0:
		b[1] |= byte(pl)
	}

	buf := new(bytes.Buffer)
	buf.Grow(pl + 14)
	buf.Write(b[:])

	switch {
	case pl > math.MaxUint16:
		err = binary.Write(buf, binary.BigEndian, uint64(pl))
	case pl > 125:
		err = binary.Write(buf, binary.BigEndian, uint16(pl))
	}

	if err != nil {
		return
	}

	if f.MASK {
		if _, err = buf.Write(f.MaskingKey[:]); err != nil {
			return
		}
	}

	if _, err = buf.Write(f.Payload); err != nil {
		return
	}

	data = buf.Bytes()
	return
}

// Decode parses frame bytes in accordance with
// [RFC6455] and ABNF [RFC5234].
//
// f.Payload is copied from data, so mutations to data
// after decoding do not affect f after Decode completes.
//
// If f.Payload is masked, Decode sets f.MASK and does not ApplyMask
func (f *Message) Decode(c core.Conn) (err error) {
	if err = f.decodeHeader(c); err != nil {
		return
	}

	f.Payload = make([]byte, f.PL)
	buf, read, n := c.Buf(), 0, 0

	for buf.Available() < f.PL-read {
		if err = buf.Fill(); err != nil {
			return
		}
		if !buf.Full() {
			continue
		}

		n, err = buf.Read(f.Payload[read:])
		if err != nil && err != io.EOF {
			return
		}

		read += n
	}

	if _, err = buf.Read(f.Payload[read:]); err == io.EOF {
		err = nil
	}

	return
}

func (f *Message) decodeHeader(c core.Conn) (err error) {
	buf, read, target := c.Buf(), 0, 2
	data := make([]byte, 2, 16)

	for buf.Available() < target {
		if err = buf.Fill(); err != nil {
			return
		}
	}

	var n int
	if n, err = buf.Read(data); err != nil {
		return
	}

	read += n

	f.FIN = (data[0] & 0x80) == 1
	f.Opcode = data[0] & 0x7

	f.PL = int(data[1] & 0x7f)

	switch f.PL {
	case 126:
		target += 2
	case 127:
		target += 8
	}

	var mstart int

	if data[1]&0x80 != 0 {
		f.MASK = true
		mstart = target
		target += 4
	}

	for buf.Available() < target-read {
		if err = buf.Fill(); err != nil {
			return
		}
	}

	data = data[:target]

	n, err = c.Buf().Read(data[read:])
	if err != nil && err != io.EOF {
		return
	}

	read += n

	if f.MASK {
		copy(f.MaskingKey[:], data[mstart:mstart+4])
	}

	// in theory, this will truncate / sign-invert
	// on a 32-bit OS if payload size >= 2GiB
	switch f.PL {
	case 126:
		f.PL = int(binary.BigEndian.Uint16(data[2:4]))
	case 127:
		f.PL = int(binary.BigEndian.Uint64(data[2:10]))
	}

	return
}

// NewMaskingKey generates a 32-bit cryptographically
// secure key for masking and unmasking client frames
func (f *Message) NewMaskingKey() *Message {
	rand.Read(f.MaskingKey[:])
	return f
}

// ApplyMask masks an unmasked payload, and unmasks
// a masked payload (the XOR-based algorithm is its own
// inverse)
func (f *Message) ApplyMask() *Message {
	key64 := binary.LittleEndian.Uint64(bytes.Repeat(f.MaskingKey[:], 2))

	var i int

	for ; i+8 <= f.PL; i += 8 {
		binary.LittleEndian.PutUint64(
			f.Payload[i:],
			binary.LittleEndian.Uint64(f.Payload[i:])^key64,
		)
	}

	for ; i < f.PL; i++ {
		f.Payload[i] ^= f.MaskingKey[i%4]
	}

	f.MASK = !f.MASK

	return f
}

// UnsafeMask bypasses type-safety & `package binary` function calls to perform
// 64-bit XOR directly on payload memory: >2x faster than ApplyMask
func (f *Message) UnsafeMask() *Message {
	n := len(f.Payload)
	bytes := unsafe.SliceData(slices.Repeat(f.MaskingKey[:], 2))
	key64 := *(*uint64)(unsafe.Pointer(bytes))
	payload64 := (unsafe.Pointer(unsafe.SliceData(f.Payload)))

	var i int

	for ; i+8 <= n; i += 8 {
		*(*uint64)(unsafe.Add(payload64, i)) ^= key64
	}

	for ; i < n; i++ {
		f.Payload[i] ^= f.MaskingKey[i%4]
	}

	f.MASK = !f.MASK

	return f
}

func NewCloseFrame(status uint16, reason string) *Message {
	m := NewMessage(OpcodeClose)

	if status != 0 {
		var b strings.Builder
		binary.Write(&b, binary.BigEndian, status)
		b.WriteString(reason)
		m.Payload = []byte(b.String())
	}

	return m
}
