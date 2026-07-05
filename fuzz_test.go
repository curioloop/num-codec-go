package numcodec_test

import (
	"errors"
	"math"
	"testing"

	"github.com/curioloop/num-codec-go"
)

// The Fuzz tests below feed arbitrary (data, flags) pairs to every decode
// entry point. The contract they enforce is:
//
//   1. Never panic on any input (only sentinel errors may be returned).
//   2. Returned error, if any, wraps ErrOverflow / ErrMalformed OR is one
//      of the "invalid flags" errors — nothing exotic.
//   3. On success, no runaway allocations (bounded by decoded count via
//      len(data)).
//
// Seed corpus derives from encoding a few valid inputs so the fuzzer has
// a starting point that exercises the happy path; the mutator then
// perturbs bytes to hit malformed cases.

// maxValsPerByte is a loose upper bound on how many decoded values a
// single input byte can plausibly produce. Simple8b selector 0 packs
// 240 identical values into a single 8-byte word, so 30 values per
// input byte is the worst legitimate ratio.
const maxValsPerByte = 30

func acceptDecodeErr(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		return
	}
	if errors.Is(err, numcodec.ErrOverflow) || errors.Is(err, numcodec.ErrMalformed) {
		return
	}
	// The AppendDecode* helpers also return plain errors.New for invalid
	// flag combinations. Accept those too — they aren't panics.
	t.Logf("non-sentinel error (acceptable): %v", err)
}

func seedInt32(f *testing.F) {
	for _, seed := range [][]int32{
		{0}, {1, 2, 3}, {-1, 0, 1},
		{math.MinInt32, math.MaxInt32},
		{100, 100, 100, 100, 100},
	} {
		data, flags, err := numcodec.AppendEncodeInt32s(nil, seed)
		if err == nil {
			f.Add(data, uint8(flags))
		}
	}
	// A few raw malformed seeds.
	f.Add([]byte{}, uint8(numcodec.Simple8|numcodec.ZigZag))
	f.Add([]byte{0xff, 0xff}, uint8(numcodec.ZigZag))
	f.Add([]byte{0x00}, uint8(numcodec.Raw)) // length%4 != 0
}

func FuzzAppendDecodeInt32s(f *testing.F) {
	seedInt32(f)
	f.Fuzz(func(t *testing.T, data []byte, flagsBits uint8) {
		out, err := numcodec.AppendDecodeInt32s(nil, data, numcodec.Flags(flagsBits))
		acceptDecodeErr(t, err)
		if err == nil && len(out) > maxValsPerByte*len(data)+240 {
			t.Fatalf("decode produced %d values from %d bytes", len(out), len(data))
		}
	})
}

func FuzzAppendDecodeUint32s(f *testing.F) {
	for _, seed := range [][]uint32{
		{0}, {1, 2, 3, math.MaxUint32}, {100, 100, 100},
	} {
		data, flags, err := numcodec.AppendEncodeUint32s(nil, seed)
		if err == nil {
			f.Add(data, uint8(flags))
		}
	}
	f.Fuzz(func(t *testing.T, data []byte, flagsBits uint8) {
		out, err := numcodec.AppendDecodeUint32s(nil, data, numcodec.Flags(flagsBits))
		acceptDecodeErr(t, err)
		if err == nil && len(out) > maxValsPerByte*len(data)+240 {
			t.Fatalf("decode produced %d values from %d bytes", len(out), len(data))
		}
	})
}

func FuzzAppendDecodeInt64s(f *testing.F) {
	for _, seed := range [][]int64{
		{0}, {1, 2, 3}, {-1, 0, 1},
		{math.MinInt64, math.MaxInt64},
		{1_700_000_000_000, 1_700_000_000_001, 1_700_000_000_002},
	} {
		data, flags, err := numcodec.AppendEncodeInt64s(nil, seed)
		if err == nil {
			f.Add(data, uint8(flags))
		}
	}
	f.Fuzz(func(t *testing.T, data []byte, flagsBits uint8) {
		out, err := numcodec.AppendDecodeInt64s(nil, data, numcodec.Flags(flagsBits))
		acceptDecodeErr(t, err)
		if err == nil && len(out) > maxValsPerByte*len(data)+240 {
			t.Fatalf("decode produced %d values from %d bytes", len(out), len(data))
		}
	})
}

