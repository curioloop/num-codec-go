package varint_test

import (
	"errors"
	"math"
	"testing"

	"github.com/curioloop/num-codec-go/internal/codecerr"
	"github.com/curioloop/num-codec-go/varint"
)

func TestUvarint32RoundTrip(t *testing.T) {
	values := []uint32{0, 1, 127, 128, 300, 16383, 16384, 1 << 21, 1 << 28, math.MaxUint32}
	data := varint.AppendUvarint32s(nil, values)
	out, err := varint.AppendDecodeUvarint32s(nil, data)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(out) != len(values) {
		t.Fatalf("count: got %d want %d", len(out), len(values))
	}
	for i, v := range values {
		if out[i] != v {
			t.Fatalf("[%d] got %d want %d", i, out[i], v)
		}
	}
}

func TestZigZag32RoundTrip(t *testing.T) {
	values := []int32{0, -1, 1, -2, 2, -63, 63, -64, 64, math.MinInt32, math.MaxInt32}
	data := varint.AppendZigZag32s(nil, values)
	out, err := varint.AppendDecodeZigZag32s(nil, data)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	for i, v := range values {
		if out[i] != v {
			t.Fatalf("[%d] got %d want %d", i, out[i], v)
		}
	}
}

func TestUvarint64RoundTrip(t *testing.T) {
	values := []uint64{0, 1, 127, 128, 300, 1 << 35, 1 << 56, math.MaxUint64}
	data := varint.AppendUvarint64s(nil, values)
	out, err := varint.AppendDecodeUvarint64s(nil, data)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	for i, v := range values {
		if out[i] != v {
			t.Fatalf("[%d] got %d want %d", i, out[i], v)
		}
	}
}

func TestZigZag64RoundTrip(t *testing.T) {
	values := []int64{0, -1, 1, math.MinInt64, math.MaxInt64, -(1 << 40), 1 << 40}
	data := varint.AppendZigZag64s(nil, values)
	out, err := varint.AppendDecodeZigZag64s(nil, data)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	for i, v := range values {
		if out[i] != v {
			t.Fatalf("[%d] got %d want %d", i, out[i], v)
		}
	}
}

// TestEncodeAppendsToDst verifies AppendUvarint32s leaves any pre-existing
// bytes in dst untouched.
func TestEncodeAppendsToDst(t *testing.T) {
	prefix := []byte{0xAA, 0xBB, 0xCC}
	dst := append([]byte(nil), prefix...)
	dst = varint.AppendUvarint32s(dst, []uint32{0, 1, 2, 3, 4})
	for i, want := range prefix {
		if dst[i] != want {
			t.Fatalf("prefix corrupted at %d: got %#x want %#x", i, dst[i], want)
		}
	}
}

// TestMalformedUvarint verifies truncated / oversized inputs raise
// ErrMalformed rather than panicking.
func TestMalformedUvarint(t *testing.T) {
	// A single 0xff byte with no continuation.
	if _, _, err := varint.Uvarint32([]byte{0xff}); !errors.Is(err, codecerr.ErrMalformed) {
		t.Fatalf("expected ErrMalformed, got %v", err)
	}
	// 5 bytes with high bits set (overflow).
	if _, _, err := varint.Uvarint32([]byte{0xff, 0xff, 0xff, 0xff, 0xff}); !errors.Is(err, codecerr.ErrMalformed) {
		t.Fatalf("expected ErrMalformed, got %v", err)
	}
	// 10th byte > 0x01 → uint64 overflow.
	if _, _, err := varint.Uvarint64([]byte{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x02}); !errors.Is(err, codecerr.ErrMalformed) {
		t.Fatalf("expected ErrMalformed, got %v", err)
	}
}
