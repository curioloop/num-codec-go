package numcodec

import (
	"encoding/binary"
	"errors"
	"math"

	"github.com/curioloop/num-codec-go/chimp"
	"github.com/curioloop/num-codec-go/delta2"
	"github.com/curioloop/num-codec-go/gorilla"
	"github.com/curioloop/num-codec-go/internal/chimpcodec"
	"github.com/curioloop/num-codec-go/internal/codecerr"
	"github.com/curioloop/num-codec-go/simple8"
	"github.com/curioloop/num-codec-go/varint"
)

// Encoder amortises the internal scratch buffers used by every
// AppendEncode* helper across many calls. A zero-value Encoder is ready
// to use:
//
//	var enc numcodec.Encoder
//	for _, series := range everything {
//	    data, flags, _ := enc.AppendEncodeFloat64s(dst[:0], series)
//	    ...
//	}
//
// An Encoder is not safe for concurrent use by multiple goroutines. For
// parallel workloads wrap one per goroutine in a sync.Pool.
//
// Buffers grow monotonically to fit the largest payload seen. If encode
// sizes vary wildly, drop the Encoder (or replace it with a fresh one)
// periodically to release memory back to the runtime.
//
// Decoding is inherently zero-alloc so there is no matching Decoder
// type; call the package-level AppendDecode* functions directly.
type Encoder struct {
	// packed holds the varint→uint64 pipeline scratch used by the
	// int32/uint32/int64/uint64 encode paths and by the Delta2 ordered
	// path (which reuses it as the delta slice fed to Simple8b).
	packed []uint64

	// s8 holds the Simple8b output written for size-comparison against
	// the raw and bare-varint alternatives. Cached across integer encode
	// calls.
	s8 []byte

	// fbuf holds the Gorilla / Chimp encoded output written for
	// size-comparison against the raw IEEE 754 alternative. Cached
	// across float encode calls.
	fbuf []byte

	// chimp holds the ChimpN ring buffer + LSB-index scratch. Cached
	// across float encode/decode calls that end up in the Chimp branch.
	chimp chimp.Scratch
}

// Reset returns e to its zero-length state while preserving the capacity
// of every internal scratch buffer. Follows the bytes.Buffer convention:
// use it between two calls if you want to keep the amortised allocation
// pattern but symbolically separate the two encoding sessions. For a
// full release of the underlying memory, assign a fresh Encoder instead:
//
//	e = numcodec.Encoder{}
func (e *Encoder) Reset() {
	if e.packed != nil {
		e.packed = e.packed[:0]
	}
	if e.s8 != nil {
		e.s8 = e.s8[:0]
	}
	if e.fbuf != nil {
		e.fbuf = e.fbuf[:0]
	}
	// chimp.Scratch has its own internal buffers; nothing to reset here
	// because they get overwritten on the next encode call.
}

// growUint64 returns s truncated or reallocated to hold exactly n values.
// Contents are not preserved.
func growUint64(s []uint64, n int) []uint64 {
	if cap(s) < n {
		return make([]uint64, n)
	}
	return s[:n]
}

// growBytes returns s reset to length 0 with capacity ≥ n. Contents are
// not preserved.
func growBytes(s []byte, n int) []byte {
	if cap(s) < n {
		return make([]byte, 0, n)
	}
	return s[:0]
}

// AppendEncodeInt32s encodes vs via ZigZag+Simple8b, bare ZigZag, then raw
// big-endian; whichever is smallest wins.
func (e *Encoder) AppendEncodeInt32s(dst []byte, vs []int32) ([]byte, Flags, error) {
	return encodeInt32Like(e, dst, vs, true)
}

// AppendEncodeUint32s encodes vs via Uvarint+Simple8b, bare Uvarint, then
// raw big-endian; whichever is smallest wins.
func (e *Encoder) AppendEncodeUint32s(dst []byte, vs []uint32) ([]byte, Flags, error) {
	return encodeInt32Like(e, dst, vs, false)
}

// AppendEncodeInt64s encodes vs via ZigZag+Simple8b, bare ZigZag, then raw
// big-endian; whichever is smallest wins.
func (e *Encoder) AppendEncodeInt64s(dst []byte, vs []int64) ([]byte, Flags, error) {
	return encodeInt64Like(e, dst, vs, true)
}

// AppendEncodeUint64s encodes vs via Uvarint+Simple8b, bare Uvarint, then
// raw big-endian; whichever is smallest wins.
func (e *Encoder) AppendEncodeUint64s(dst []byte, vs []uint64) ([]byte, Flags, error) {
	return encodeInt64Like(e, dst, vs, false)
}

