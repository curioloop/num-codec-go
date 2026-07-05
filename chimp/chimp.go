// Package chimp implements the Chimp lossless floating-point compression
// algorithm and its ring-buffer variant ChimpN (Liakos et al. VLDB'22).
// Bit layout, table values, and byte framing match the Java number-codec
// library.
//
// # Choosing N
//
// N == 0 selects base Chimp (no ring buffer, single-previous XOR). Any
// power of two in [4, 256] selects ChimpN with that ring size — larger
// N usually compresses better on slowly-varying data at the cost of a
// per-encode ~ (N + 2^(6+log2 N)) * word-size scratch allocation. Use
// [Scratch] to amortise that across many calls.
//
// # Wire self-description
//
// This package encodes/decodes with the value of N known out-of-band.
// If you need the encoded bytes to be self-describing (a header byte
// carrying log2 N so the decoder can pick the right variant without
// prior knowledge), use the top-level numcodec package which routes
// through an internal Codec dispatcher.
package chimp

import (
	"math"
	"math/bits"

	"github.com/curioloop/num-codec-go/internal/bitpack"
	"github.com/curioloop/num-codec-go/internal/codecerr"
)

// DefaultN is a reasonable default ring size (32) for AppendEncode*Float*s
// when the caller doesn't have a preference.
const DefaultN = 32

// AppendEncodeFloat64s appends Chimp-encoded output to dst. n selects the
// variant: 0 → base Chimp; a power of two in [4, 256] → ChimpN.
// Allocates internal scratch on every call; use [Scratch] for reuse.
func AppendEncodeFloat64s(dst []byte, vs []float64, n int) []byte {
	if n == 0 {
		return appendEncodeBaseFloat64s(dst, vs)
	}
	var s Scratch
	return s.AppendEncodeFloat64s(dst, vs, n)
}

// AppendDecodeFloat64s decodes Chimp-encoded data into dst. n must match
// the value used at encode time (0 for base Chimp).
func AppendDecodeFloat64s(dst []float64, data []byte, n int) ([]float64, error) {
	if n == 0 {
		return appendDecodeBaseFloat64s(dst, data)
	}
	var s Scratch
	return s.AppendDecodeFloat64s(dst, data, n)
}

// AppendEncodeFloat32s is the float32 counterpart of AppendEncodeFloat64s.
func AppendEncodeFloat32s(dst []byte, vs []float32, n int) []byte {
	if n == 0 {
		return appendEncodeBaseFloat32s(dst, vs)
	}
	var s Scratch
	return s.AppendEncodeFloat32s(dst, vs, n)
}

// AppendDecodeFloat32s is the float32 counterpart of AppendDecodeFloat64s.
func AppendDecodeFloat32s(dst []float32, data []byte, n int) ([]float32, error) {
	if n == 0 {
		return appendDecodeBaseFloat32s(dst, data)
	}
	var s Scratch
	return s.AppendDecodeFloat32s(dst, data, n)
}

// assertN panics on an invalid ChimpN ring size. n == 0 is caller error
// here; the public entry points route to the base variant before calling
// into ChimpN.
func assertN(n int) {
	if n < 4 || n > 256 || bits.OnesCount(uint(n)) != 1 {
		panic("chimp: n must be a power of two in [4, 256]")
	}
}

// -----------------------------------------------------------------------
// Base Chimp — no ring buffer, XOR against immediately previous value.
// -----------------------------------------------------------------------

func appendEncodeBaseFloat64s(dst []byte, vs []float64) []byte {
	if len(vs) == 0 {
		return dst
	}
	var w bitpack.Writer
	w.Reset(dst)
	prevLeading := 0
	previous := math.Float64bits(vs[0])
	w.WriteBits(previous, 64)
	for n := 1; n < len(vs); n++ {
		current := math.Float64bits(vs[n])
		xor := previous ^ current
		if xor == 0 {
			w.WriteBits(0, ctrlFlagBits) // 0b00
			prevLeading = 64 + 1
		} else {
			leading := int(leadingRound[bits.LeadingZeros64(xor)])
			trailing := bits.TrailingZeros64(xor)
			if trailing > maxLog2_64 {
				sigBits := 64 - leading - trailing
				meta := (uint64(0b01) << (leadingCountBits + doubleCenterBits)) |
					(uint64(leadingEncode[leading]) << doubleCenterBits) |
					uint64(sigBits)
				w.WriteBits(meta, ctrlFlagBits+leadingCountBits+doubleCenterBits)
				w.WriteBits(xor>>uint(trailing), sigBits)
				prevLeading = 64 + 1
			} else if leading == prevLeading {
				w.WriteBits(0b10, ctrlFlagBits)
				w.WriteBits(xor, 64-leading)
			} else {
				prevLeading = leading
				meta := (uint64(0b11) << leadingCountBits) | uint64(leadingEncode[leading])
				w.WriteBits(meta, ctrlFlagBits+leadingCountBits)
				w.WriteBits(xor, 64-leading)
			}
		}
		previous = current
	}
	w.Flush()
	return w.Bytes()
}

