// Package gorilla implements Gorilla-style XOR compression for time-series
// float64/float32 data. Bit format matches the Java number-codec library
// exactly.
package gorilla

import (
	"math"
	"math/bits"

	"github.com/curioloop/num-codec-go/internal/bitpack"
	"github.com/curioloop/num-codec-go/internal/codecerr"
)

const (
	maxLeadingZeroBits = 6                                // encoded field width for leading-zero count
	maxBlockSizeBits   = 7                                // encoded field width for significant-bit count
	blockSizeMask      = int32(^(^0 << maxBlockSizeBits)) // 0x7F
)

// encodeBlock encodes one delta: writes a single 0 bit for "value unchanged",
// otherwise emits the meta (if different from previous) plus the value's
// significant bits.
//
// meta layout (13 bits): [6-bit leading count | 7-bit block size].
// meta is passed as int32 so negative diffSize truncates consistently via
// the low-13-bits mask.
func encodeBlock(w *bitpack.Writer, prev, meta int32, value uint64) {
	if value == 0 {
		w.WriteBit(false)
		return
	}
	w.WriteBit(true)
	ctrlBit := meta != prev
	w.WriteBit(ctrlBit)
	if ctrlBit {
		w.WriteBits(uint64(uint32(meta)), maxLeadingZeroBits+maxBlockSizeBits)
	}
	numBits := int(meta & blockSizeMask)
	w.WriteBits(value, numBits)
}

// AppendEncodeFloat64s appends a Gorilla-encoded stream of vs to dst.
func AppendEncodeFloat64s(dst []byte, vs []float64) []byte {
	if len(vs) == 0 {
		return dst
	}
	var w bitpack.Writer
	w.Reset(dst)
	var prevBlock int32
	previous := math.Float64bits(vs[0])
	w.WriteBits(previous, 64)
	for n := 1; n < len(vs); n++ {
		current := math.Float64bits(vs[n])
		xor := previous ^ current
		leading := bits.LeadingZeros64(xor)
		trailing := bits.TrailingZeros64(xor)
		diffBits := xor >> uint(trailing&63) // trailing may equal 64 when xor==0
		// sigBits is always non-negative (0..64). Historically this was
		// multiplied by signum(int64(diffBits)), which could produce a
		// negative int32 that overflowed the 7-bit blockSize field via
		// sign extension, silently corrupting the leading-zero field on
		// sign-flip + LSB-flip corner cases. Fixed to match the Java lib.
		// Guard xor==0 (where leading=trailing=64 would give sigBits=-64):
		// encodeBlock short-circuits that case anyway, but prevBlock still
		// participates in later meta-change detection so keep it clean.
		var sigBits int
		if xor != 0 {
			sigBits = 64 - leading - trailing
		}
		currBlock := (int32(leading) << maxBlockSizeBits) | int32(sigBits)
		encodeBlock(&w, prevBlock, currBlock, diffBits)
		prevBlock = currBlock
		previous = current
	}
	w.Flush()
	return w.Bytes()
}

// AppendDecodeFloat64s decodes a Gorilla-encoded stream from data and
// appends the values to dst.
func AppendDecodeFloat64s(dst []float64, data []byte) ([]float64, error) {
	if len(data) < 2 {
		return dst, codecerr.ErrMalformed
	}
	var r bitpack.Reader
	r.Reset(data)
	var trailing, blockSize int
	value := r.ReadBits(64)
	dst = append(dst, math.Float64frombits(value))
	for r.HasMore() {
		var b uint64
		if r.ReadBit() {
			if r.ReadBit() {
				meta := int(r.ReadBits(maxLeadingZeroBits + maxBlockSizeBits))
				blockSize = meta & int(blockSizeMask)
				trailing = 64 - blockSize - (meta >> maxBlockSizeBits)
			}
			if (blockSize | trailing) == 0 {
				return dst, codecerr.ErrMalformed
			}
			// Java: `bits = readBits(blockSize) << trailing;` where trailing
			// may be negative in the pathological diffBits<0 branch. Java's
			// long shift masks the count with 63, so mirror that.
			b = r.ReadBits(blockSize) << uint(trailing&63)
		}
		value ^= b
		dst = append(dst, math.Float64frombits(value))
	}
	if err := r.Err(); err != nil {
		return dst, err
	}
	return dst, nil
}

// AppendEncodeFloat32s appends a Gorilla-encoded stream of vs to dst.
func AppendEncodeFloat32s(dst []byte, vs []float32) []byte {
	if len(vs) == 0 {
		return dst
	}
	var w bitpack.Writer
	w.Reset(dst)
	var prevBlock int32
	previous := math.Float32bits(vs[0])
	w.WriteBits(uint64(previous), 32)
	for n := 1; n < len(vs); n++ {
		current := math.Float32bits(vs[n])
		xor := previous ^ current
		leading := bits.LeadingZeros32(xor)
		trailing := bits.TrailingZeros32(xor)
		diffBits := xor >> uint(trailing&31)
		// See AppendEncodeFloat64s for why the historic signum trick was
		// dropped and why xor==0 needs an explicit guard.
		var sigBits int
		if xor != 0 {
			sigBits = 32 - leading - trailing
		}
		currBlock := (int32(leading) << maxBlockSizeBits) | int32(sigBits)
		encodeBlock(&w, prevBlock, currBlock, uint64(diffBits))
		prevBlock = currBlock
		previous = current
	}
	w.Flush()
	return w.Bytes()
}

// AppendDecodeFloat32s decodes a Gorilla-encoded stream from data and
// appends the values to dst.
func AppendDecodeFloat32s(dst []float32, data []byte) ([]float32, error) {
	if len(data) < 2 {
		return dst, codecerr.ErrMalformed
	}
	var r bitpack.Reader
	r.Reset(data)
	var trailing, blockSize int
	value := uint32(r.ReadBits(32))
	dst = append(dst, math.Float32frombits(value))
	for r.HasMore() {
		var b uint32
		if r.ReadBit() {
			if r.ReadBit() {
				meta := int(r.ReadBits(maxLeadingZeroBits + maxBlockSizeBits))
				blockSize = meta & int(blockSizeMask)
				trailing = 32 - blockSize - (meta >> maxBlockSizeBits)
			}
			if (blockSize | trailing) == 0 {
				return dst, codecerr.ErrMalformed
			}
			b = uint32(r.ReadBits(blockSize)) << uint(trailing&31)
		}
		value ^= b
		dst = append(dst, math.Float32frombits(value))
	}
	if err := r.Err(); err != nil {
		return dst, err
	}
	return dst, nil
}
