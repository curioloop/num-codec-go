package chimp

import (
	"math"
	"math/bits"

	"github.com/curioloop/num-codec-go/internal/bitpack"
	"github.com/curioloop/num-codec-go/internal/codecerr"
)

// Scratch holds the ring buffer and LSB→index lookup table reused by
// ChimpN encode/decode. A zero-value Scratch is ready to use. Buffers
// grow monotonically to fit the largest N and payload seen.
//
// Not safe for concurrent use by multiple goroutines. For parallel
// workloads use one Scratch per goroutine, or wrap in a sync.Pool.
//
// Reuse gains only apply to ChimpN (n != 0). The base Chimp variant has
// no large internal state to amortise; Scratch methods with n == 0
// simply forward to the base implementation.
type Scratch struct {
	prev64 []uint64 // ring buffer for double encode/decode
	prev32 []uint32 // ring buffer for float encode/decode
	idx    []int    // LSB→index lookup; encode-only
}

// AppendEncodeFloat64s encodes vs using this Scratch's ring buffer.
// n == 0 forwards to the base Chimp implementation (no scratch used);
// otherwise n must be a power of two in [4, 256].
func (s *Scratch) AppendEncodeFloat64s(dst []byte, vs []float64, n int) []byte {
	if n == 0 {
		return appendEncodeBaseFloat64s(dst, vs)
	}
	return s.appendEncodeChimpNFloat64s(dst, vs, n)
}

// AppendDecodeFloat64s decodes vs using this Scratch's ring buffer.
func (s *Scratch) AppendDecodeFloat64s(dst []float64, data []byte, n int) ([]float64, error) {
	if n == 0 {
		return appendDecodeBaseFloat64s(dst, data)
	}
	return s.appendDecodeChimpNFloat64s(dst, data, n)
}

// AppendEncodeFloat32s is the float32 counterpart.
func (s *Scratch) AppendEncodeFloat32s(dst []byte, vs []float32, n int) []byte {
	if n == 0 {
		return appendEncodeBaseFloat32s(dst, vs)
	}
	return s.appendEncodeChimpNFloat32s(dst, vs, n)
}

// AppendDecodeFloat32s is the float32 counterpart.
func (s *Scratch) AppendDecodeFloat32s(dst []float32, data []byte, n int) ([]float32, error) {
	if n == 0 {
		return appendDecodeBaseFloat32s(dst, data)
	}
	return s.appendDecodeChimpNFloat32s(dst, data, n)
}

// -----------------------------------------------------------------------
// ChimpN implementations reused by both the package-level AppendEncode*
// (via a throw-away Scratch) and the Scratch methods above.
// -----------------------------------------------------------------------

func growUint64Slice(s []uint64, n int) []uint64 {
	if cap(s) < n {
		return make([]uint64, n)
	}
	return s[:n]
}

func growUint32Slice(s []uint32, n int) []uint32 {
	if cap(s) < n {
		return make([]uint32, n)
	}
	return s[:n]
}

func growIntSlice(s []int, n int) []int {
	if cap(s) < n {
		return make([]int, n)
	}
	return s[:n]
}

func zeroInts(s []int) {
	for i := range s {
		s[i] = 0
	}
}

