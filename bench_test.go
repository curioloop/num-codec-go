package numcodec_test

import (
	"testing"

	"github.com/curioloop/num-codec-go"
)

// synth returns three canonical workloads used by the benchmarks below.
func synth() (times []int64, prices []float32, sizes []int32) {
	const n = 10_000
	times = make([]int64, n)
	prices = make([]float32, n)
	sizes = make([]int32, n)
	base := int64(1_700_000_000_000)
	for i := 0; i < n; i++ {
		times[i] = base + int64(i)
		prices[i] = 100.0 + float32(i%50)*0.01
		sizes[i] = int32(100 + i%900)
	}
	return
}

func BenchmarkEncodeDelta2(b *testing.B) {
	times, _, _ := synth()
	dst := make([]byte, 0, 4096)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, _, err := numcodec.AppendEncodeDelta2(dst[:0], times)
		if err != nil {
			b.Fatal(err)
		}
		dst = out
	}
}

func BenchmarkDecodeDelta2(b *testing.B) {
	times, _, _ := synth()
	data, flags, err := numcodec.AppendEncodeDelta2(nil, times)
	if err != nil {
		b.Fatal(err)
	}
	dst := make([]int64, 0, len(times))
	b.SetBytes(int64(8 * len(times)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, err := numcodec.AppendDecodeDelta2(dst[:0], data, flags)
		if err != nil {
			b.Fatal(err)
		}
		dst = out
	}
}

func BenchmarkEncodeFloat32s(b *testing.B) {
	_, prices, _ := synth()
	dst := make([]byte, 0, 4096)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, _, err := numcodec.AppendEncodeFloat32s(dst[:0], prices)
		if err != nil {
			b.Fatal(err)
		}
		dst = out
	}
}

func BenchmarkDecodeFloat32s(b *testing.B) {
	_, prices, _ := synth()
	data, flags, err := numcodec.AppendEncodeFloat32s(nil, prices)
	if err != nil {
		b.Fatal(err)
	}
	dst := make([]float32, 0, len(prices))
	b.SetBytes(int64(4 * len(prices)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, err := numcodec.AppendDecodeFloat32s(dst[:0], data, flags)
		if err != nil {
			b.Fatal(err)
		}
		dst = out
	}
}

func BenchmarkEncodeInt32s(b *testing.B) {
	_, _, sizes := synth()
	dst := make([]byte, 0, 4096)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, _, err := numcodec.AppendEncodeInt32s(dst[:0], sizes)
		if err != nil {
			b.Fatal(err)
		}
		dst = out
	}
}

func BenchmarkDecodeInt32s(b *testing.B) {
	_, _, sizes := synth()
	data, flags, err := numcodec.AppendEncodeInt32s(nil, sizes)
	if err != nil {
		b.Fatal(err)
	}
	dst := make([]int32, 0, len(sizes))
	b.SetBytes(int64(4 * len(sizes)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, err := numcodec.AppendDecodeInt32s(dst[:0], data, flags)
		if err != nil {
			b.Fatal(err)
		}
		dst = out
	}
}

// synthUints mirrors synth's `sizes` column but as unsigned to exercise
// the AppendEncodeUint32s / AppendDecodeUint32s pathways.
func synthUints() []uint32 {
	const n = 10_000
	out := make([]uint32, n)
	for i := 0; i < n; i++ {
		out[i] = uint32(100 + i%900)
	}
	return out
}

func BenchmarkEncodeUint32s(b *testing.B) {
	sizes := synthUints()
	dst := make([]byte, 0, 4096)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, _, err := numcodec.AppendEncodeUint32s(dst[:0], sizes)
		if err != nil {
			b.Fatal(err)
		}
		dst = out
	}
}

func BenchmarkDecodeUint32s(b *testing.B) {
	sizes := synthUints()
	data, flags, err := numcodec.AppendEncodeUint32s(nil, sizes)
	if err != nil {
		b.Fatal(err)
	}
	dst := make([]uint32, 0, len(sizes))
	b.SetBytes(int64(4 * len(sizes)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, err := numcodec.AppendDecodeUint32s(dst[:0], data, flags)
		if err != nil {
			b.Fatal(err)
		}
		dst = out
	}
}

// ---------------------------------------------------------------------------
// Encoder-reuse benchmarks: one Encoder amortised across all iterations,
// demonstrating the zero-alloc encode path for repeat callers.
// ---------------------------------------------------------------------------

func BenchmarkEncoderEncodeDelta2(b *testing.B) {
	times, _, _ := synth()
	var enc numcodec.Encoder
	dst := make([]byte, 0, 4096)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, _, err := enc.AppendEncodeDelta2(dst[:0], times)
		if err != nil {
			b.Fatal(err)
		}
		dst = out
	}
}

func BenchmarkEncoderEncodeFloat32s(b *testing.B) {
	_, prices, _ := synth()
	var enc numcodec.Encoder
	dst := make([]byte, 0, 4096)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, _, err := enc.AppendEncodeFloat32s(dst[:0], prices)
		if err != nil {
			b.Fatal(err)
		}
		dst = out
	}
}

func BenchmarkEncoderEncodeInt32s(b *testing.B) {
	_, _, sizes := synth()
	var enc numcodec.Encoder
	dst := make([]byte, 0, 4096)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, _, err := enc.AppendEncodeInt32s(dst[:0], sizes)
		if err != nil {
			b.Fatal(err)
		}
		dst = out
	}
}

