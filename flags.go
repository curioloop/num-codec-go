package numcodec

import (
	"fmt"
	"strings"
)

// Flags is the codec-selection bitmask stored alongside encoded data.
// Values match the Java number-codec library exactly to preserve wire
// compatibility.
//
// Every Append*Encode* function returns one or a bitwise OR of two Flags;
// pass the same value back to the matching Append*Decode*.
type Flags uint8

const (
	Raw     Flags = 1 << 0 // 0x01 raw big-endian primitives, no compression
	Gorilla Flags = 1 << 1 // 0x02 Gorilla XOR compression (float / double)
	Uvarint Flags = 1 << 2 // 0x04 unsigned variable-length integer
	ZigZag  Flags = 1 << 3 // 0x08 zig-zag transform (signed variable-length)
	Simple8 Flags = 1 << 4 // 0x10 Simple8b bit-packing
	Delta2  Flags = 1 << 5 // 0x20 delta-of-delta encoding
	Chimp   Flags = 1 << 6 // 0x40 Chimp compression (float / double)
)

// String returns a "|"-joined list of flag names, e.g. "Delta2|Simple8".
// Bits outside the known set are emitted as their hex value, e.g.
// "Gorilla|0x80". A zero Flags returns "None".
func (f Flags) String() string {
	if f == 0 {
		return "None"
	}
	var parts []string
	names := [...]struct {
		bit  Flags
		name string
	}{
		{Raw, "Raw"},
		{Gorilla, "Gorilla"},
		{Uvarint, "Uvarint"},
		{ZigZag, "ZigZag"},
		{Simple8, "Simple8"},
		{Delta2, "Delta2"},
		{Chimp, "Chimp"},
	}
	rest := f
	for _, e := range names {
		if f&e.bit != 0 {
			parts = append(parts, e.name)
			rest &^= e.bit
		}
	}
	if rest != 0 {
		parts = append(parts, fmt.Sprintf("%#x", uint8(rest)))
	}
	return strings.Join(parts, "|")
}
