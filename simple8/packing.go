// Package simple8 packing tables. Each selector has its own hand-unrolled
// pack / unpack / unpackFunc implementation so the shift constants and
// integer count are visible to the compiler at each use — matching what
// the Java library does with per-Packing subclasses.
package simple8

// packing describes one of the 16 Simple8b schemes. Selectors 0 and 1
// (integersCoded==240 or 120, bitsPerInteger==0) store only the LSB of
// a single value repeated N times. Selectors 2..15 use a uniform-width
// bit-packing.
//
// pack and unpack are per-selector function pointers so the compiler can
// constant-fold the shifts and counts. UnpackFunc is dispatched via a
// switch on the selector nibble rather than through this struct so the
// caller's callback can stay on the stack (indirect calls defeat escape
// analysis for the fn parameter).
type packing struct {
	selector       uint64
	integersCoded  int
	bitsPerInteger int
	pack           packFn
	unpack         unpackFn
}

type (
	packFn   func(src []uint64, pos int) uint64
	unpackFn func(word uint64, dst []uint64) []uint64
	// unpackFuncFn is the callback-style variant used by UnpackFunc; kept
	// as a named type for readability at the definition sites even though
	// UnpackFunc calls the concrete functions directly through a switch.
	unpackFuncFn func(word uint64, fn func(uint64) error) error
)

// selectors is indexed by selector value (top nibble of the encoded word).
// Order matches Simple8Codec.selector[] in the Java library.
var selectors = [16]packing{
	{0, 240, 0, pack240, unpack240},
	{1, 120, 0, pack120, unpack120},
	{2, 60, 1, pack60, unpack60},
	{3, 30, 2, pack30, unpack30},
	{4, 20, 3, pack20, unpack20},
	{5, 15, 4, pack15, unpack15},
	{6, 12, 5, pack12, unpack12},
	{7, 10, 6, pack10, unpack10},
	{8, 8, 7, pack8, unpack8},
	{9, 7, 8, pack7, unpack7},
	{10, 6, 10, pack6, unpack6},
	{11, 5, 12, pack5, unpack5},
	{12, 4, 15, pack4, unpack4},
	{13, 3, 20, pack3, unpack3},
	{14, 2, 30, pack2, unpack2},
	{15, 1, 60, pack1, unpack1},
}

// ============================================================================
// Selector 0: 240 identical values, LSB only. Selector 1: same but 120.
// ============================================================================

func pack240(src []uint64, pos int) uint64 {
	// selector = 0 → top nibble stays 0; only the LSB matters.
	return src[pos] & 1
}

func unpack240(word uint64, dst []uint64) []uint64 {
	v := word & 1
	for i := 0; i < 240; i++ {
		dst = append(dst, v)
	}
	return dst
}

func unpackFunc240(word uint64, fn func(uint64) error) error {
	v := word & 1
	for i := 0; i < 240; i++ {
		if err := fn(v); err != nil {
			return err
		}
	}
	return nil
}

func pack120(src []uint64, pos int) uint64 {
	return (1 << 60) | (src[pos] & 1)
}

func unpack120(word uint64, dst []uint64) []uint64 {
	v := word & 1
	for i := 0; i < 120; i++ {
		dst = append(dst, v)
	}
	return dst
}

func unpackFunc120(word uint64, fn func(uint64) error) error {
	v := word & 1
	for i := 0; i < 120; i++ {
		if err := fn(v); err != nil {
			return err
		}
	}
	return nil
}

// ============================================================================
// Selectors 2..15: uniform-width packing, hand-unrolled per Java's Packing*.
// The three-index slice `src[pos:pos+N:pos+N]` gives the compiler a single
// bounds check per call and lets the shifts constant-fold.
// ============================================================================

// selector 2: 60 × 1 bit

func pack60(src []uint64, pos int) uint64 {
	s := src[pos : pos+60 : pos+60]
	return (2 << 60) |
		s[0] | s[1]<<1 | s[2]<<2 | s[3]<<3 | s[4]<<4 | s[5]<<5 |
		s[6]<<6 | s[7]<<7 | s[8]<<8 | s[9]<<9 | s[10]<<10 | s[11]<<11 |
		s[12]<<12 | s[13]<<13 | s[14]<<14 | s[15]<<15 | s[16]<<16 | s[17]<<17 |
		s[18]<<18 | s[19]<<19 | s[20]<<20 | s[21]<<21 | s[22]<<22 | s[23]<<23 |
		s[24]<<24 | s[25]<<25 | s[26]<<26 | s[27]<<27 | s[28]<<28 | s[29]<<29 |
		s[30]<<30 | s[31]<<31 | s[32]<<32 | s[33]<<33 | s[34]<<34 | s[35]<<35 |
		s[36]<<36 | s[37]<<37 | s[38]<<38 | s[39]<<39 | s[40]<<40 | s[41]<<41 |
		s[42]<<42 | s[43]<<43 | s[44]<<44 | s[45]<<45 | s[46]<<46 | s[47]<<47 |
		s[48]<<48 | s[49]<<49 | s[50]<<50 | s[51]<<51 | s[52]<<52 | s[53]<<53 |
		s[54]<<54 | s[55]<<55 | s[56]<<56 | s[57]<<57 | s[58]<<58 | s[59]<<59
}

