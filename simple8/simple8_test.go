package simple8_test

import (
	"errors"
	"testing"

	"github.com/curioloop/num-codec-go/internal/codecerr"
	"github.com/curioloop/num-codec-go/simple8"
)

func roundTrip(t *testing.T, name string, values []uint64) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		data, err := simple8.AppendPack(nil, values)
		if err != nil {
			t.Fatalf("pack: %v", err)
		}
		out, err := simple8.AppendUnpack(nil, data)
		if err != nil {
			t.Fatalf("unpack: %v", err)
		}
		if len(out) < len(values) {
			t.Fatalf("count: got %d want %d", len(out), len(values))
		}
		for i, v := range values {
			if out[i] != v {
				t.Fatalf("[%d] got %d want %d", i, out[i], v)
			}
		}
	})
}

func TestSimple8AllSelectors(t *testing.T) {
	roundTrip(t, "single_60bit", []uint64{(1 << 60) - 1})
	roundTrip(t, "two_30bit", []uint64{(1 << 30) - 1, (1 << 30) - 2})
	roundTrip(t, "three_20bit", []uint64{(1 << 20) - 1, 1, 2})

	sixty := make([]uint64, 60)
	for i := range sixty {
		sixty[i] = uint64(i & 1)
	}
	roundTrip(t, "sixty_1bit", sixty)

	oneTwenty := make([]uint64, 120)
	roundTrip(t, "one_twenty_zero", oneTwenty)
	for i := range oneTwenty {
		oneTwenty[i] = 1
	}
	roundTrip(t, "one_twenty_ones", oneTwenty)

	twoForty := make([]uint64, 240)
	for i := range twoForty {
		twoForty[i] = 1
	}
	roundTrip(t, "two_forty_ones", twoForty)
}

func TestSimple8Mixed(t *testing.T) {
	values := make([]uint64, 0)
	for i := 0; i < 100; i++ {
		values = append(values, uint64(i))
	}
	for i := 0; i < 30; i++ {
		values = append(values, uint64(1<<20)+uint64(i))
	}
	roundTrip(t, "mixed", values)
}

// TestSimple8OverflowReturnsError verifies values ≥ 2^60 raise ErrOverflow.
func TestSimple8OverflowReturnsError(t *testing.T) {
	_, err := simple8.AppendPack(nil, []uint64{1 << 60})
	if !errors.Is(err, codecerr.ErrOverflow) {
		t.Fatalf("want ErrOverflow got %v", err)
	}
}

// TestSimple8MalformedInput verifies short data is rejected cleanly.
func TestSimple8MalformedInput(t *testing.T) {
	_, err := simple8.AppendUnpack(nil, []byte{1, 2, 3})
	if !errors.Is(err, codecerr.ErrMalformed) {
		t.Fatalf("want ErrMalformed got %v", err)
	}
}
