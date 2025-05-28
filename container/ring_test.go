package container_test

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"
	"testing"

	"github.com/willmroliver/goagain/container"
)

func TestNewRing(t *testing.T) {
	type Test struct {
		size, exp uint
	}

	doTest := func(test *Test) bool {
		r := container.NewRing[int](test.size)
		if r.Cap() == test.exp {
			return true
		}

		t.Errorf(
			"%s failed for size %d: exp %d, got %d\n",
			t.Name(),
			test.size,
			test.exp,
			r.Cap(),
		)
		return false
	}

	tests := []*Test{
		{0, 0x1000},
		{0x1, 0x1},
		{0x100, 0x100},
		{0xfffff, 0x1000},
		{0x10000000, 0x10000000},
		{38, 0x1000},
	}

	for _, test := range tests {
		doTest(test)
	}
}

func TestPush(t *testing.T) {
	type Test struct {
		size, fill uint
	}

	doTest := func(test *Test) bool {
		r := container.NewRing[int](test.size)
		for range test.fill {
			r.Push(1)
		}

		exp := test.fill < r.Cap()

		if got := r.Push(1); got != exp {
			t.Errorf(
				"%s failed: exp %t, got %t",
				t.Name(),
				exp,
				got,
			)
			return false
		}

		size := min(test.fill, r.Cap())
		if exp {
			size++
		}

		if got := r.Size(); got != size {
			t.Errorf(
				"%s failed: exp %d, got %d",
				t.Name(),
				size,
				got,
			)
			return false
		}

		return true
	}

	tests := []*Test{
		{0x1, 0},
		{0x1, 1},
		{0x100, 0xff},
		{0x100, 0x111},
		{0xff, 0xfff},
		{0xff, 0x1000},
	}

	for _, test := range tests {
		doTest(test)
	}
}

func BenchmarkPush(t *testing.B) {
	t.Run("Push ints", func(t *testing.B) {
		r := container.NewRing[int](math.MaxUint32)
		for t.Loop() {
			r.Push(1)
		}
	})

	t.Run("Push structs", func(t *testing.B) {
		type Type struct {
			ints [10]int
			strs [10]string
		}

		r := container.NewRing[Type](math.MaxUint32)
		for t.Loop() {
			r.Push(Type{
				[10]int{1},
				[10]string{"abdefghijklmnopqrstuvwxyz0123456789"},
			})
		}
	})
}

func TestPop(t *testing.T) {
	type Test struct {
		size, fill uint
	}

	doTest := func(test *Test) bool {
		r := container.NewRing[int](test.size)
		gotVal, expVal := 0, int(r.Cap())

		for i := range test.fill {
			r.Push(int(r.Cap() - i))
		}

		exp := test.fill > 0

		if got := r.Pop(&gotVal); got != exp || got && gotVal != expVal {
			t.Errorf(
				"%s failed: exp (%t, %d), got (%t, %d)\n",
				t.Name(),
				exp,
				expVal,
				got,
				gotVal,
			)
			return false
		}

		size := min(test.fill, r.Cap())
		if exp {
			size--
		}

		if got := r.Size(); got != size {
			t.Errorf(
				"%s failed: exp %d, got %d\n",
				t.Name(),
				size,
				got,
			)
			return false
		}

		return true
	}

	tests := []*Test{
		{0x1, 0},
		{0x1, 1},
		{0x10, 0},
		{0x10, 1},
		{0x10, 0x11},
		{0xff, 0},
		{0xff, 1},
		{0xff, 0x1001},
		{0x10000, 0},
		{0x10000, 1},
		{0x10000, 0x10001},
	}

	for _, test := range tests {
		doTest(test)
	}
}

func TestClear(t *testing.T) {
	r := container.NewRing[byte](0x1000)
	r.Write(bytes.Repeat([]byte{1, 2, 3, 4}, 16))

	if r.Empty() {
		t.Errorf("exp not empty")
	}

	r.Clear()

	if r.Full() || !r.Empty() || r.Size() != 0 {
		t.Errorf("exp empty")
	}
}

func TestWriteFunc(t *testing.T) {
	w := func(b []byte, arg any) (n int, err error) {
		data := arg.([]byte)

		var m int

		if n, m = len(data), len(b); m < n {
			err = io.EOF
			n = m
		}

		if n == 0 {
			err = errors.New("Buffer full")
			return
		}

		copy(b[:n], data[:n])
		return
	}

	t.Run("Write without wrap", func(t *testing.T) {
		var n uint = 0x100
		r := container.NewRing[byte](n)
		data := []byte(strings.Repeat(string([]byte{1}), int(n)))

		r.WriteFunc(w, data)

		if size := r.Size(); size != n {
			t.Errorf("exp %d, got %d\n", n, size)
			return
		}
		if !r.Full() {
			t.Error("exp full, got not full")
			return
		}
		if r.Empty() {
			t.Error("exp not empty, got empty")
		}

		var b byte
		for range n {
			if r.Pop(&b); b != 1 {
				t.Errorf("exp 1, got %v\n", b)
			}
		}

		if r.Pop(&b) {
			t.Errorf("pop success, exp fail")
		}
	})

	t.Run("Write with wrap", func(t *testing.T) {
		var n uint = 0x8
		r := container.NewRing[byte](n)
		data := []byte(strings.Repeat(string([]byte{2}), int(n)))

		if m, _ := r.WriteFunc(w, data[:(n/2)]); m != int(n/2) {
			t.Errorf("exp %d, got %d\n", n/2, m)
			return
		}

		for range n / 2 {
			var b byte
			r.Pop(&b)
		}

		if !r.Empty() {
			t.Errorf("expected empty after pushing and popping %d\n", n/2)
			return
		}

		if m, _ := r.WriteFunc(w, data); m != int(n) {
			t.Errorf("exp %d, got %d", n, m)
			return
		}

		if f, e, m := r.Full(), r.Empty(), r.Size(); !f || e || m != n {
			t.Errorf(
				"exp %t, %t, %d, got %t, %t, %d",
				true, false, n, f, e, m,
			)
			return
		}
	})
}

