package numcodec_test

import (
	"errors"
	"fmt"

	"github.com/curioloop/num-codec-go"
)

func ExampleAppendEncodeInt32s() {
	sizes := []int32{100, 100, 105, 110, 112, 108, 100}
	data, flags, _ := numcodec.AppendEncodeInt32s(nil, sizes)
	fmt.Printf("bytes=%d flags=%s\n", len(data), flags)

	out, _ := numcodec.AppendDecodeInt32s(nil, data, flags)
	fmt.Println(out)
	// Output:
	// bytes=16 flags=ZigZag|Simple8
	// [100 100 105 110 112 108 100]
}

func ExampleAppendEncodeFloat64s() {
	prices := []float64{100.01, 100.02, 100.02, 100.03, 100.05, 100.06}
	data, flags, _ := numcodec.AppendEncodeFloat64s(nil, prices)
	fmt.Printf("bytes=%d flags=%s\n", len(data), flags)

	out, _ := numcodec.AppendDecodeFloat64s(nil, data, flags)
	fmt.Println(out)
	// Output:
	// bytes=37 flags=Gorilla
	// [100.01 100.02 100.02 100.03 100.05 100.06]
}

func ExampleAppendEncodeDelta2() {
	// Timestamps at 1-second intervals compress via Delta2+Simple8b.
	base := int64(1_700_000_000)
	ts := make([]int64, 100)
	for i := range ts {
		ts[i] = base + int64(i)
	}
	data, flags, _ := numcodec.AppendEncodeDelta2(nil, ts)
	fmt.Printf("bytes=%d flags=%s\n", len(data), flags)
	// Output:
	// bytes=40 flags=Simple8|Delta2
}

// ExampleEncoder shows the Encoder-based scratch reuse pattern for
// heavy workloads.
func ExampleEncoder() {
	var enc numcodec.Encoder
	dst := make([]byte, 0, 4096)

	// Encoding many series in a loop: internal scratches are amortised
	// across all iterations for zero allocations per call.
	series := [][]float64{
		{100.01, 100.02, 100.02, 100.03, 100.05, 100.06, 100.05, 100.07},
		{50.5, 50.5, 50.6, 50.7, 50.5, 50.6, 50.7, 50.8},
	}
	for _, s := range series {
		data, flags, _ := enc.AppendEncodeFloat64s(dst[:0], s)
		fmt.Printf("len=%d flags=%s\n", len(data), flags)
	}
	// Output:
	// len=49 flags=Gorilla
	// len=54 flags=Gorilla
}

// ExampleFlags shows how to interrogate the returned Flags bitmask.
func ExampleFlags() {
	_, flags, _ := numcodec.AppendEncodeInt32s(nil, []int32{1, 2, 3, 4, 5})

	// Human-readable summary via Flags.String().
	fmt.Println(flags)

	// Programmatic dispatch on a specific codec bit.
	switch {
	case flags == numcodec.Raw:
		fmt.Println("raw big-endian storage")
	case flags&numcodec.Simple8 != 0:
		fmt.Println("Simple8b packing was chosen")
	default:
		fmt.Println("some other codec")
	}
	// Output:
	// ZigZag|Simple8
	// Simple8b packing was chosen
}

// ExampleAppendDecodeInt32s_invalidFlags shows the ErrBadFlags path.
func ExampleAppendDecodeInt32s_invalidFlags() {
	// Passing a Flags value with no active codec bit is a programmer
	// error, surfaced as ErrBadFlags (compare via errors.Is).
	_, err := numcodec.AppendDecodeInt32s(nil, []byte{0x01, 0x02}, 0)
	fmt.Println(errors.Is(err, numcodec.ErrBadFlags))
	// Output:
	// true
}
