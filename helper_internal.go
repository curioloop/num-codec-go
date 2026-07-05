package numcodec

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/curioloop/num-codec-go/internal/codecerr"
	"github.com/curioloop/num-codec-go/simple8"
	"github.com/curioloop/num-codec-go/varint"
)

// int32Like constrains 32-bit integer types accepted by the shared
// int32/uint32 encode/decode logic. Both share the same GC shape so the
// generic function has a single instantiation with no dictionary
// overhead in practice.
type int32Like interface{ ~int32 | ~uint32 }

// int64Like is the 64-bit counterpart.
type int64Like interface{ ~int64 | ~uint64 }

// appendDecodeInt32Like decodes into a slice of int32 or uint32 according
// to flags. Both types receive the same underlying bit pattern
// (ZigZag/Uvarint semantics are preserved bit-exactly across the cast).
func appendDecodeInt32Like[T int32Like](dst []T, data []byte, flags Flags) ([]T, error) {
	switch {
	case flags&Simple8 != 0 && flags&(ZigZag|Uvarint) != 0:
		return appendDecodeSimple8Int32Like[T](dst, data, flags&ZigZag != 0)
	case flags&ZigZag != 0:
		for len(data) > 0 {
			v, n, err := varint.ZigZag32(data)
			if err != nil {
				return dst, err
			}
			dst = append(dst, T(v))
			data = data[n:]
		}
		return dst, nil
	case flags&Uvarint != 0:
		for len(data) > 0 {
			v, n, err := varint.Uvarint32(data)
			if err != nil {
				return dst, err
			}
			dst = append(dst, T(v))
			data = data[n:]
		}
		return dst, nil
	case flags == Raw:
		return appendDecodeRawInt32Like[T](dst, data)
	default:
		return dst, fmt.Errorf("numcodec: AppendDecodeInt32s/Uint32s: %w (%s)", ErrBadFlags, flags)
	}
}

// appendDecodeInt64Like is the 64-bit counterpart.
func appendDecodeInt64Like[T int64Like](dst []T, data []byte, flags Flags) ([]T, error) {
	switch {
	case flags&Simple8 != 0 && flags&(ZigZag|Uvarint) != 0:
		return appendDecodeSimple8Int64Like[T](dst, data, flags&ZigZag != 0)
	case flags&ZigZag != 0:
		for len(data) > 0 {
			v, n, err := varint.ZigZag64(data)
			if err != nil {
				return dst, err
			}
			dst = append(dst, T(v))
			data = data[n:]
		}
		return dst, nil
	case flags&Uvarint != 0:
		for len(data) > 0 {
			v, n, err := varint.Uvarint64(data)
			if err != nil {
				return dst, err
			}
			dst = append(dst, T(v))
			data = data[n:]
		}
		return dst, nil
	case flags == Raw:
		return appendDecodeRawInt64Like[T](dst, data)
	default:
		return dst, fmt.Errorf("numcodec: AppendDecodeInt64s/Uint64s: %w (%s)", ErrBadFlags, flags)
	}
}

// appendDecodeSimple8Int32Like unpacks a Simple8b-encoded stream of varint
// bytes into T values. Streams via simple8.UnpackFunc so no intermediate
// []uint64 is allocated.
func appendDecodeSimple8Int32Like[T int32Like](dst []T, data []byte, signed bool) ([]T, error) {
	var buf [8]byte
	err := simple8.UnpackFunc(data, func(code uint64) error {
		binary.LittleEndian.PutUint64(buf[:], code)
		if signed {
			v, _, e := varint.ZigZag32(buf[:])
			if e != nil {
				return e
			}
			dst = append(dst, T(v))
			return nil
		}
		v, _, e := varint.Uvarint32(buf[:])
		if e != nil {
			return e
		}
		dst = append(dst, T(v))
		return nil
	})
	return dst, err
}

// appendDecodeSimple8Int64Like is the 64-bit counterpart.
func appendDecodeSimple8Int64Like[T int64Like](dst []T, data []byte, signed bool) ([]T, error) {
	var buf [8]byte
	err := simple8.UnpackFunc(data, func(code uint64) error {
		binary.LittleEndian.PutUint64(buf[:], code)
		if signed {
			v, _, e := varint.ZigZag64(buf[:])
			if e != nil {
				return e
			}
			dst = append(dst, T(v))
			return nil
		}
		v, _, e := varint.Uvarint64(buf[:])
		if e != nil {
			return e
		}
		dst = append(dst, T(v))
		return nil
	})
	return dst, err
}

func appendDecodeRawInt32Like[T int32Like](dst []T, data []byte) ([]T, error) {
	if len(data)%4 != 0 {
		return dst, codecerr.ErrMalformed
	}
	for i := 0; i < len(data); i += 4 {
		dst = append(dst, T(binary.BigEndian.Uint32(data[i:])))
	}
	return dst, nil
}

func appendDecodeRawInt64Like[T int64Like](dst []T, data []byte) ([]T, error) {
	if len(data)%8 != 0 {
		return dst, codecerr.ErrMalformed
	}
	for i := 0; i < len(data); i += 8 {
		dst = append(dst, T(binary.BigEndian.Uint64(data[i:])))
	}
	return dst, nil
}

func appendDecodeRawFloat32s(dst []float32, data []byte) ([]float32, error) {
	if len(data)%4 != 0 {
		return dst, codecerr.ErrMalformed
	}
	for i := 0; i < len(data); i += 4 {
		dst = append(dst, math.Float32frombits(binary.BigEndian.Uint32(data[i:])))
	}
	return dst, nil
}

func appendDecodeRawFloat64s(dst []float64, data []byte) ([]float64, error) {
	if len(data)%8 != 0 {
		return dst, codecerr.ErrMalformed
	}
	for i := 0; i < len(data); i += 8 {
		dst = append(dst, math.Float64frombits(binary.BigEndian.Uint64(data[i:])))
	}
	return dst, nil
}
