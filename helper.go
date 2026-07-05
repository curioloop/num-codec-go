// Package-level API and thin convenience wrappers over Encoder. See
// doc.go for the top-level package overview.
package numcodec

import (
	"fmt"

	"github.com/curioloop/num-codec-go/chimp"
	"github.com/curioloop/num-codec-go/delta2"
	"github.com/curioloop/num-codec-go/gorilla"
	"github.com/curioloop/num-codec-go/internal/chimpcodec"
)

// DefaultChimpN is the ring-buffer size used by AppendEncodeFloat32s /
// AppendEncodeFloat64s when Chimp is chosen. Must be a power of two in
// [4, 256]; see chimp.DefaultN.
const DefaultChimpN = chimp.DefaultN

// ---------------------------------------------------------------------------
// Package-level Encode wrappers: allocate a fresh zero-value Encoder per
// call. For heavy workloads instantiate an [Encoder] once and reuse it.
// ---------------------------------------------------------------------------

// AppendEncodeInt32s encodes vs via ZigZag+Simple8b, bare ZigZag, then raw
// big-endian; whichever is smallest wins.
//
// Allocates ~ len(vs)*8 bytes of internal scratch per call; heavy
// callers should reuse an [Encoder] via [Encoder.AppendEncodeInt32s].
func AppendEncodeInt32s(dst []byte, vs []int32) ([]byte, Flags, error) {
	var e Encoder
	return e.AppendEncodeInt32s(dst, vs)
}

// AppendEncodeUint32s encodes vs via Uvarint+Simple8b, bare Uvarint, then
// raw big-endian; whichever is smallest wins.
//
// See [Encoder.AppendEncodeUint32s] for scratch reuse.
func AppendEncodeUint32s(dst []byte, vs []uint32) ([]byte, Flags, error) {
	var e Encoder
	return e.AppendEncodeUint32s(dst, vs)
}

// AppendEncodeInt64s encodes vs via ZigZag+Simple8b, bare ZigZag, then raw
// big-endian; whichever is smallest wins.
//
// See [Encoder.AppendEncodeInt64s] for scratch reuse.
func AppendEncodeInt64s(dst []byte, vs []int64) ([]byte, Flags, error) {
	var e Encoder
	return e.AppendEncodeInt64s(dst, vs)
}

// AppendEncodeUint64s encodes vs via Uvarint+Simple8b, bare Uvarint, then
// raw big-endian; whichever is smallest wins.
//
// See [Encoder.AppendEncodeUint64s] for scratch reuse.
func AppendEncodeUint64s(dst []byte, vs []uint64) ([]byte, Flags, error) {
	var e Encoder
	return e.AppendEncodeUint64s(dst, vs)
}

// AppendEncodeFloat32s encodes vs via Gorilla, then Chimp
// (N=DefaultChimpN), then raw IEEE 754; whichever is smallest wins.
//
// See [Encoder.AppendEncodeFloat32s] for scratch reuse.
func AppendEncodeFloat32s(dst []byte, vs []float32) ([]byte, Flags, error) {
	var e Encoder
	return e.AppendEncodeFloat32s(dst, vs)
}

// AppendEncodeFloat64s encodes vs via Gorilla, then Chimp
// (N=DefaultChimpN), then raw IEEE 754; whichever is smallest wins.
//
// See [Encoder.AppendEncodeFloat64s] for scratch reuse.
func AppendEncodeFloat64s(dst []byte, vs []float64) ([]byte, Flags, error) {
	var e Encoder
	return e.AppendEncodeFloat64s(dst, vs)
}

// AppendEncodeDelta2 encodes vs via Delta2+Simple8b (ordered path) when
// values are monotonically non-decreasing, otherwise falls back to
// Delta2+ZigZag+Uvarint.
//
// See [Encoder.AppendEncodeDelta2] for scratch reuse.
func AppendEncodeDelta2(dst []byte, vs []int64) ([]byte, Flags, error) {
	var e Encoder
	return e.AppendEncodeDelta2(dst, vs)
}

// ---------------------------------------------------------------------------
// Decode: no scratch to amortise, so no Decoder type is needed.
// ---------------------------------------------------------------------------

// AppendDecodeInt32s decodes vs into dst according to flags.
func AppendDecodeInt32s(dst []int32, data []byte, flags Flags) ([]int32, error) {
	return appendDecodeInt32Like(dst, data, flags)
}

// AppendDecodeUint32s decodes vs into dst according to flags.
func AppendDecodeUint32s(dst []uint32, data []byte, flags Flags) ([]uint32, error) {
	return appendDecodeInt32Like(dst, data, flags)
}

// AppendDecodeInt64s decodes vs into dst.
func AppendDecodeInt64s(dst []int64, data []byte, flags Flags) ([]int64, error) {
	return appendDecodeInt64Like(dst, data, flags)
}

// AppendDecodeUint64s decodes vs into dst.
func AppendDecodeUint64s(dst []uint64, data []byte, flags Flags) ([]uint64, error) {
	return appendDecodeInt64Like(dst, data, flags)
}

// AppendDecodeFloat32s decodes vs into dst.
func AppendDecodeFloat32s(dst []float32, data []byte, flags Flags) ([]float32, error) {
	switch flags {
	case Gorilla:
		return gorilla.AppendDecodeFloat32s(dst, data)
	case Chimp:
		return chimpcodec.AppendDecodeFloat32s(dst, data)
	case Raw:
		return appendDecodeRawFloat32s(dst, data)
	default:
		return dst, fmt.Errorf("numcodec: AppendDecodeFloat32s: %w (%s)", ErrBadFlags, flags)
	}
}

// AppendDecodeFloat64s decodes vs into dst.
func AppendDecodeFloat64s(dst []float64, data []byte, flags Flags) ([]float64, error) {
	switch flags {
	case Gorilla:
		return gorilla.AppendDecodeFloat64s(dst, data)
	case Chimp:
		return chimpcodec.AppendDecodeFloat64s(dst, data)
	case Raw:
		return appendDecodeRawFloat64s(dst, data)
	default:
		return dst, fmt.Errorf("numcodec: AppendDecodeFloat64s: %w (%s)", ErrBadFlags, flags)
	}
}

// AppendDecodeDelta2 decodes into dst.
func AppendDecodeDelta2(dst []int64, data []byte, flags Flags) ([]int64, error) {
	if flags&Delta2 == 0 || flags&(Simple8|ZigZag) == 0 {
		return dst, fmt.Errorf("numcodec: AppendDecodeDelta2: %w (%s)", ErrBadFlags, flags)
	}
	if flags&Simple8 != 0 {
		return delta2.AppendDecodeOrdered(dst, data)
	}
	return delta2.AppendDecodeUnordered(dst, data)
}

// growAppend expands dst to reserve n bytes of tail space then returns
// the slice header with the original length (ready for further appends
// without reallocation until n bytes are added).
func growAppend(dst []byte, n int) []byte {
	if cap(dst)-len(dst) < n {
		bigger := make([]byte, len(dst), len(dst)+n)
		copy(bigger, dst)
		return bigger
	}
	return dst
}