func unpack60(word uint64, dst []uint64) []uint64 {
	for i := 0; i < 60; i++ {
		dst = append(dst, (word>>uint(i))&1)
	}
	return dst
}

func unpackFunc60(word uint64, fn func(uint64) error) error {
	for i := 0; i < 60; i++ {
		if err := fn((word >> uint(i)) & 1); err != nil {
			return err
		}
	}
	return nil
}

// selector 3: 30 × 2 bits

func pack30(src []uint64, pos int) uint64 {
	s := src[pos : pos+30 : pos+30]
	return (3 << 60) |
		s[0] | s[1]<<2 | s[2]<<4 | s[3]<<6 | s[4]<<8 |
		s[5]<<10 | s[6]<<12 | s[7]<<14 | s[8]<<16 | s[9]<<18 |
		s[10]<<20 | s[11]<<22 | s[12]<<24 | s[13]<<26 | s[14]<<28 |
		s[15]<<30 | s[16]<<32 | s[17]<<34 | s[18]<<36 | s[19]<<38 |
		s[20]<<40 | s[21]<<42 | s[22]<<44 | s[23]<<46 | s[24]<<48 |
		s[25]<<50 | s[26]<<52 | s[27]<<54 | s[28]<<56 | s[29]<<58
}

func unpack30(word uint64, dst []uint64) []uint64 {
	for i := 0; i < 30; i++ {
		dst = append(dst, (word>>uint(i*2))&3)
	}
	return dst
}

func unpackFunc30(word uint64, fn func(uint64) error) error {
	for i := 0; i < 30; i++ {
		if err := fn((word >> uint(i*2)) & 3); err != nil {
			return err
		}
	}
	return nil
}

// selector 4: 20 × 3 bits

func pack20(src []uint64, pos int) uint64 {
	s := src[pos : pos+20 : pos+20]
	return (4 << 60) |
		s[0] | s[1]<<3 | s[2]<<6 | s[3]<<9 | s[4]<<12 |
		s[5]<<15 | s[6]<<18 | s[7]<<21 | s[8]<<24 | s[9]<<27 |
		s[10]<<30 | s[11]<<33 | s[12]<<36 | s[13]<<39 | s[14]<<42 |
		s[15]<<45 | s[16]<<48 | s[17]<<51 | s[18]<<54 | s[19]<<57
}

func unpack20(word uint64, dst []uint64) []uint64 {
	for i := 0; i < 20; i++ {
		dst = append(dst, (word>>uint(i*3))&7)
	}
	return dst
}

func unpackFunc20(word uint64, fn func(uint64) error) error {
	for i := 0; i < 20; i++ {
		if err := fn((word >> uint(i*3)) & 7); err != nil {
			return err
		}
	}
	return nil
}

// selector 5: 15 × 4 bits

func pack15(src []uint64, pos int) uint64 {
	s := src[pos : pos+15 : pos+15]
	return (5 << 60) |
		s[0] | s[1]<<4 | s[2]<<8 | s[3]<<12 | s[4]<<16 |
		s[5]<<20 | s[6]<<24 | s[7]<<28 | s[8]<<32 | s[9]<<36 |
		s[10]<<40 | s[11]<<44 | s[12]<<48 | s[13]<<52 | s[14]<<56
}

func unpack15(word uint64, dst []uint64) []uint64 {
	for i := 0; i < 15; i++ {
		dst = append(dst, (word>>uint(i*4))&0xF)
	}
	return dst
}

func unpackFunc15(word uint64, fn func(uint64) error) error {
	for i := 0; i < 15; i++ {
		if err := fn((word >> uint(i*4)) & 0xF); err != nil {
			return err
		}
	}
	return nil
}

// selector 6: 12 × 5 bits

func pack12(src []uint64, pos int) uint64 {
	s := src[pos : pos+12 : pos+12]
	return (6 << 60) |
		s[0] | s[1]<<5 | s[2]<<10 | s[3]<<15 |
		s[4]<<20 | s[5]<<25 | s[6]<<30 | s[7]<<35 |
		s[8]<<40 | s[9]<<45 | s[10]<<50 | s[11]<<55
}

