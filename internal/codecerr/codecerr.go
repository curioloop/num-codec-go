// Package codecerr holds the concrete error instances returned by every
// codec sub-package. The root package re-exports them as its public
// ErrOverflow / ErrMalformed. Users compare via errors.Is against the
// root package's names; this internal package exists only to break the
// import cycle between root and sub-packages.
package codecerr

import "errors"

var (
	ErrOverflow  = errors.New("numcodec: value overflow")
	ErrMalformed = errors.New("numcodec: malformed data")
)
