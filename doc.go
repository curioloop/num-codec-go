// Package numcodec implements lossless numerical compression codecs
// specialised for time-series data. It is a Go port of the Java
// number-codec library (https://github.com/curioloop/number-codec),
// preserving byte-for-byte wire compatibility with the Java originals.
//
// # Overview
//
// The high-level API is slice-first and follows the AppendXxx convention
// (like [encoding/binary.BigEndian.AppendUint64]): every AppendEncode*
// takes a destination []byte and returns the extended slice plus a
// [Flags] value describing which internal codec was chosen. Every
// AppendDecode* takes a destination slice and returns it extended with
// the decoded values. Pass nil for dst to have the function allocate.
//
//	data, flags, _ := numcodec.AppendEncodeFloat64s(nil, prices)
//	out,  _        := numcodec.AppendDecodeFloat64s(nil, data, flags)
//
// # Codec selection
//
// Each AppendEncode* tries several codecs in priority order and picks
// the smallest output; the returned [Flags] identifies the winner:
//
//	AppendEncodeInt32s / Int64s   → ZigZag+Simple8b → bare ZigZag → Raw
//	AppendEncodeUint32s / Uint64s → Uvarint+Simple8b → bare Uvarint → Raw
//	AppendEncodeFloat32s / Float64s → Gorilla → Chimp (N=32) → Raw
//	AppendEncodeDelta2            → Delta2+Simple8b (sorted) → Delta2+ZigZag+Uvarint (unsorted)
//
// # When to use Encoder
//
// The package-level AppendEncode* functions instantiate a fresh
// zero-value [Encoder] per call. For repeat encoding of many series,
// hold onto an Encoder to amortise the internal scratch buffers to
// zero allocations:
//
//	var enc numcodec.Encoder
//	for _, series := range everything {
//	    data, flags, _ := enc.AppendEncodeFloat64s(dst[:0], series)
//	    // ...
//	}
//
// Decoding is inherently zero-alloc so there is no matching Decoder
// type — call the package-level AppendDecode* functions directly.
//
// # Errors
//
// All decode entry points return one of three sentinel errors:
//
//	ErrOverflow      — value out of range for the codec (encode fallback)
//	ErrMalformed     — corrupt or truncated encoded byte stream
//	ErrBadFlags  — decode called with a Flags value that no codec
//	                   understands (caller-side programmer error)
//
// Compare with [errors.Is].
//
// # Sub-packages
//
// Every codec is also available as a stand-alone sub-package for users
// who need direct access without going through the top-level try-many
// helper:
//
//	varint  — VarInt (unsigned) and ZigZag (signed) variable-length ints
//	simple8 — Simple8b bit-packing (up to 240 values per 64-bit word)
//	gorilla — Gorilla XOR-based float / double compression
//	chimp   — Chimp / ChimpN float / double compression
//	delta2  — Delta-of-delta over Simple8b or ZigZag+Uvarint
package numcodec