func unpack12(word uint64, dst []uint64) []uint64 {
	return append(dst,
		word&31, (word>>5)&31, (word>>10)&31, (word>>15)&31,
		(word>>20)&31, (word>>25)&31, (word>>30)&31, (word>>35)&31,
		(word>>40)&31, (word>>45)&31, (word>>50)&31, (word>>55)&31,
	)
}

func unpackFunc12(word uint64, fn func(uint64) error) error {
	if err := fn(word & 31); err != nil {
		return err
	}
	if err := fn((word >> 5) & 31); err != nil {
		return err
	}
	if err := fn((word >> 10) & 31); err != nil {
		return err
	}
	if err := fn((word >> 15) & 31); err != nil {
		return err
	}
	if err := fn((word >> 20) & 31); err != nil {
		return err
	}
	if err := fn((word >> 25) & 31); err != nil {
		return err
	}
	if err := fn((word >> 30) & 31); err != nil {
		return err
	}
	if err := fn((word >> 35) & 31); err != nil {
		return err
	}
	if err := fn((word >> 40) & 31); err != nil {
		return err
	}
	if err := fn((word >> 45) & 31); err != nil {
		return err
	}
	if err := fn((word >> 50) & 31); err != nil {
		return err
	}
	return fn((word >> 55) & 31)
}

// selector 7: 10 × 6 bits

func pack10(src []uint64, pos int) uint64 {
	s := src[pos : pos+10 : pos+10]
	return (7 << 60) |
		s[0] | s[1]<<6 | s[2]<<12 | s[3]<<18 | s[4]<<24 |
		s[5]<<30 | s[6]<<36 | s[7]<<42 | s[8]<<48 | s[9]<<54
}

func unpack10(word uint64, dst []uint64) []uint64 {
	return append(dst,
		word&63, (word>>6)&63, (word>>12)&63, (word>>18)&63, (word>>24)&63,
		(word>>30)&63, (word>>36)&63, (word>>42)&63, (word>>48)&63, (word>>54)&63,
	)
}

func unpackFunc10(word uint64, fn func(uint64) error) error {
	if err := fn(word & 63); err != nil {
		return err
	}
	if err := fn((word >> 6) & 63); err != nil {
		return err
	}
	if err := fn((word >> 12) & 63); err != nil {
		return err
	}
	if err := fn((word >> 18) & 63); err != nil {
		return err
	}
	if err := fn((word >> 24) & 63); err != nil {
		return err
	}
	if err := fn((word >> 30) & 63); err != nil {
		return err
	}
	if err := fn((word >> 36) & 63); err != nil {
		return err
	}
	if err := fn((word >> 42) & 63); err != nil {
		return err
	}
	if err := fn((word >> 48) & 63); err != nil {
		return err
	}
	return fn((word >> 54) & 63)
}

// selector 8: 8 × 7 bits

func pack8(src []uint64, pos int) uint64 {
	s := src[pos : pos+8 : pos+8]
	return (8 << 60) |
		s[0] | s[1]<<7 | s[2]<<14 | s[3]<<21 |
		s[4]<<28 | s[5]<<35 | s[6]<<42 | s[7]<<49
}

func unpack8(word uint64, dst []uint64) []uint64 {
	return append(dst,
		word&127, (word>>7)&127, (word>>14)&127, (word>>21)&127,
		(word>>28)&127, (word>>35)&127, (word>>42)&127, (word>>49)&127,
	)
}

func unpackFunc8(word uint64, fn func(uint64) error) error {
	if err := fn(word & 127); err != nil {
		return err
	}
	if err := fn((word >> 7) & 127); err != nil {
		return err
	}
	if err := fn((word >> 14) & 127); err != nil {
		return err
	}
	if err := fn((word >> 21) & 127); err != nil {
		return err
	}
	if err := fn((word >> 28) & 127); err != nil {
		return err
	}
	if err := fn((word >> 35) & 127); err != nil {
		return err
	}
	if err := fn((word >> 42) & 127); err != nil {
		return err
	}
	return fn((word >> 49) & 127)
}

// selector 9: 7 × 8 bits

func pack7(src []uint64, pos int) uint64 {
	s := src[pos : pos+7 : pos+7]
	return (9 << 60) |
		s[0] | s[1]<<8 | s[2]<<16 | s[3]<<24 |
		s[4]<<32 | s[5]<<40 | s[6]<<48
}

func unpack7(word uint64, dst []uint64) []uint64 {
	return append(dst,
		word&255, (word>>8)&255, (word>>16)&255, (word>>24)&255,
		(word>>32)&255, (word>>40)&255, (word>>48)&255,
	)
}