func (s *Scratch) appendEncodeChimpNFloat64s(dst []byte, vs []float64, n int) []byte {
	assertN(n)
	if len(vs) == 0 {
		return dst
	}
	var w bitpack.Writer
	w.Reset(dst)

	log2N := 31 - bits.LeadingZeros32(uint32(n))
	threshold := maxLog2_64 + log2N
	nMask := n - 1

	s.prev64 = growUint64Slice(s.prev64, n)
	previous := s.prev64

	// LSB→index lookup MUST start clean: a stale entry pointing to a
	// large prior-call index would incorrectly satisfy the
	// (index - currIndex) < n check and read a garbage ring slot.
	indicesSize := 1 << uint(threshold+1)
	maskLSB := indicesSize - 1
	s.idx = growIntSlice(s.idx, indicesSize)
	indices := s.idx
	zeroInts(indices)

	index := 0
	current := 0
	prevLeading := 0

	previous[current] = math.Float64bits(vs[0])
	w.WriteBits(previous[current], 64)
	indices[int(previous[current])&maskLSB] = index

	for step := 1; step < len(vs); step++ {
		value := math.Float64bits(vs[step])
		key := int(value) & maskLSB

		var xor uint64
		var trailing int
		var prevIndex int
		currIndex := indices[key]
		if (index - currIndex) < n {
			tempXor := value ^ previous[currIndex&nMask]
			trailing = bits.TrailingZeros64(tempXor)
			if trailing > threshold {
				prevIndex = currIndex & nMask
				xor = tempXor
			} else {
				prevIndex = index & nMask
				xor = previous[prevIndex] ^ value
			}
		} else {
			prevIndex = index & nMask
			xor = previous[prevIndex] ^ value
		}

		if xor == 0 {
			w.WriteBits(uint64(prevIndex), ctrlFlagBits+log2N)
			prevLeading = 64 + 1
		} else {
			leading := int(leadingRound[bits.LeadingZeros64(xor)])
			if trailing > threshold {
				sigBits := 64 - leading - trailing
				var meta uint64 = 0b01
				meta = (meta << uint(log2N)) | uint64(prevIndex)
				meta = (meta << leadingCountBits) | uint64(leadingEncode[leading])
				meta = (meta << doubleCenterBits) | uint64(sigBits)
				w.WriteBits(meta, ctrlFlagBits+log2N+leadingCountBits+doubleCenterBits)
				w.WriteBits(xor>>uint(trailing), sigBits)
				prevLeading = 64 + 1
			} else if leading == prevLeading {
				w.WriteBits(0b10, ctrlFlagBits)
				w.WriteBits(xor, 64-leading)
			} else {
				prevLeading = leading
				w.WriteBits(uint64(24+int(leadingEncode[leading])), ctrlFlagBits+leadingCountBits)
				w.WriteBits(xor, 64-leading)
			}
		}

		current = (current + 1) & nMask
		previous[current] = value
		index++
		indices[key] = index
	}
	w.Flush()
	return w.Bytes()
}

func (s *Scratch) appendDecodeChimpNFloat64s(dst []float64, data []byte, n int) ([]float64, error) {
	assertN(n)
	if len(data) < 2 {
		return dst, codecerr.ErrMalformed
	}
	var r bitpack.Reader
	r.Reset(data)

	// Ring buffer writes precede reads within each call; no zeroing needed.
	s.prev64 = growUint64Slice(s.prev64, n)
	previous := s.prev64

	log2N := 31 - bits.LeadingZeros32(uint32(n))
	prevMask := (1 << uint(log2N)) - 1
	nMask := n - 1

	current := 0
	value := r.ReadBits(64)
	dst = append(dst, math.Float64frombits(value))
	previous[current] = value

	prevLeading := 0
	for r.HasMore() {
		var b uint64
		switch r.ReadBits(ctrlFlagBits) {
		case 0b11:
			prevLeading = int(leadingDecode[r.ReadBits(leadingCountBits)])
			fallthrough
		case 0b10:
			b = r.ReadBits(64 - prevLeading)
		case 0b01:
			meta := int(r.ReadBits(log2N + leadingCountBits + doubleCenterBits))
			idx := (meta >> (leadingCountBits + doubleCenterBits)) & prevMask
			prevLeading = int(leadingDecode[(meta>>doubleCenterBits)&leadingCountMask])
			sigBits := meta & doubleCenterMask
			if sigBits == 0 {
				// Java quirk: 64 wraps to 0 in the 6-bit field; treat as 64.
				sigBits = 64
			}
			trailing := 64 - sigBits - prevLeading
			b = r.ReadBits(sigBits) << uint(trailing)
			value = previous[idx]
		default: // 0b00
			value = previous[r.ReadBits(log2N)]
		}

		value ^= b
		dst = append(dst, math.Float64frombits(value))

		current = (current + 1) & nMask
		previous[current] = value
	}
	if err := r.Err(); err != nil {
		return dst, err
	}
	return dst, nil
}

