package numcodec

import (
	"errors"

	"github.com/curioloop/num-codec-go/internal/codecerr"
)

// Sentinel error values returned by every public Encode/Decode entry point.
// Compare with errors.Is.
var (
	// ErrOverflow means a value could not be represented in the current
	// codec (e.g. a Simple8b bucket cannot hold a 61-bit value, or a
	// VarInt-encoded value exceeded 8 bytes when packed into Simple8).
	// Encoders raise it to signal the caller may retry with a wider codec;
	// the top-level Append* helpers absorb it internally and fall back.
	ErrOverflow = codecerr.ErrOverflow

	// ErrMalformed indicates the byte stream does not form a valid codec
	// output (out-of-range selector, truncated bit stream, invalid varint).
	ErrMalformed = codecerr.ErrMalformed

	// ErrBadFlags means the Flags value passed to a decode function is
	// either zero, has no active codec bit, or contains an incompatible
	// combination for the decoder (e.g. Delta2 without Simple8 or ZigZag).
	// Signals a caller-side programmer error rather than corrupt data.
	ErrBadFlags = errors.New("numcodec: bad flags")
)
