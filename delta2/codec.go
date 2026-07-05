// Package delta2 implements Delta2 (a.k.a. delta-of-delta) encoding for
// int64 sequences. Two paths, chosen by the caller:
//
//   - AppendEncodeOrdered: first value stored raw as a big-endian int64,
//     followed by Simple8b-packed non-negative deltas. Returns
//     ErrOverflow if any delta is negative or exceeds 60 bits — caller
//     retries with the unordered variant.
//   - AppendEncodeUnordered: every value encoded as a ZigZag+Uvarint delta
//     from the previous (or from 0 for the first). Cannot overflow.
package delta2

import (
	"encoding/binary"

	"github.com/curioloop/num-codec-go/internal/codecerr"
	"github.com/curioloop/num-codec-go/simple8"
	"github.com/curioloop/num-codec-go/varint"
)

// AppendEncodeOrdered encodes vs as base + Simple8b-packed positive deltas.
// Returns ErrOverflow if any delta is negative or does not fit in 60 bits.
func AppendEncodeOrdered(dst []byte, vs []int64) ([]byte, error) {
	if len(vs) == 0 {
		return dst, nil
	}
	base := vs[0]
	dst = binary.BigEndian.AppendUint64(dst, uint64(base))
	if len(vs) == 1 {
		return dst, nil
	}
	// Build the delta stream in a scratch buffer, checking for negatives.
	deltas := make([]uint64, len(vs)-1)
	prev := base
	for i := 1; i < len(vs); i++ {
		d := vs[i] - prev
		if d < 0 {
			return dst, codecerr.ErrOverflow
		}
		deltas[i-1] = uint64(d)
		prev = vs[i]
	}
	return simple8.AppendPack(dst, deltas)
}

// AppendEncodeUnordered encodes vs as ZigZag+Uvarint deltas from the
// previous value (0 for the first). Cannot overflow.
func AppendEncodeUnordered(dst []byte, vs []int64) []byte {
	var prev int64
	for _, v := range vs {
		dst = varint.AppendZigZag64(dst, v-prev)
		prev = v
	}
	return dst
}

// AppendDecodeOrdered decodes an ordered Delta2 stream, appending values
// to dst.
func AppendDecodeOrdered(dst []int64, data []byte) ([]int64, error) {
	if len(data) < 8 {
		return dst, codecerr.ErrMalformed
	}
	base := int64(binary.BigEndian.Uint64(data))
	dst = append(dst, base)
	if len(data) == 8 {
		return dst, nil
	}
	acc := base
	err := simple8.UnpackFunc(data[8:], func(d uint64) error {
		acc += int64(d)
		dst = append(dst, acc)
		return nil
	})
	return dst, err
}

// AppendDecodeUnordered decodes an unordered Delta2 stream.
func AppendDecodeUnordered(dst []int64, data []byte) ([]int64, error) {
	var acc int64
	for len(data) > 0 {
		d, n, err := varint.ZigZag64(data)
		if err != nil {
			return dst, err
		}
		acc += d
		dst = append(dst, acc)
		data = data[n:]
	}
	return dst, nil
}
