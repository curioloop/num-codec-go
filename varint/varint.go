// Package varint implements variable-length integer encoding (Uvarint) and
// signed-to-unsigned transformation (ZigZag). Byte format matches the Java
// number-codec library exactly.
//
// Uvarint stores an unsigned integer as 1..N little-endian 7-bit groups
// with the high bit of every non-final byte set. N is 5 for uint32 and
// 10 for uint64. Signed integers go through the ZigZag transform first
// so small negatives encode as small unsigned values.
//
// The public API follows the AppendXxx convention: pass a destination
// []byte to which the encoded bytes are appended; the extended slice is
// returned. Decoders take a data slice and return the decoded value plus
// the number of bytes consumed.
package varint

import "github.com/curioloop/num-codec-go/internal/codecerr"

// MaxLen32 is the maximum encoded length of a uint32 Uvarint (5 bytes).
const MaxLen32 = 5

// MaxLen64 is the maximum encoded length of a uint64 Uvarint (10 bytes).
const MaxLen64 = 10

// AppendUvarint32 appends the Uvarint encoding of v to dst.
func AppendUvarint32(dst []byte, v uint32) []byte {
	for v > 0x7f {
		dst = append(dst, byte(v)|0x80)
		v >>= 7
	}
	return append(dst, byte(v))
}

// AppendUvarint64 appends the Uvarint encoding of v to dst.
func AppendUvarint64(dst []byte, v uint64) []byte {
	for v > 0x7f {
		dst = append(dst, byte(v)|0x80)
		v >>= 7
	}
	return append(dst, byte(v))
}

// AppendZigZag32 appends the ZigZag+Uvarint encoding of v to dst.
func AppendZigZag32(dst []byte, v int32) []byte {
	return AppendUvarint32(dst, uint32(v<<1)^uint32(v>>31))
}

// AppendZigZag64 appends the ZigZag+Uvarint encoding of v to dst.
func AppendZigZag64(dst []byte, v int64) []byte {
	return AppendUvarint64(dst, uint64(v<<1)^uint64(v>>63))
}

// Uvarint32 decodes a single Uvarint at buf[0:]. Returns the value, the
// number of bytes consumed, and ErrMalformed if buf is truncated or the
// encoding overflows uint32.
func Uvarint32(buf []byte) (uint32, int, error) {
	var v uint32
	var s uint
	for i, b := range buf {
		if i == MaxLen32 {
			return 0, 0, codecerr.ErrMalformed
		}
		if b < 0x80 {
			if i == MaxLen32-1 && b > 0x0f {
				return 0, 0, codecerr.ErrMalformed
			}
			return v | uint32(b)<<s, i + 1, nil
		}
		v |= uint32(b&0x7f) << s
		s += 7
	}
	return 0, 0, codecerr.ErrMalformed
}

// Uvarint64 decodes a single Uvarint at buf[0:].
func Uvarint64(buf []byte) (uint64, int, error) {
	var v uint64
	var s uint
	for i, b := range buf {
		if i == MaxLen64 {
			return 0, 0, codecerr.ErrMalformed
		}
		if b < 0x80 {
			if i == MaxLen64-1 && b > 0x01 {
				return 0, 0, codecerr.ErrMalformed
			}
			return v | uint64(b)<<s, i + 1, nil
		}
		v |= uint64(b&0x7f) << s
		s += 7
	}
	return 0, 0, codecerr.ErrMalformed
}

// ZigZag32 decodes a single ZigZag+Uvarint at buf[0:].
func ZigZag32(buf []byte) (int32, int, error) {
	u, n, err := Uvarint32(buf)
	if err != nil {
		return 0, 0, err
	}
	return int32(u>>1) ^ -int32(u&1), n, nil
}

// ZigZag64 decodes a single ZigZag+Uvarint at buf[0:].
func ZigZag64(buf []byte) (int64, int, error) {
	u, n, err := Uvarint64(buf)
	if err != nil {
		return 0, 0, err
	}
	return int64(u>>1) ^ -int64(u&1), n, nil
}