func BenchmarkEncoderEncodeUint32s(b *testing.B) {
	sizes := synthUints()
	var enc numcodec.Encoder
	dst := make([]byte, 0, 4096)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, _, err := enc.AppendEncodeUint32s(dst[:0], sizes)
		if err != nil {
			b.Fatal(err)
		}
		dst = out
	}
}

// ---------------------------------------------------------------------------
// 64-bit workloads: int64 (mixed magnitudes), uint64, float64 (Chimp-hitting).
// ---------------------------------------------------------------------------

func synth64() (ints []int64, uints []uint64, floats []float64) {
	const n = 10_000
	ints = make([]int64, n)
	uints = make([]uint64, n)
	floats = make([]float64, n)
	for i := 0; i < n; i++ {
		// signed: alternating polarity to exercise zig-zag
		if i&1 == 0 {
			ints[i] = int64(1_000 + i%997)
		} else {
			ints[i] = -int64(500 + i%503)
		}
		uints[i] = uint64(100_000 + i%9973)
		// float64 series designed to hit Chimp's "matched previous" and
		// "new leading count" branches with a realistic mix.
		floats[i] = 100.0 + float64(i%17)*0.01 + float64(i%7)*0.001
	}
	return
}

func BenchmarkEncodeInt64s(b *testing.B) {
	ints, _, _ := synth64()
	dst := make([]byte, 0, 8192)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, _, err := numcodec.AppendEncodeInt64s(dst[:0], ints)
		if err != nil {
			b.Fatal(err)
		}
		dst = out
	}
}

func BenchmarkDecodeInt64s(b *testing.B) {
	ints, _, _ := synth64()
	data, flags, err := numcodec.AppendEncodeInt64s(nil, ints)
	if err != nil {
		b.Fatal(err)
	}
	dst := make([]int64, 0, len(ints))
	b.SetBytes(int64(8 * len(ints)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, err := numcodec.AppendDecodeInt64s(dst[:0], data, flags)
		if err != nil {
			b.Fatal(err)
		}
		dst = out
	}
}

func BenchmarkEncodeUint64s(b *testing.B) {
	_, uints, _ := synth64()
	dst := make([]byte, 0, 8192)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, _, err := numcodec.AppendEncodeUint64s(dst[:0], uints)
		if err != nil {
			b.Fatal(err)
		}
		dst = out
	}
}

func BenchmarkDecodeUint64s(b *testing.B) {
	_, uints, _ := synth64()
	data, flags, err := numcodec.AppendEncodeUint64s(nil, uints)
	if err != nil {
		b.Fatal(err)
	}
	dst := make([]uint64, 0, len(uints))
	b.SetBytes(int64(8 * len(uints)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, err := numcodec.AppendDecodeUint64s(dst[:0], data, flags)
		if err != nil {
			b.Fatal(err)
		}
		dst = out
	}
}

func BenchmarkEncodeFloat64s(b *testing.B) {
	_, _, floats := synth64()
	dst := make([]byte, 0, 8192)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, _, err := numcodec.AppendEncodeFloat64s(dst[:0], floats)
		if err != nil {
			b.Fatal(err)
		}
		dst = out
	}
}

func BenchmarkDecodeFloat64s(b *testing.B) {
	_, _, floats := synth64()
	data, flags, err := numcodec.AppendEncodeFloat64s(nil, floats)
	if err != nil {
		b.Fatal(err)
	}
	dst := make([]float64, 0, len(floats))
	b.SetBytes(int64(8 * len(floats)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, err := numcodec.AppendDecodeFloat64s(dst[:0], data, flags)
		if err != nil {
			b.Fatal(err)
		}
		dst = out
	}
}

// Encoder-reuse variants for the 64-bit paths.

func BenchmarkEncoderEncodeInt64s(b *testing.B) {
	ints, _, _ := synth64()
	var enc numcodec.Encoder
	dst := make([]byte, 0, 8192)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, _, err := enc.AppendEncodeInt64s(dst[:0], ints)
		if err != nil {
			b.Fatal(err)
		}
		dst = out
	}
}

func BenchmarkEncoderEncodeFloat64s(b *testing.B) {
	_, _, floats := synth64()
	var enc numcodec.Encoder
	dst := make([]byte, 0, 8192)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, _, err := enc.AppendEncodeFloat64s(dst[:0], floats)
		if err != nil {
			b.Fatal(err)
		}
		dst = out
	}
}

// Delta2 on an unordered int64 series: forces the unordered path
// (zig-zag over the second-order differences) rather than the varint
// path used for monotonic timestamps.
func synthUnordered() []int64 {
	const n = 10_000
	out := make([]int64, n)
	x := int64(1_000_000)
	for i := 0; i < n; i++ {
		// bounded random-ish walk without RNG cost.
		x += int64(((i*2654435761)>>16)%201) - 100
		out[i] = x
	}
	return out
}

func BenchmarkEncodeDelta2Unordered(b *testing.B) {
	xs := synthUnordered()
	dst := make([]byte, 0, 8192)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, _, err := numcodec.AppendEncodeDelta2(dst[:0], xs)
		if err != nil {
			b.Fatal(err)
		}
		dst = out
	}
}

func BenchmarkDecodeDelta2Unordered(b *testing.B) {
	xs := synthUnordered()
	data, flags, err := numcodec.AppendEncodeDelta2(nil, xs)
	if err != nil {
		b.Fatal(err)
	}
	dst := make([]int64, 0, len(xs))
	b.SetBytes(int64(8 * len(xs)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, err := numcodec.AppendDecodeDelta2(dst[:0], data, flags)
		if err != nil {
			b.Fatal(err)
		}
		dst = out
	}
}