func unpackFunc7(word uint64, fn func(uint64) error) error {
	if err := fn(word & 255); err != nil {
		return err
	}
	if err := fn((word >> 8) & 255); err != nil {
		return err
	}
	if err := fn((word >> 16) & 255); err != nil {
		return err
	}
	if err := fn((word >> 24) & 255); err != nil {
		return err
	}
	if err := fn((word >> 32) & 255); err != nil {
		return err
	}
	if err := fn((word >> 40) & 255); err != nil {
		return err
	}
	return fn((word >> 48) & 255)
}

// selector 10: 6 × 10 bits

func pack6(src []uint64, pos int) uint64 {
	s := src[pos : pos+6 : pos+6]
	return (10 << 60) |
		s[0] | s[1]<<10 | s[2]<<20 | s[3]<<30 | s[4]<<40 | s[5]<<50
}

func unpack6(word uint64, dst []uint64) []uint64 {
	return append(dst,
		word&1023, (word>>10)&1023, (word>>20)&1023,
		(word>>30)&1023, (word>>40)&1023, (word>>50)&1023,
	)
}

func unpackFunc6(word uint64, fn func(uint64) error) error {
	if err := fn(word & 1023); err != nil {
		return err
	}
	if err := fn((word >> 10) & 1023); err != nil {
		return err
	}
	if err := fn((word >> 20) & 1023); err != nil {
		return err
	}
	if err := fn((word >> 30) & 1023); err != nil {
		return err
	}
	if err := fn((word >> 40) & 1023); err != nil {
		return err
	}
	return fn((word >> 50) & 1023)
}

// selector 11: 5 × 12 bits

func pack5(src []uint64, pos int) uint64 {
	s := src[pos : pos+5 : pos+5]
	return (11 << 60) |
		s[0] | s[1]<<12 | s[2]<<24 | s[3]<<36 | s[4]<<48
}

func unpack5(word uint64, dst []uint64) []uint64 {
	return append(dst,
		word&4095, (word>>12)&4095, (word>>24)&4095,
		(word>>36)&4095, (word>>48)&4095,
	)
}

func unpackFunc5(word uint64, fn func(uint64) error) error {
	if err := fn(word & 4095); err != nil {
		return err
	}
	if err := fn((word >> 12) & 4095); err != nil {
		return err
	}
	if err := fn((word >> 24) & 4095); err != nil {
		return err
	}
	if err := fn((word >> 36) & 4095); err != nil {
		return err
	}
	return fn((word >> 48) & 4095)
}

// selector 12: 4 × 15 bits

func pack4(src []uint64, pos int) uint64 {
	s := src[pos : pos+4 : pos+4]
	return (12 << 60) |
		s[0] | s[1]<<15 | s[2]<<30 | s[3]<<45
}

func unpack4(word uint64, dst []uint64) []uint64 {
	return append(dst,
		word&32767, (word>>15)&32767, (word>>30)&32767, (word>>45)&32767,
	)
}

func unpackFunc4(word uint64, fn func(uint64) error) error {
	if err := fn(word & 32767); err != nil {
		return err
	}
	if err := fn((word >> 15) & 32767); err != nil {
		return err
	}
	if err := fn((word >> 30) & 32767); err != nil {
		return err
	}
	return fn((word >> 45) & 32767)
}

// selector 13: 3 × 20 bits

func pack3(src []uint64, pos int) uint64 {
	s := src[pos : pos+3 : pos+3]
	return (13 << 60) | s[0] | s[1]<<20 | s[2]<<40
}

func unpack3(word uint64, dst []uint64) []uint64 {
	return append(dst, word&0xFFFFF, (word>>20)&0xFFFFF, (word>>40)&0xFFFFF)
}

func unpackFunc3(word uint64, fn func(uint64) error) error {
	if err := fn(word & 0xFFFFF); err != nil {
		return err
	}
	if err := fn((word >> 20) & 0xFFFFF); err != nil {
		return err
	}
	return fn((word >> 40) & 0xFFFFF)
}

// selector 14: 2 × 30 bits

func pack2(src []uint64, pos int) uint64 {
	s := src[pos : pos+2 : pos+2]
	return (14 << 60) | s[0] | s[1]<<30
}

func unpack2(word uint64, dst []uint64) []uint64 {
	return append(dst, word&0x3FFFFFFF, (word>>30)&0x3FFFFFFF)
}

func unpackFunc2(word uint64, fn func(uint64) error) error {
	if err := fn(word & 0x3FFFFFFF); err != nil {
		return err
	}
	return fn((word >> 30) & 0x3FFFFFFF)
}

// selector 15: 1 × 60 bits

func pack1(src []uint64, pos int) uint64 {
	return (15 << 60) | src[pos]
}

func unpack1(word uint64, dst []uint64) []uint64 {
	return append(dst, word&0x0FFFFFFFFFFFFFFF)
}

func unpackFunc1(word uint64, fn func(uint64) error) error {
	return fn(word & 0x0FFFFFFFFFFFFFFF)
}