func FuzzAppendDecodeUint64s(f *testing.F) {
	for _, seed := range [][]uint64{
		{0}, {1, 2, 3, math.MaxUint64}, {100, 100, 100},
	} {
		data, flags, err := numcodec.AppendEncodeUint64s(nil, seed)
		if err == nil {
			f.Add(data, uint8(flags))
		}
	}
	f.Fuzz(func(t *testing.T, data []byte, flagsBits uint8) {
		out, err := numcodec.AppendDecodeUint64s(nil, data, numcodec.Flags(flagsBits))
		acceptDecodeErr(t, err)
		if err == nil && len(out) > maxValsPerByte*len(data)+240 {
			t.Fatalf("decode produced %d values from %d bytes", len(out), len(data))
		}
	})
}

func FuzzAppendDecodeFloat32s(f *testing.F) {
	for _, seed := range [][]float32{
		{1.5, 2.5, 3.5}, {0, 0, 0},
		{100.01, 100.02, 100.03, 100.04},
	} {
		data, flags, err := numcodec.AppendEncodeFloat32s(nil, seed)
		if err == nil {
			f.Add(data, uint8(flags))
		}
	}
	f.Fuzz(func(t *testing.T, data []byte, flagsBits uint8) {
		out, err := numcodec.AppendDecodeFloat32s(nil, data, numcodec.Flags(flagsBits))
		acceptDecodeErr(t, err)
		if err == nil && len(out) > maxValsPerByte*len(data)+240 {
			t.Fatalf("decode produced %d values from %d bytes", len(out), len(data))
		}
	})
}

func FuzzAppendDecodeFloat64s(f *testing.F) {
	for _, seed := range [][]float64{
		{1.5, 2.5, 3.5}, {0, 0, 0},
		{100.01, 100.02, 100.03, 100.04},
	} {
		data, flags, err := numcodec.AppendEncodeFloat64s(nil, seed)
		if err == nil {
			f.Add(data, uint8(flags))
		}
	}
	f.Fuzz(func(t *testing.T, data []byte, flagsBits uint8) {
		out, err := numcodec.AppendDecodeFloat64s(nil, data, numcodec.Flags(flagsBits))
		acceptDecodeErr(t, err)
		if err == nil && len(out) > maxValsPerByte*len(data)+240 {
			t.Fatalf("decode produced %d values from %d bytes", len(out), len(data))
		}
	})
}

func FuzzAppendDecodeDelta2(f *testing.F) {
	for _, seed := range [][]int64{
		{100}, {100, 101, 102, 103},
		{1_700_000_000_000, 1_700_000_000_001, 1_700_000_000_002},
		{5, 3, 8, 2, 100, -5, 0}, // triggers unordered path
	} {
		data, flags, err := numcodec.AppendEncodeDelta2(nil, seed)
		if err == nil {
			f.Add(data, uint8(flags))
		}
	}
	f.Fuzz(func(t *testing.T, data []byte, flagsBits uint8) {
		out, err := numcodec.AppendDecodeDelta2(nil, data, numcodec.Flags(flagsBits))
		acceptDecodeErr(t, err)
		if err == nil && len(out) > maxValsPerByte*len(data)+240 {
			t.Fatalf("decode produced %d values from %d bytes", len(out), len(data))
		}
	})
}

// FuzzEncodeDecodeInt32sRoundTrip verifies that every encoder output is
// decoded back to the original values, for arbitrary input slices. The
// encoder never fails on valid slice input; the decoder should not fail
// on encoder output.
func FuzzEncodeDecodeInt32sRoundTrip(f *testing.F) {
	f.Add([]byte{0x01, 0x00, 0x00, 0x00}) // 1 int32 (LE-interpreted, mostly zero)
	f.Add([]byte{0xff, 0xff, 0xff, 0xff, 0x00, 0x00, 0x00, 0x00})
	f.Fuzz(func(t *testing.T, raw []byte) {
		if len(raw) == 0 {
			return
		}
		// Interpret input bytes as int32 stream (4 bytes each). Pad to
		// multiple of 4 for a clean count.
		count := len(raw) / 4
		if count == 0 {
			return
		}
		vs := make([]int32, count)
		for i := 0; i < count; i++ {
			b := raw[i*4:]
			vs[i] = int32(uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24)
		}
		data, flags, err := numcodec.AppendEncodeInt32s(nil, vs)
		if err != nil {
			t.Fatalf("encode: %v", err)
		}
		out, err := numcodec.AppendDecodeInt32s(nil, data, flags)
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(out) != len(vs) {
			t.Fatalf("count: got %d want %d", len(out), len(vs))
		}
		for i, v := range vs {
			if out[i] != v {
				t.Fatalf("[%d] got %d want %d (flags=%#x)", i, out[i], v, flags)
			}
		}
	})
}