// AppendEncodeFloat64s encodes vs via Gorilla, then Chimp (N=DefaultChimpN),
// then raw big-endian; whichever is smallest wins.
func (e *Encoder) AppendEncodeFloat64s(dst []byte, vs []float64) ([]byte, Flags, error) {
	if len(vs) == 0 {
		return dst, 0, errors.New("numcodec: empty input")
	}
	rawSize := 8 * len(vs)

	e.fbuf = growBytes(e.fbuf, rawSize+16)
	gor := gorilla.AppendEncodeFloat64s(e.fbuf, vs)
	e.fbuf = gor
	if len(gor) < rawSize {
		return append(dst, gor...), Gorilla, nil
	}

	e.fbuf = e.fbuf[:0]
	chi := chimpcodec.ScratchEncodeFloat64s(&e.chimp, e.fbuf, vs, DefaultChimpN)
	e.fbuf = chi
	if len(chi) < rawSize {
		return append(dst, chi...), Chimp, nil
	}

	dst = growAppend(dst, rawSize)
	for _, v := range vs {
		dst = binary.BigEndian.AppendUint64(dst, math.Float64bits(v))
	}
	return dst, Raw, nil
}

// AppendEncodeFloat32s is the float32 counterpart of AppendEncodeFloat64s.
func (e *Encoder) AppendEncodeFloat32s(dst []byte, vs []float32) ([]byte, Flags, error) {
	if len(vs) == 0 {
		return dst, 0, errors.New("numcodec: empty input")
	}
	rawSize := 4 * len(vs)

	e.fbuf = growBytes(e.fbuf, rawSize+16)
	gor := gorilla.AppendEncodeFloat32s(e.fbuf, vs)
	e.fbuf = gor
	if len(gor) < rawSize {
		return append(dst, gor...), Gorilla, nil
	}

	e.fbuf = e.fbuf[:0]
	chi := chimpcodec.ScratchEncodeFloat32s(&e.chimp, e.fbuf, vs, DefaultChimpN)
	e.fbuf = chi
	if len(chi) < rawSize {
		return append(dst, chi...), Chimp, nil
	}

	dst = growAppend(dst, rawSize)
	for _, v := range vs {
		dst = binary.BigEndian.AppendUint32(dst, math.Float32bits(v))
	}
	return dst, Raw, nil
}

// AppendEncodeDelta2 encodes vs via Delta2+Simple8b (ordered path) when
// values are monotonically non-decreasing, otherwise falls back to
// Delta2+ZigZag+Uvarint.
func (e *Encoder) AppendEncodeDelta2(dst []byte, vs []int64) ([]byte, Flags, error) {
	if len(vs) == 0 {
		return dst, 0, errors.New("numcodec: empty input")
	}
	start := len(dst)
	out, ok, err := e.tryDelta2Ordered(dst, vs)
	if err != nil {
		return dst[:start], 0, err
	}
	if ok {
		return out, Delta2 | Simple8, nil
	}
	// Unordered fallback is inherently zero-scratch (ZigZag+Varint stream).
	out = delta2.AppendEncodeUnordered(dst[:start], vs)
	return out, Delta2 | ZigZag, nil
}

// tryDelta2Ordered inlines delta2.AppendEncodeOrdered but reuses e.packed
// for the delta slice. Returns (dst, true, nil) on success,
// (dst, false, nil) if the input violates the ordered invariant (caller
// should retry with the unordered path), or (dst, false, err) on any
// other error.
func (e *Encoder) tryDelta2Ordered(dst []byte, vs []int64) ([]byte, bool, error) {
	base := vs[0]
	dst = binary.BigEndian.AppendUint64(dst, uint64(base))
	if len(vs) == 1 {
		return dst, true, nil
	}
	e.packed = growUint64(e.packed, len(vs)-1)
	deltas := e.packed
	prev := base
	for i := 1; i < len(vs); i++ {
		d := vs[i] - prev
		if d < 0 {
			return dst, false, nil
		}
		deltas[i-1] = uint64(d)
		prev = vs[i]
	}
	out, err := simple8.AppendPack(dst, deltas)
	if err != nil {
		if errors.Is(err, codecerr.ErrOverflow) {
			return dst, false, nil
		}
		return dst, false, err
	}
	return out, true, nil
}

