package numcodec_test

import (
	"math"
	"math/rand"
	"testing"

	"github.com/curioloop/num-codec-go"
)

// TestHelperInts covers signed int32 round trips via ZigZag.
func TestHelperInts(t *testing.T) {
	cases := []struct {
		name   string
		values []int32
	}{
		{"tiny_signed", []int32{-3, -1, 0, 1, 5, 42, -42}},
		{"single_value_repeats", make([]int32, 300)},
		{"increasing", func() []int32 {
			s := make([]int32, 1000)
			for i := range s {
				s[i] = int32(i)
			}
			return s
		}()},
		{"mixed_magnitude", []int32{0, 1, 2, 1 << 20, 3, 4, 1 << 25, 5}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data, flags, err := numcodec.AppendEncodeInt32s(nil, tc.values)
			if err != nil {
				t.Fatalf("encode: %v", err)
			}
			out, err := numcodec.AppendDecodeInt32s(nil, data, flags)
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			if len(out) != len(tc.values) {
				t.Fatalf("count: got %d want %d", len(out), len(tc.values))
			}
			for i, v := range tc.values {
				if out[i] != v {
					t.Fatalf("[%d] got %d want %d (flags=%#x)", i, out[i], v, flags)
				}
			}
		})
	}
}

// TestHelperUints covers unsigned uint32 round trips via Uvarint.
func TestHelperUints(t *testing.T) {
	values := []uint32{0, 1, 5, 42, 100, 1000, 10000, math.MaxUint32}
	data, flags, err := numcodec.AppendEncodeUint32s(nil, values)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	out, err := numcodec.AppendDecodeUint32s(nil, data, flags)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	for i, v := range values {
		if out[i] != v {
			t.Fatalf("[%d] got %d want %d (flags=%#x)", i, out[i], v, flags)
		}
	}
}

func TestHelperLongs(t *testing.T) {
	cases := []struct {
		name   string
		values []int64
	}{
		{"tiny_signed", []int64{-3, -1, 0, 1, math.MinInt64, math.MaxInt64}},
		{"big_uniform", func() []int64 {
			s := make([]int64, 500)
			for i := range s {
				s[i] = int64(i) + 1_700_000_000
			}
			return s
		}()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data, flags, err := numcodec.AppendEncodeInt64s(nil, tc.values)
			if err != nil {
				t.Fatalf("encode: %v", err)
			}
			out, err := numcodec.AppendDecodeInt64s(nil, data, flags)
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			for i, v := range tc.values {
				if out[i] != v {
					t.Fatalf("[%d] got %d want %d (flags=%#x)", i, out[i], v, flags)
				}
			}
		})
	}
}

func TestHelperULongs(t *testing.T) {
	values := []uint64{0, 1, 100, 1 << 40, 1 << 55, math.MaxUint64}
	data, flags, err := numcodec.AppendEncodeUint64s(nil, values)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	out, err := numcodec.AppendDecodeUint64s(nil, data, flags)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	for i, v := range values {
		if out[i] != v {
			t.Fatalf("[%d] got %d want %d (flags=%#x)", i, out[i], v, flags)
		}
	}
}

func TestHelperFloats(t *testing.T) {
	prices := []float32{50.01, 50.02, 50.02, 50.03, 50.05, 50.05, 50.06, 50.05, 50.05, 50.07, 50.10}
	data, flags, err := numcodec.AppendEncodeFloat32s(nil, prices)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	out, err := numcodec.AppendDecodeFloat32s(nil, data, flags)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	for i, v := range prices {
		if math.Float32bits(out[i]) != math.Float32bits(v) {
			t.Fatalf("[%d] bits differ (flags=%#x)", i, flags)
		}
	}
}

func TestHelperDoubles(t *testing.T) {
	prices := []float64{100.01, 100.02, 100.02, 100.03, 100.02, 100.05, 100.05, 100.06, 100.06, 100.07, 100.10}
	data, flags, err := numcodec.AppendEncodeFloat64s(nil, prices)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	out, err := numcodec.AppendDecodeFloat64s(nil, data, flags)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	for i, v := range prices {
		if math.Float64bits(out[i]) != math.Float64bits(v) {
			t.Fatalf("[%d] bits differ (flags=%#x)", i, flags)
		}
	}
}

func TestHelperDelta2(t *testing.T) {
	// ordered timestamps → Simple8
	ts := make([]int64, 100)
	base := int64(1_700_000_000_000)
	for i := range ts {
		ts[i] = base + int64(i)*1000
	}
	data, flags, err := numcodec.AppendEncodeDelta2(nil, ts)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if flags != numcodec.Delta2|numcodec.Simple8 {
		t.Fatalf("expected Delta2|Simple8, got %#x", flags)
	}
	out, err := numcodec.AppendDecodeDelta2(nil, data, flags)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	for i, v := range ts {
		if out[i] != v {
			t.Fatalf("[%d] got %d want %d", i, out[i], v)
		}
	}

	// unordered → ZigZag
	rnd := rand.New(rand.NewSource(42))
	unordered := make([]int64, 50)
	unordered[0] = 1000
	for i := 1; i < len(unordered); i++ {
		unordered[i] = unordered[i-1] + int64(rnd.Intn(2000)-1000)
	}
	data2, flags2, err := numcodec.AppendEncodeDelta2(nil, unordered)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	out2, err := numcodec.AppendDecodeDelta2(nil, data2, flags2)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	for i, v := range unordered {
		if out2[i] != v {
			t.Fatalf("[%d] got %d want %d (flags=%#x)", i, out2[i], v, flags2)
		}
	}
}

// TestHelperFallbacksToRaw exercises the "compressed > raw" fallback path.
func TestHelperFallbacksToRaw(t *testing.T) {
	rnd := rand.New(rand.NewSource(1))
	values := make([]uint32, 200)
	for i := range values {
		values[i] = uint32(rnd.Int31())
	}
	data, flags, err := numcodec.AppendEncodeUint32s(nil, values)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if flags == numcodec.Raw && len(data) != 4*len(values) {
		t.Fatalf("raw size %d != %d", len(data), 4*len(values))
	}
	out, err := numcodec.AppendDecodeUint32s(nil, data, flags)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	for i, v := range values {
		if out[i] != v {
			t.Fatalf("[%d] mismatch (flags=%#x)", i, flags)
		}
	}
}

// TestAppendPreservesPrefix ensures dst content before the encoded payload
// is not touched.
func TestAppendPreservesPrefix(t *testing.T) {
	prefix := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	dst := append([]byte(nil), prefix...)
	dst, _, err := numcodec.AppendEncodeInt64s(dst, []int64{1, 2, 3, 4, 5})
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range prefix {
		if dst[i] != want {
			t.Fatalf("prefix byte %d clobbered: got %#x want %#x", i, dst[i], want)
		}
	}
}
