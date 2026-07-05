package simple8

import "math/bits"

// numOfIntToSelector maps the "number of integers packed together" (index)
// to the selector index into `selectors`. Only entries at indices
// {1..8, 10, 12, 15, 20, 30, 60, 120, 240} are meaningful; other slots
// hold 0 and are never queried because getPacking is only called with
// valid counts produced by lookupPacking.
var numOfIntToSelector = func() [241]uint8 {
	var t [241]uint8
	for i, sel := range map[int]uint8{
		10: 7, 12: 6, 15: 5, 20: 4, 30: 3, 60: 2, 120: 1, 240: 0,
	} {
		t[i] = sel
	}
	return t
}()

// stateSpace is the precomputed FastLookup pruning table. It reduces Simple8b
// selection from O(n²) backtracking to O(n) table lookups. Layout matches
// FastLookup.java's static initializer.
var stateSpace [261]byte

func init() {
	beg, end := 0, 60
	for i := 1; i <= 60; i++ {
		if i != 1 {
			end += 60 / i
		}
		nStart := beg + 60/(i+1)
		for n := nStart; n < end; n++ {
			stateSpace[n] = byte(i)
		}
		beg = end
	}
}

// significantBits returns 1..64: the number of low-order bits needed to
// represent i (i==0 returns 1 to match Java semantics).
func significantBits(v uint64) int {
	if v == 0 {
		return 1
	}
	return 64 - bits.LeadingZeros64(v)
}

// getPacking returns the packing scheme that packs numOfInt integers.
// Assumes numOfInt is one of the valid counts.
func getPacking(numOfInt int) *packing {
	if numOfInt >= 1 && numOfInt <= 8 {
		return &selectors[16-numOfInt]
	}
	return &selectors[numOfIntToSelector[numOfInt]]
}

// lookupPacking returns the optimal packing for the next chunk of values
// in src[pos:]. Returns nil when some value in the current chunk exceeds
// 60 bits (i.e. cannot be Simple8b-packed at all — caller must overflow).
func lookupPacking(src []uint64, pos int) *packing {
	length := len(src)
	remain := length - pos
	num := remain
	if num > 60 {
		num = 60
	}

	pruning := 0
	indicator := src[pos] // used to detect the "all values identical" case
	beg, end, match := 0, 60, 1
	for i := 1; i <= num; i++ {
		if i != 1 {
			end += 60 / i
		}
		var value uint64
		if i == 1 {
			value = indicator
		} else {
			value = src[pos+i-1]
		}
		if value >= (1 << 60) {
			return nil
		}
		if indicator != value {
			indicator = ^uint64(0) // sentinel: not all-identical
		}

		n := significantBits(value) - 1
		if n > pruning {
			pruning = n
		}
		n = pruning

		if beg+n >= end {
			return getPacking(match)
		}
		if i < 60 && stateSpace[beg+n] > 0 {
			return getPacking(int(stateSpace[beg+n]))
		}
		if stateSpace[end-1] > 0 {
			match = int(stateSpace[end-1])
		}
		beg = end
	}

	if num < 60 || remain < 120 || indicator == ^uint64(0) {
		return getPacking(match)
	}

	// All 60 values are equal; try to extend to 120 / 240 identical values.
	j := remain
	if j > 240 {
		j = 240
	}
	var i int
	for i = 60; i < j; i++ {
		if src[pos+i] != indicator {
			break
		}
	}
	if i == 240 {
		return getPacking(240)
	}
	if i >= 120 {
		return getPacking(120)
	}
	return getPacking(match)
}
