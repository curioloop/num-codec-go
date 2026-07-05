package simple8_test

import (
	"testing"

	"github.com/curioloop/num-codec-go/simple8"
)

func benchData(tb testing.TB) []byte {
	tb.Helper()
	vs := make([]uint64, 10_000)
	for i := range vs {
		vs[i] = uint64(i & 0x3FF) // fits in 10 bits — dense packing
	}
	data, err := simple8.AppendPack(nil, vs)
	if err != nil {
		tb.Fatal(err)
	}
	return data
}

// Baseline: append into a caller slice.
func BenchmarkUnpackAppend(b *testing.B) {
	data := benchData(b)
	dst := make([]uint64, 0, 10_240)
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, err := simple8.AppendUnpack(dst[:0], data)
		if err != nil {
			b.Fatal(err)
		}
		dst = out
	}
}

// Callback form — should be zero-alloc (closure stays on stack).
func BenchmarkUnpackFunc(b *testing.B) {
	data := benchData(b)
	var sum uint64
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sum = 0
		if err := simple8.UnpackFunc(data, func(v uint64) error {
			sum += v
			return nil
		}); err != nil {
			b.Fatal(err)
		}
	}
	_ = sum
}
