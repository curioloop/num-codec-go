package chimp_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/curioloop/num-codec-go/chimp"
)

func roundTripFloat64s(t *testing.T, name string, values []float64, n int) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		data := chimp.AppendEncodeFloat64s(nil, values, n)
		out, err := chimp.AppendDecodeFloat64s(nil, data, n)
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

func roundTripFloat32s(t *testing.T, name string, values []float32, n int) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		data := chimp.AppendEncodeFloat32s(nil, values, n)
		out, err := chimp.AppendDecodeFloat32s(nil, data, n)
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

func TestChimpBase(t *testing.T) {
	roundTripFloat64s(t, "identical", []float64{1.5, 1.5, 1.5, 1.5, 1.5}, 0)
	roundTripFloat64s(t, "monotonic", []float64{1.1, 1.2, 1.3, 1.4, 1.5, 1.6}, 0)
	roundTripFloat64s(t, "prices", []float64{100.01, 100.02, 100.02, 100.03, 100.02, 100.05, 100.05, 100.06}, 0)
	roundTripFloat32s(t, "identical32", []float32{2.5, 2.5, 2.5, 2.5}, 0)
	roundTripFloat32s(t, "prices32", []float32{50.01, 50.02, 50.02, 50.03, 50.05}, 0)

	// XOR bit-63 + bit-0 corner case that broke Gorilla — Chimp is immune.
	prev64 := 1.5
	curr64 := math.Float64frombits(math.Float64bits(prev64) ^ 0x8000000000000001)
	roundTripFloat64s(t, "signflip_lsbflip_64", []float64{prev64, curr64}, 0)

	prev32 := float32(1.5)
	curr32 := math.Float32frombits(math.Float32bits(prev32) ^ 0x80000001)
	roundTripFloat32s(t, "signflip_lsbflip_32", []float32{prev32, curr32}, 0)
}

func TestChimpN(t *testing.T) {
	prices := []float64{100.01, 100.02, 100.02, 100.03, 100.02, 100.05, 100.05, 100.06, 100.07, 100.08, 100.05, 100.05, 100.06}
	for _, n := range []int{4, 8, 16, 32, 64, 128, 256} {
		roundTripFloat64s(t, fmt.Sprintf("f64_n=%d", n), prices, n)
	}
	pricesF := []float32{50.01, 50.02, 50.02, 50.03, 50.05, 50.05, 50.06, 50.07, 50.08, 50.05, 50.05, 50.06}
	for _, n := range []int{4, 8, 16, 32, 64, 128, 256} {
		roundTripFloat32s(t, fmt.Sprintf("f32_n=%d", n), pricesF, n)
	}
}

// TestScratchReuse: LSB→index table MUST be zeroed on reuse (stale
// entries corrupt the "recent match" branch).
func TestScratchReuse(t *testing.T) {
	a := []float64{100.01, 100.02, 100.02, 100.03, 100.02, 100.05, 100.06, 100.07, 100.08}
	b := []float64{50.5, 50.6, 50.5, 50.4, 50.3, 50.2, 50.1, 50.0, 49.9, 49.8}

	var s chimp.Scratch
	dataA1 := s.AppendEncodeFloat64s(nil, a, 32)
	dataB := s.AppendEncodeFloat64s(nil, b, 32)
	dataA2 := s.AppendEncodeFloat64s(nil, a, 32) // encode `a` again after `b`

	oneShotA := chimp.AppendEncodeFloat64s(nil, a, 32)
	oneShotB := chimp.AppendEncodeFloat64s(nil, b, 32)

	if !bytesEq(dataA1, oneShotA) {
		t.Fatalf("first encode of a differs from one-shot")
	}
	if !bytesEq(dataB, oneShotB) {
		t.Fatalf("encode of b differs from one-shot")
	}
	if !bytesEq(dataA2, oneShotA) {
		t.Fatalf("second encode of a (after b) differs from one-shot — Scratch reuse leaked state")
	}

	var ds chimp.Scratch
	outA, err := ds.AppendDecodeFloat64s(nil, dataA1, 32)
	if err != nil {
		t.Fatal(err)
	}
	outB, err := ds.AppendDecodeFloat64s(nil, dataB, 32)
	if err != nil {
		t.Fatal(err)
	}
	for i, v := range a {
		if math.Float64bits(outA[i]) != math.Float64bits(v) {
			t.Fatalf("decode a[%d]: got %x want %x", i, math.Float64bits(outA[i]), math.Float64bits(v))
		}
	}
	for i, v := range b {
		if math.Float64bits(outB[i]) != math.Float64bits(v) {
			t.Fatalf("decode b[%d]: got %x want %x", i, math.Float64bits(outB[i]), math.Float64bits(v))
		}
	}
}

func bytesEq(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func chimpBenchSeries() []float64 {
	const n = 10_000
	out := make([]float64, n)
	base := 100.0
	for i := 0; i < n; i++ {
		out[i] = base + float64(i%17)*0.01 + float64(i%7)*0.001
	}
	return out
}

func BenchmarkAppendEncodeFloat64s(b *testing.B) {
	vs := chimpBenchSeries()
	dst := make([]byte, 0, 4096)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dst = chimp.AppendEncodeFloat64s(dst[:0], vs, chimp.DefaultN)
	}
}

func BenchmarkScratchEncodeFloat64s(b *testing.B) {
	vs := chimpBenchSeries()
	var s chimp.Scratch
	dst := make([]byte, 0, 4096)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dst = s.AppendEncodeFloat64s(dst[:0], vs, chimp.DefaultN)
	}
}

func BenchmarkAppendDecodeFloat64s(b *testing.B) {
	vs := chimpBenchSeries()
	data := chimp.AppendEncodeFloat64s(nil, vs, chimp.DefaultN)
	dst := make([]float64, 0, len(vs))
	b.SetBytes(int64(8 * len(vs)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, err := chimp.AppendDecodeFloat64s(dst[:0], data, chimp.DefaultN)
		if err != nil {
			b.Fatal(err)
		}
		dst = out
	}
}

func BenchmarkScratchDecodeFloat64s(b *testing.B) {
	vs := chimpBenchSeries()
	data := chimp.AppendEncodeFloat64s(nil, vs, chimp.DefaultN)
	var s chimp.Scratch
	dst := make([]float64, 0, len(vs))
	b.SetBytes(int64(8 * len(vs)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, err := s.AppendDecodeFloat64s(dst[:0], data, chimp.DefaultN)
		if err != nil {
			b.Fatal(err)
		}
		dst = out
	}
}
