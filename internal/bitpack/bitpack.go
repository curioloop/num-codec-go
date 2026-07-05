// Package bitpack provides shared bit-level I/O primitives used by the
// gorilla and chimp XOR codecs. It lives under internal/ because both
// consumers depend on byte-exact wire format compatibility with the
// reference Java number-codec library, and neither wants third parties
// building on this low-level surface directly.
package bitpack

import (
	"encoding/binary"

	"github.com/curioloop/num-codec-go/internal/codecerr"
)

// Reader reads sequential bits from an underlying byte slice window.
// The very last byte of the window records how many padding bits were
// added by the writer so the reader can stop at the exact bit boundary.
//
// Reader uses a sticky-error pattern (like bufio.Scanner): once an
// out-of-range read is attempted, Err returns ErrMalformed and subsequent
// reads return zero. Callers check Err once at the end of decoding instead
// of after every read.
type Reader struct {
	data      []byte
	totalBits int
	bitCursor int
	err       error
}

// Reset re-initialises r to read from data. data must include the trailing
// padding byte written by Writer.
func (r *Reader) Reset(data []byte) {
	r.data = data
	r.bitCursor = 0
	r.err = nil
	if len(data) < 1 {
		r.err = codecerr.ErrMalformed
		r.totalBits = 0
		return
	}
	r.totalBits = (len(data)-1)*8 - int(data[len(data)-1])
	if r.totalBits < 0 {
		r.err = codecerr.ErrMalformed
		r.totalBits = 0
	}
}

// Err returns the first malformed-input error encountered, if any.
func (r *Reader) Err() error { return r.err }

// HasMore reports whether unread bits remain and no error is set.
func (r *Reader) HasMore() bool { return r.err == nil && r.bitCursor < r.totalBits }

// ReadBit reads the next single bit. Returns false and sets Err if
// exhausted.
func (r *Reader) ReadBit() bool {
	if r.err != nil || r.bitCursor >= r.totalBits {
		r.err = codecerr.ErrMalformed
		return false
	}
	pos := r.bitCursor
	r.bitCursor++
	b := r.data[pos/8]
	return (b>>(7-uint(pos%8)))&0x1 > 0
}

// nextBits reads numOfBits from within the current byte (numOfBits <= 8).
func (r *Reader) nextBits(numOfBits int) uint32 {
	rest := 8 - r.bitCursor%8
	if rest < numOfBits {
		r.err = codecerr.ErrMalformed
		return 0
	}
	chunk := uint32(r.data[r.bitCursor/8])
	mask := uint32(0xFF) >> (8 - numOfBits)
	offset := rest - numOfBits
	r.bitCursor += numOfBits
	return (chunk >> offset) & mask
}

// ReadBits reads 1..64 bits and returns them right-aligned.
func (r *Reader) ReadBits(numOfBits int) uint64 {
	if r.err != nil {
		return 0
	}
	if numOfBits <= 0 || numOfBits > 64 || r.bitCursor+numOfBits > r.totalBits {
		r.err = codecerr.ErrMalformed
		return 0
	}
	rest := 8 - r.bitCursor%8
	if numOfBits <= rest {
		return uint64(r.nextBits(numOfBits))
	}
	v := uint64(r.nextBits(rest))
	numOfBits -= rest
	for numOfBits > 0 {
		n := numOfBits
		if n > 8 {
			n = 8
		}
		v <<= uint(n)
		v |= uint64(r.nextBits(n))
		numOfBits -= n
	}
	return v
}

// Writer accumulates bits into a 64-bit staging word and appends
// completed bytes to an underlying []byte. Flush must be called once at
// the end; the completed bytes are then available via Bytes.
//
// Zero value is ready to use (writes to an internal, nil-initialised
// buffer that will grow via append). Reset(dst) lets callers pass a
// preallocated destination slice.
type Writer struct {
	buf       []byte
	staging   uint64
	bufBits   int // bits currently held in `staging`; -1 after Flush
	totalBits int
}

// Reset re-initialises w to append into dst. dst may be nil or have spare
// capacity.
func (w *Writer) Reset(dst []byte) {
	w.buf = dst
	w.staging = 0
	w.bufBits = 0
	w.totalBits = 0
}

// Bytes returns the underlying byte slice.
func (w *Writer) Bytes() []byte { return w.buf }

// TotalBits returns the cumulative bit count written (excluding padding).
func (w *Writer) TotalBits() int { return w.totalBits }

// currentPos returns the bit index (63..0) at which the next bit will be
// placed inside the 64-bit staging word; -1 means full.
func (w *Writer) currentPos() int { return (64 - w.bufBits) - 1 }

func (w *Writer) flushStaging() {
	bufBytes := w.bufBits / 8
	if w.bufBits%8 != 0 {
		bufBytes++
	}
	if bufBytes == 8 {
		w.buf = binary.BigEndian.AppendUint64(w.buf, w.staging)
	} else {
		for i := 0; i < bufBytes; i++ {
			shift := uint((7 - i) * 8)
			w.buf = append(w.buf, byte((w.staging>>shift)&0xFF))
		}
	}
	w.bufBits = 0
	w.staging = 0
}

// WriteBit writes a single bit.
func (w *Writer) WriteBit(bit bool) {
	if bit {
		w.staging |= 1 << uint(w.currentPos())
	}
	w.bufBits++
	w.totalBits++
	if w.currentPos() < 0 {
		w.flushStaging()
	}
}

// WriteBits writes the num low-order bits of v. Panics on invalid num;
// this is a programming error, not runtime data corruption.
//
// Splits into at most two writes: if v spans past the staging boundary,
// the top `capacity` bits fill the current staging word (which then
// flushes) and the remaining `num - capacity` bits go into the fresh
// staging. Unrolled from a recursive pair of calls into a straight-line
// sequence so the common path avoids the function-call overhead.
func (w *Writer) WriteBits(v uint64, num int) {
	if num <= 0 || num > 64 {
		panic("bitpack: Writer.WriteBits num out of range")
	}
	capacity := 64 - w.bufBits
	if capacity < num {
		top := v >> uint(num-capacity)
		mask := ^uint64(0) >> uint(64-capacity)
		w.staging |= top & mask
		w.bufBits += capacity
		w.totalBits += capacity
		w.flushStaging()
		num -= capacity
		capacity = 64
	}
	var mask uint64
	if num == 64 {
		mask = ^uint64(0)
	} else {
		mask = ^uint64(0) >> uint(64-num)
	}
	w.staging |= (v & mask) << uint(capacity-num)
	w.bufBits += num
	w.totalBits += num
	if w.bufBits == 64 {
		w.flushStaging()
	}
}

// Flush flushes any remaining buffered bits and records a padding byte
// carrying the number of zero bits inserted to reach the byte boundary
// (0..7). Must only be called once.
func (w *Writer) Flush() {
	w.flushStaging()
	w.bufBits = -1
	w.buf = append(w.buf, byte((8-(w.totalBits%8))%8))
}
