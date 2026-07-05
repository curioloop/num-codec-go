package gorilla_test

import (
	"math"
	"testing"

	"github.com/curioloop/num-codec-go/gorilla"
)

func roundTripDoubles(t *testing.T, name string, values []float64) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		data := gorilla.AppendEncodeFloat64s(nil, values)
		out, err := gorilla.AppendDecodeFloat64s(nil, data)
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(out) != len(values) {
			t.Fatalf("count: got %d want %d", len(out), len(values))
		}
		for i, v := range values {
			if math.Float64bits(out[i]) != math.Float64bits(v) {
				t.Fatalf("[%d] bits differ: got %#x want %#x", i, math.Float64bits(out[i]), math.Float64bits(v))
			}
		}
	})
}

func roundTripFloats(t *testing.T, name string, values []float32) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		data := gorilla.AppendEncodeFloat32s(nil, values)
		out, err := gorilla.AppendDecodeFloat32s(nil, data)
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(out) != len(values) {
			t.Fatalf("count: got %d want %d", len(out), len(values))
		}
		for i, v := range values {
			if math.Float32bits(out[i]) != math.Float32bits(v) {
				t.Fatalf("[%d] bits differ: got %#x want %#x", i, math.Float32bits(out[i]), math.Float32bits(v))
			}
		}
	})
}

func TestGorilla64RoundTrip(t *testing.T) {
	roundTripDoubles(t, "identical", []float64{1.5, 1.5, 1.5, 1.5, 1.5})
	roundTripDoubles(t, "monotonic", []float64{1.1, 1.2, 1.3, 1.4, 1.5, 1.6})
	roundTripDoubles(t, "prices", []float64{100.01, 100.02, 100.02, 100.03, 100.02, 100.05, 100.05, 100.06})
	roundTripDoubles(t, "zeros", []float64{0, 0, 0, 0.5, 0.5, 0, 0})
	// NOTE: Gorilla silently corrupts pathological XORs where the top and
	// bottom bits of `previous^current` are both set. Java has the same
	// behaviour; real time-series data never hits it.
}

func TestGorilla32RoundTrip(t *testing.T) {
	roundTripFloats(t, "identical", []float32{2.5, 2.5, 2.5, 2.5})
	roundTripFloats(t, "monotonic", []float32{1, 2, 3, 4, 5, 6, 7})
	roundTripFloats(t, "prices", []float32{50.01, 50.02, 50.02, 50.03})
	roundTripFloats(t, "zeros", []float32{0, 0, 0.5, 0.5, 0})
}

// TestAppendPreservesPrefix ensures existing dst bytes are not overwritten.
func TestAppendPreservesPrefix(t *testing.T) {
	prefix := []byte{0x11, 0x22, 0x33}
	dst := append([]byte(nil), prefix...)
	dst = gorilla.AppendEncodeFloat64s(dst, []float64{1, 2, 3, 4})
	for i, want := range prefix {
		if dst[i] != want {
			t.Fatalf("prefix byte %d clobbered: got %#x want %#x", i, dst[i], want)
		}
	}
}

// TestSignFlipLsbFlipRoundTrip64 covers the historic XOR bit-63 + bit-0
// corner case where the encoder used a signed diffSize that overflowed
// the 7-bit blockSize field via int32 sign extension, silently corrupting
// the leading-zero field. Fixed in v2 (Go & Java).
func TestSignFlipLsbFlipRoundTrip64(t *testing.T) {
	prev := 1.5
	curr := math.Float64frombits(math.Float64bits(prev) ^ 0x8000000000000001)
	values := []float64{prev, curr}

	data := gorilla.AppendEncodeFloat64s(nil, values)
	out, err := gorilla.AppendDecodeFloat64s(nil, data)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("count: got %d want 2", len(out))
	}
	if math.Float64bits(out[0]) != math.Float64bits(prev) {
		t.Fatalf("prev bits differ: got %#x want %#x", math.Float64bits(out[0]), math.Float64bits(prev))
	}
	if math.Float64bits(out[1]) != math.Float64bits(curr) {
		t.Fatalf("curr bits differ (XOR bit-63+bit-0 corner case): got %#x want %#x",
			math.Float64bits(out[1]), math.Float64bits(curr))
	}
}

// TestSignFlipLsbFlipRoundTrip32 is the float32 counterpart. On the old
// encoder this even threw ErrOverflow (not silent corruption) because the
// int32 sign extension pushed diffSize past the 7-bit overflow check.
func TestSignFlipLsbFlipRoundTrip32(t *testing.T) {
	prev := float32(1.5)
	curr := math.Float32frombits(math.Float32bits(prev) ^ 0x80000001)
	values := []float32{prev, curr}

	data := gorilla.AppendEncodeFloat32s(nil, values)
	out, err := gorilla.AppendDecodeFloat32s(nil, data)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("count: got %d want 2", len(out))
	}
	if math.Float32bits(out[0]) != math.Float32bits(prev) {
		t.Fatalf("prev bits differ")
	}
	if math.Float32bits(out[1]) != math.Float32bits(curr) {
		t.Fatalf("curr bits differ (XOR bit-31+bit-0 corner case)")
	}
}