func appendDecodeBaseFloat64s(dst []float64, data []byte) ([]float64, error) {
	if len(data) < 2 {
		return dst, codecerr.ErrMalformed
	}
	var r bitpack.Reader
	r.Reset(data)
	prevLeading := 0
	value := r.ReadBits(64)
	dst = append(dst, math.Float64frombits(value))
	for r.HasMore() {
		var b uint64
		switch r.ReadBits(ctrlFlagBits) {
		case 0b11:
			prevLeading = int(leadingDecode[r.ReadBits(leadingCountBits)])
			fallthrough
		case 0b10:
			b = r.ReadBits(64 - prevLeading)
		case 0b01:
			meta := int(r.ReadBits(leadingCountBits + doubleCenterBits))
			prevLeading = int(leadingDecode[meta>>doubleCenterBits])
			sigBits := meta & doubleCenterMask
			trailing := 64 - sigBits - prevLeading
			b = r.ReadBits(sigBits) << uint(trailing)
		}
		value ^= b
		dst = append(dst, math.Float64frombits(value))
	}
	if err := r.Err(); err != nil {
		return dst, err
	}
	return dst, nil
}

func appendEncodeBaseFloat32s(dst []byte, vs []float32) []byte {
	if len(vs) == 0 {
		return dst
	}
	var w bitpack.Writer
	w.Reset(dst)
	prevLeading := 0
	previous := math.Float32bits(vs[0])
	w.WriteBits(uint64(previous), 32)
	for n := 1; n < len(vs); n++ {
		current := math.Float32bits(vs[n])
		xor := previous ^ current
		if xor == 0 {
			w.WriteBits(0, ctrlFlagBits)
			prevLeading = 32 + 1
		} else {
			leading := int(leadingRound[bits.LeadingZeros32(xor)])
			trailing := bits.TrailingZeros32(xor)
			if trailing > maxLog2_32 {
				sigBits := 32 - leading - trailing
				sigMask := (uint64(1) << uint(sigBits)) - 1
				ctrl := (uint64(0b01) << (leadingCountBits + floatCenterBits)) |
					(uint64(leadingEncode[leading]) << floatCenterBits) |
					uint64(sigBits)
				w.WriteBits((ctrl<<uint(sigBits))|((uint64(xor)>>uint(trailing))&sigMask),
					ctrlFlagBits+leadingCountBits+floatCenterBits+sigBits)
				prevLeading = 32 + 1
			} else if leading == prevLeading {
				sigBits := 32 - leading
				sigMask := (uint64(1) << uint(sigBits)) - 1
				w.WriteBits((uint64(0b10)<<uint(sigBits))|(uint64(xor)&sigMask),
					ctrlFlagBits+sigBits)
			} else {
				prevLeading = leading
				sigBits := 32 - leading
				sigMask := (uint64(1) << uint(sigBits)) - 1
				ctrl := (uint64(0b11) << leadingCountBits) | uint64(leadingEncode[leading])
				w.WriteBits((ctrl<<uint(sigBits))|(uint64(xor)&sigMask),
					ctrlFlagBits+leadingCountBits+sigBits)
			}
		}
		previous = current
	}
	w.Flush()
	return w.Bytes()
}

func appendDecodeBaseFloat32s(dst []float32, data []byte) ([]float32, error) {
	if len(data) < 2 {
		return dst, codecerr.ErrMalformed
	}
	var r bitpack.Reader
	r.Reset(data)
	prevLeading := 0
	value := uint32(r.ReadBits(32))
	dst = append(dst, math.Float32frombits(value))
	for r.HasMore() {
		var b uint32
		switch r.ReadBits(ctrlFlagBits) {
		case 0b11:
			prevLeading = int(leadingDecode[r.ReadBits(leadingCountBits)])
			fallthrough
		case 0b10:
			b = uint32(r.ReadBits(32 - prevLeading))
		case 0b01:
			meta := int(r.ReadBits(leadingCountBits + floatCenterBits))
			prevLeading = int(leadingDecode[meta>>floatCenterBits])
			sigBits := meta & floatCenterMask
			trailing := 32 - sigBits - prevLeading
			b = uint32(r.ReadBits(sigBits)) << uint(trailing)
		}
		value ^= b
		dst = append(dst, math.Float32frombits(value))
	}
	if err := r.Err(); err != nil {
		return dst, err
	}
	return dst, nil
}