// encodeInt32Like is the shared implementation for the Encoder int32/uint32
// methods and the package-level counterparts (which pass a zero-value
// Encoder). The wire format is identical between the two paths — the
// only difference is whether ZigZag transformation is applied before
// Uvarint encoding.
func encodeInt32Like[T int32Like](e *Encoder, dst []byte, vs []T, signed bool) ([]byte, Flags, error) {
	if len(vs) == 0 {
		return dst, 0, errors.New("numcodec: empty input")
	}
	rawSize := 4 * len(vs)

	e.packed = growUint64(e.packed, len(vs))
	packed := e.packed

	var varintBytes int
	for i, v := range vs {
		var buf [varint.MaxLen32]byte
		var enc []byte
		if signed {
			enc = varint.AppendZigZag32(buf[:0], int32(v))
		} else {
			enc = varint.AppendUvarint32(buf[:0], uint32(v))
		}
		n := len(enc)
		varintBytes += n
		packed[i] = packLE(enc, n)
	}

	// Java's helper compares simple8-vs-varint using a call-count-inflated
	// byte tally (each value is probed 2× during Simple8 packing), so it
	// prefers Simple8 more aggressively than a "true" byte comparison.
	// Matching that heuristic for wire compat: pick Simple8 whenever it
	// succeeds and beats raw. Pre-size the scratch to rawSize+8 so
	// simple8.AppendPack never has to realloc.
	e.s8 = growBytes(e.s8, rawSize+8)
	s8, err := simple8.AppendPack(e.s8, packed)
	e.s8 = s8
	if err == nil && len(s8) < rawSize {
		flag := Simple8 | ZigZag
		if !signed {
			flag = Simple8 | Uvarint
		}
		return append(dst, s8...), flag, nil
	}
	if err != nil && !errors.Is(err, codecerr.ErrOverflow) {
		return dst, 0, err
	}

	if varintBytes < rawSize {
		if signed {
			for _, v := range vs {
				dst = varint.AppendZigZag32(dst, int32(v))
			}
			return dst, ZigZag, nil
		}
		for _, v := range vs {
			dst = varint.AppendUvarint32(dst, uint32(v))
		}
		return dst, Uvarint, nil
	}

	dst = growAppend(dst, rawSize)
	for _, v := range vs {
		dst = binary.BigEndian.AppendUint32(dst, uint32(v))
	}
	return dst, Raw, nil
}

// encodeInt64Like is the 64-bit counterpart. Simple8b can only carry
// values ≤ 8 bytes wide once varint-encoded; oversized values force the
// bare-varint fallback.
func encodeInt64Like[T int64Like](e *Encoder, dst []byte, vs []T, signed bool) ([]byte, Flags, error) {
	if len(vs) == 0 {
		return dst, 0, errors.New("numcodec: empty input")
	}
	rawSize := 8 * len(vs)

	e.packed = growUint64(e.packed, 0)[:0]
	// We may skip some values (n > 8) — build packed incrementally.
	packed := e.packed
	if cap(packed) < len(vs) {
		packed = make([]uint64, 0, len(vs))
	}

	var varintBytes int
	simple8Possible := true
	for _, v := range vs {
		var buf [varint.MaxLen64]byte
		var enc []byte
		if signed {
			enc = varint.AppendZigZag64(buf[:0], int64(v))
		} else {
			enc = varint.AppendUvarint64(buf[:0], uint64(v))
		}
		n := len(enc)
		varintBytes += n
		if n > 8 {
			simple8Possible = false
			continue
		}
		if simple8Possible {
			packed = append(packed, packLE(enc, n))
		}
	}
	e.packed = packed

	if simple8Possible {
		e.s8 = growBytes(e.s8, rawSize+8)
		s8, err := simple8.AppendPack(e.s8, packed)
		e.s8 = s8
		if err == nil && len(s8) < rawSize {
			flag := Simple8 | ZigZag
			if !signed {
				flag = Simple8 | Uvarint
			}
			return append(dst, s8...), flag, nil
		}
		if err != nil && !errors.Is(err, codecerr.ErrOverflow) {
			return dst, 0, err
		}
	}

	if varintBytes < rawSize {
		if signed {
			for _, v := range vs {
				dst = varint.AppendZigZag64(dst, int64(v))
			}
			return dst, ZigZag, nil
		}
		for _, v := range vs {
			dst = varint.AppendUvarint64(dst, uint64(v))
		}
		return dst, Uvarint, nil
	}

	dst = growAppend(dst, rawSize)
	for _, v := range vs {
		dst = binary.BigEndian.AppendUint64(dst, uint64(v))
	}
	return dst, Raw, nil
}

// packLE packs the first n bytes (n ≤ 8) of enc into a uint64 with byte 0
// in the low 8 bits, byte 1 in the next, etc. Matches the Java
// VarIntCodec pipeline used by Simple8b integer encoding.
func packLE(enc []byte, n int) uint64 {
	var code uint64
	for j := 0; j < n; j++ {
		code |= uint64(enc[j]) << (8 * j)
	}
	return code
}
