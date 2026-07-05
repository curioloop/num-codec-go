// Package chimp implements Chimp / ChimpN floating-point compression as
// described in Liakos et al. VLDB'22. Bit layout, table values, and byte
// framing match the Java number-codec library exactly.
package chimp

// Shared constants and precomputed tables. Values MUST match the Java
// library (chimp/Chimp.java static block) verbatim for wire compatibility.

const (
	ctrlFlagBits     = 2
	leadingCountBits = 3
	leadingCountMask = (1 << leadingCountBits) - 1

	doubleCenterBits = 6
	doubleCenterMask = ^(^0 << doubleCenterBits) // 0x3F

	floatCenterBits = 5
	floatCenterMask = ^(^0 << floatCenterBits) // 0x1F

	maxLog2_64 = 6 // log2(64) — threshold for trailing-zero count in doubles
	maxLog2_32 = 5 // log2(32) — threshold for trailing-zero count in floats
)

// leadingRound quantises the actual number of leading zeros of an xor to
// one of the 8 representable buckets (0, 8, 12, 16, 18, 20, 22, 24).
// Indexed by numberOfLeadingZeros(xor).
var leadingRound = [64]uint8{
	0, 0, 0, 0, 0, 0, 0, 0,
	8, 8, 8, 8, 12, 12, 12, 12,
	16, 16, 18, 18, 20, 20, 22, 22,
	24, 24, 24, 24, 24, 24, 24, 24,
	24, 24, 24, 24, 24, 24, 24, 24,
	24, 24, 24, 24, 24, 24, 24, 24,
	24, 24, 24, 24, 24, 24, 24, 24,
	24, 24, 24, 24, 24, 24, 24, 24,
}

// leadingEncode maps a rounded leading-zero count to its 3-bit code (0..7).
// Indexed by the rounded leading-zero value (so index 0..24 is used).
var leadingEncode = [64]uint8{
	0, 0, 0, 0, 0, 0, 0, 0,
	1, 1, 1, 1, 2, 2, 2, 2,
	3, 3, 4, 4, 5, 5, 6, 6,
	7, 7, 7, 7, 7, 7, 7, 7,
	7, 7, 7, 7, 7, 7, 7, 7,
	7, 7, 7, 7, 7, 7, 7, 7,
	7, 7, 7, 7, 7, 7, 7, 7,
	7, 7, 7, 7, 7, 7, 7, 7,
}

// leadingDecode is the inverse: 3-bit code → leading-zero count.
var leadingDecode = [8]uint8{0, 8, 12, 16, 18, 20, 22, 24}
