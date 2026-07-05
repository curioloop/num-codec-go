// Package chimpcodec implements Chimp with a self-describing header byte
// carrying log2(N). Only the top-level numcodec package needs this
// wire format; standalone users of the chimp package always know their
// N out-of-band, so keeping this dispatcher out of chimp's public
// surface keeps that surface minimal.
package chimpcodec

import (
	"math/bits"

	"github.com/curioloop/num-codec-go/chimp"
	"github.com/curioloop/num-codec-go/internal/codecerr"
)

// AppendEncodeFloat64s prepends a header byte carrying log2(n) (or 0
// for base Chimp) then calls chimp.AppendEncodeFloat64s.
func AppendEncodeFloat64s(dst []byte, vs []float64, n int) []byte {
	dst = append(dst, headerByte(n))
	return chimp.AppendEncodeFloat64s(dst, vs, n)
}

// AppendDecodeFloat64s reads the header byte then dispatches to the
// matching decoder. Returns ErrMalformed for header values outside the
// valid set {0, 2..8}.
func AppendDecodeFloat64s(dst []float64, data []byte) ([]float64, error) {
	n, body, err := parseHeader(data)
	if err != nil {
		return dst, err
	}
	return chimp.AppendDecodeFloat64s(dst, body, n)
}

// AppendEncodeFloat32s is the float32 counterpart.
func AppendEncodeFloat32s(dst []byte, vs []float32, n int) []byte {
	dst = append(dst, headerByte(n))
	return chimp.AppendEncodeFloat32s(dst, vs, n)
}

// AppendDecodeFloat32s is the float32 counterpart.
func AppendDecodeFloat32s(dst []float32, data []byte) ([]float32, error) {
	n, body, err := parseHeader(data)
	if err != nil {
		return dst, err
	}
	return chimp.AppendDecodeFloat32s(dst, body, n)
}

// ScratchEncodeFloat64s / ScratchDecodeFloat64s / ScratchEncodeFloat32s /
// ScratchDecodeFloat32s: same as the four above but reuse a
// caller-provided chimp.Scratch. Only the ChimpN branch actually uses
// the scratch; base Chimp has no large internal state.
func ScratchEncodeFloat64s(s *chimp.Scratch, dst []byte, vs []float64, n int) []byte {
	dst = append(dst, headerByte(n))
	return s.AppendEncodeFloat64s(dst, vs, n)
}

func ScratchDecodeFloat64s(s *chimp.Scratch, dst []float64, data []byte) ([]float64, error) {
	n, body, err := parseHeader(data)
	if err != nil {
		return dst, err
	}
	return s.AppendDecodeFloat64s(dst, body, n)
}

func ScratchEncodeFloat32s(s *chimp.Scratch, dst []byte, vs []float32, n int) []byte {
	dst = append(dst, headerByte(n))
	return s.AppendEncodeFloat32s(dst, vs, n)
}

func ScratchDecodeFloat32s(s *chimp.Scratch, dst []float32, data []byte) ([]float32, error) {
	n, body, err := parseHeader(data)
	if err != nil {
		return dst, err
	}
	return s.AppendDecodeFloat32s(dst, body, n)
}

// headerByte returns 0 for base Chimp (n == 0) or log2(n) for ChimpN.
// Caller is responsible for ensuring n is 0 or a power of two in [4,256]
// (chimp.Append*Float*s validates this).
func headerByte(n int) byte {
	if n == 0 {
		return 0
	}
	return byte(bits.TrailingZeros(uint(n)))
}

// parseHeader consumes the leading log2(N) byte and returns (n, body,
// err). Valid header values are 0 (base) and 2..8 (ChimpN with N=4..256).
func parseHeader(data []byte) (int, []byte, error) {
	if len(data) < 1 {
		return 0, nil, codecerr.ErrMalformed
	}
	h := data[0]
	if h != 0 && (h < 2 || h > 8) {
		return 0, nil, codecerr.ErrMalformed
	}
	n := 0
	if h != 0 {
		n = 1 << uint(h)
	}
	return n, data[1:], nil
}