func TestWrite(t *testing.T) {
	r := container.NewRing[byte](0x10)
	buf := bufio.NewWriter(r)
	data := []byte("0123456789")

	buf.Write(data)
	buf.Flush()

	if size, exp := r.Size(), uint(10); size != exp {
		t.Errorf("exp %d, got %d\n", exp, size)
	}

	for i := range 10 {
		var b byte
		if success := r.Pop(&b); !success || b != byte(i)+'0' {
			t.Errorf("exp (%t, %d), got (%t, %d)", true, i+'0', success, b)
		}
	}

	n, _ := buf.Write(data)
	m, _ := buf.Write(data)
	buf.Flush()

	if exp := r.Size() + uint(buf.Buffered()); uint(n+m) != exp {
		t.Errorf("exp %d, got %d\n", exp, n+m)
		return
	}
}

func TestWriteTo(t *testing.T) {
	r, s := container.NewRing[byte](0x10), container.NewRing[byte](0x10)
	data := []byte("12345678")

	r.Write(data)
	r.Write(data)

	io.Copy(s, r)

	if !r.Empty() || !s.Full() {
		t.Errorf("exp r empty, s full\n")
	}
}

func TestRead(t *testing.T) {
	r := container.NewRing[byte](0x10)
	buf := bufio.NewReader(r)
	data := []byte("12345678")

	for i := range r.Cap() {
		r.Push(byte(data[i%uint(len(data))]))
	}

	res, err := buf.ReadString(255)

	if err != io.EOF {
		t.Errorf("exp %v, got %v\n", io.EOF, err)
		return
	}
	if exp := strings.Repeat(string(data), 2); res != exp {
		t.Errorf("exp %s, got %s\n", exp, res)
		return
	}
}

type hideWriteTo struct {
	io.Reader
}

func TestReadFrom(t *testing.T) {
	r, s := container.NewRing[byte](0x10), container.NewRing[byte](0x10)
	data := []byte("123456789")

	reader := hideWriteTo{r}

	for range 4 {
		r.Write(data)
		io.Copy(s, reader)
		s.Read(data)

		if exp := "123456789"; string(data) != exp {
			t.Errorf("exp %s, got %s\n", exp, string(data))
			return
		}
	}
}

func TestHasSuffix(t *testing.T) {
	n := 0x10

	type Test struct {
		offset         int
		suffix, search string
	}

	tests := []*Test{
		{0, "1234", "1234"},
		{n - 4, "1234", "1234"},
		{n - 2, "1234", "1234"},
		{n - 1, "1234", "1234"},
		{n, "1234", "1234"},
		{2*n - 2, "1234", "1234"},
		{0, "1234", "12"},
		{n - 4, "1234", "34"},
		{n - 2, "1234", "123456"},
		{n - 1, "1234", "123"},
		{n, "1234", "234"},
		{2*n - 2, "1234", "34"},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			r := container.NewRing[byte](uint(n))
			b, n := byte(0), len(test.search)
			exp := len(test.suffix) >= n &&
				test.suffix[len(test.suffix)-n:] == test.search

			for range test.offset {
				r.Push(0)
				r.Pop(&b)
			}

			r.Write([]byte(test.suffix))

			if got := r.HasSuffix([]byte(test.search)); exp != got {
				t.Errorf(
					"suffix: %s, search: %s, exp: %t, got: %t\n",
					test.suffix, test.search, exp, got,
				)
				return
			}
		})
	}
}

func TestIndexOf(t *testing.T) {
	n := 0x10

	type Test struct {
		offset           int
		haystack, needle string
	}

	tests := []*Test{
		{0, "", ""},
		{0, "1", "1"},
		{n - 1, "1", "1"},
		{n, "1", "1"},
		{n + 1, "1", "1"},
		{0, "12", "12"},
		{0, "123", "1"},
		{0, "123", "2"},
		{0, "123", "3"},
		{n - 1, "123", "1"},
		{n, "123", "2"},
		{n + 1, "123", "3"},
		{0, "1234", "12"},
		{n - 2, "12345", "234"},
		{n - 1, "12345", "234"},
		{n, "12345", "123"},
		{n - 4, "12345", "345"},
		{n - 3, "12345", "345"},
		{n - 2, "12345", "345"},
		{n - 1, "12345", "345"},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			exp := strings.Index(test.haystack, test.needle)

			r, b := container.NewRing[byte](uint(n)), byte(0)

			for range test.offset {
				r.Push(0)
				r.Pop(&b)
			}

			r.Write([]byte(test.haystack))

			if got := r.IndexOf([]byte(test.needle)); exp != got {
				t.Errorf("exp %d, got %d", exp, got)
				return
			}
		})
	}
}
