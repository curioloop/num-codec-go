package simple8

import (
	"encoding/binary"

	"github.com/curioloop/num-codec-go/internal/codecerr"
)

// AppendPack appends the Simple8b-packed encoding of vs to dst, returning
// the extended slice. Returns ErrOverflow if any value ≥ 2^60.
func AppendPack(dst []byte, vs []uint64) ([]byte, error) {
	pos := 0
	length := len(vs)
	for pos < length {
		p := lookupPacking(vs, pos)
		if p == nil {
			return dst, codecerr.ErrOverflow
		}
		dst = binary.BigEndian.AppendUint64(dst, p.pack(vs, pos))
		pos += p.integersCoded
	}
	return dst, nil
}

// AppendUnpack appends the Simple8b-unpacked values from data to dst,
// returning the extended slice. Returns ErrMalformed if data length is
// not a multiple of 8 or the selector is out of range.
func AppendUnpack(dst []uint64, data []byte) ([]uint64, error) {
	if len(data)%8 != 0 {
		return dst, codecerr.ErrMalformed
	}
	for i := 0; i < len(data); i += 8 {
		word := binary.BigEndian.Uint64(data[i:])
		p := &selectors[word>>60]
		dst = p.unpack(word, dst)
	}
	return dst, nil
}

// UnpackFunc invokes fn once per value packed in data. Iteration stops
// and returns an error if fn returns non-nil, or if data is malformed.
// Prefer AppendUnpack unless you need to stream directly to a caller-owned
// sink without allocating an intermediate []uint64.
//
// Dispatched via a switch (not the packing struct's function pointer)
// so the compiler can prove fn does not escape through each specialised
// call site and keep the caller's closure on the stack.
func UnpackFunc(data []byte, fn func(v uint64) error) error {
	if len(data)%8 != 0 {
		return codecerr.ErrMalformed
	}
	for i := 0; i < len(data); i += 8 {
		word := binary.BigEndian.Uint64(data[i:])
		var err error
		switch word >> 60 {
		case 0:
			err = unpackFunc240(word, fn)
		case 1:
			err = unpackFunc120(word, fn)
		case 2:
			err = unpackFunc60(word, fn)
		case 3:
			err = unpackFunc30(word, fn)
		case 4:
			err = unpackFunc20(word, fn)
		case 5:
			err = unpackFunc15(word, fn)
		case 6:
			err = unpackFunc12(word, fn)
		case 7:
			err = unpackFunc10(word, fn)
		case 8:
			err = unpackFunc8(word, fn)
		case 9:
			err = unpackFunc7(word, fn)
		case 10:
			err = unpackFunc6(word, fn)
		case 11:
			err = unpackFunc5(word, fn)
		case 12:
			err = unpackFunc4(word, fn)
		case 13:
			err = unpackFunc3(word, fn)
		case 14:
			err = unpackFunc2(word, fn)
		case 15:
			err = unpackFunc1(word, fn)
		}
		if err != nil {
			return err
		}
	}
	return nil
}
