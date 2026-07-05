package delta2_test

import (
	"errors"
	"testing"

	"github.com/curioloop/num-codec-go/delta2"
	"github.com/curioloop/num-codec-go/internal/codecerr"
)

func roundTripOrdered(t *testing.T, name string, values []int64) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		data, err := delta2.AppendEncodeOrdered(nil, values)
		if err != nil {
			t.Fatalf("encode: %v", err)
		}
		out, err := delta2.AppendDecodeOrdered(nil, data)
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
	})
}

func roundTripUnordered(t *testing.T, name string, values []int64) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		data := delta2.AppendEncodeUnordered(nil, values)
		out, err := delta2.AppendDecodeUnordered(nil, data)
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
	})
}

func TestDelta2OrderedRoundTrip(t *testing.T) {
	roundTripOrdered(t, "monotonic", []int64{100, 101, 103, 106, 110, 115, 121})
	ts := make([]int64, 100)
	base := int64(1_700_000_000)
	for i := range ts {
		ts[i] = base + int64(i)
	}
	roundTripOrdered(t, "seconds", ts)
	roundTripOrdered(t, "single", []int64{42})
}

func TestDelta2UnorderedRoundTrip(t *testing.T) {
	roundTripUnordered(t, "mixed", []int64{5, 3, 8, 2, 100, -5, 0})
	roundTripUnordered(t, "single", []int64{42})
	roundTripUnordered(t, "empty", []int64{})
}

func TestDelta2OrderedNegativeOverflow(t *testing.T) {
	_, err := delta2.AppendEncodeOrdered(nil, []int64{100, 90, 110})
	if !errors.Is(err, codecerr.ErrOverflow) {
		t.Fatalf("want ErrOverflow got %v", err)
	}
}