func (s *Scratch) appendEncodeChimpNFloat32s(dst []byte, vs []float32, n int) []byte {
	assertN(n)
	if len(vs) == 0 {
		return dst
	}
	var w bitpack.Writer
	w.Reset(dst)

	log2N := 31 - bits.LeadingZeros32(uint32(n))
	threshold := maxLog2_32 + log2N
	nMask := n - 1

	s.prev32 = growUint32Slice(s.prev32, n)
	previous := s.prev32

	indicesSize := 1 << uint(threshold+1)
	maskLSB := indicesSize - 1
	s.idx = growIntSlice(s.idx, indicesSize)
	indices := s.idx
	zeroInts(indices)

	index := 0
	current := 0
	prevLeading := 0

	previous[current] = math.Float32bits(vs[0])
	w.WriteBits(uint64(previous[current]), 32)
	indices[int(previous[current])&maskLSB] = index

	for step := 1; step < len(vs); step++ {
		value := math.Float32bits(vs[step])
		key := int(value) & maskLSB

		var xor uint32
		var trailing int
		var prevIndex int
		currIndex := indices[key]
		if (index - currIndex) < n {
			tempXor := value ^ previous[currIndex&nMask]
			trailing = bits.TrailingZeros32(tempXor)
			if trailing > threshold {
				prevIndex = currIndex & nMask
				xor = tempXor
			} else {
				prevIndex = index & nMask
				xor = previous[prevIndex] ^ value
			}
		} else {
			prevIndex = index & nMask
			xor = previous[prevIndex] ^ value
		}

		if xor == 0 {
			w.WriteBits(uint64(prevIndex), ctrlFlagBits+log2N)
			prevLeading = 32 + 1
		} else {
			leading := int(leadingRound[bits.LeadingZeros32(xor)])
			if trailing > threshold {
				sigBits := 32 - leading - trailing
				sigMask := (uint64(1) << uint(sigBits)) - 1
				var meta uint64 = 0b01
				meta = (meta << uint(log2N)) | uint64(prevIndex)
				meta = (meta << leadingCountBits) | uint64(leadingEncode[leading])
				meta = (meta << floatCenterBits) | uint64(sigBits)
				w.WriteBits(
					(meta<<uint(sigBits))|((uint64(xor)>>uint(trailing))&sigMask),
					ctrlFlagBits+log2N+leadingCountBits+floatCenterBits+sigBits,
				)
				prevLeading = 32 + 1
			} else if leading == prevLeading {
				sigBits := 32 - leading
				sigMask := (uint64(1) << uint(sigBits)) - 1
				w.WriteBits(
					(uint64(0b10)<<uint(sigBits))|(uint64(xor)&sigMask),
					ctrlFlagBits+sigBits,
				)
			} else {
				prevLeading = leading
				sigBits := 32 - leading
				sigMask := (uint64(1) << uint(sigBits)) - 1
				meta := uint64(24 + int(leadingEncode[leading]))
				w.WriteBits(
					(meta<<uint(sigBits))|(uint64(xor)&sigMask),
					ctrlFlagBits+leadingCountBits+sigBits,
				)
			}
		}

		current = (current + 1) & nMask
		previous[current] = value
		index++
		indices[key] = index
	}
	w.Flush()
	return w.Bytes()
}

func (s *Scratch) appendDecodeChimpNFloat32s(dst []float32, data []byte, n int) ([]float32, error) {
	assertN(n)
	if len(data) < 2 {
		return dst, codecerr.ErrMalformed
	}
	var r bitpack.Reader
	r.Reset(data)

	s.prev32 = growUint32Slice(s.prev32, n)
	previous := s.prev32

	log2N := 31 - bits.LeadingZeros32(uint32(n))
	prevMask := (1 << uint(log2N)) - 1
	nMask := n - 1

	current := 0
	value := uint32(r.ReadBits(32))
	dst = append(dst, math.Float32frombits(value))
	previous[current] = value

	prevLeading := 0
	for r.HasMore() {
		var b uint32
		switch r.ReadBits(ctrlFlagBits) {
		case 0b11:
			prevLeading = int(leadingDecode[r.ReadBits(leadingCountBits)])
			fallthrough
		case 0b10:
			b = uint32(r.ReadBits(32 - prevLeading))
		case 0b01:
			meta := int(r.ReadBits(log2N + leadingCountBits + floatCenterBits))
			idx := (meta >> (leadingCountBits + floatCenterBits)) & prevMask
			prevLeading = int(leadingDecode[(meta>>floatCenterBits)&leadingCountMask])
			sigBits := meta & floatCenterMask
			if sigBits == 0 {
				sigBits = 32
			}
			trailing := 32 - sigBits - prevLeading
			b = uint32(r.ReadBits(sigBits)) << uint(trailing)
			value = previous[idx]
		default: // 0b00
			value = previous[r.ReadBits(log2N)]
		}

		value ^= b
		dst = append(dst, math.Float32frombits(value))

		current = (current + 1) & nMask
		previous[current] = value
	}
	if err := r.Err(); err != nil {
		return dst, err
	}
	return dst, nil
}
