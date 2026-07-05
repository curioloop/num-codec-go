# number-codec-go

Go port of the Java [number-codec](https://github.com/curioloop/number-codec)
library — lossless numerical compression codecs specialised for time-series
data. Output is **byte-for-byte wire compatible** with the Java library, so
messages encoded on one side can be decoded on the other.

> Status: **v0.0.x — unstable**. The API is being polished; expect breaking
> changes on minor bumps until v0.1.0.

## Install

```
go get github.com/curioloop/num-codec-go
```

Requires Go 1.22+. No external dependencies.

## Quick start

```go
package main

import (
    "fmt"
    "github.com/curioloop/num-codec-go"
)

func main() {
    prices := []float64{100.01, 100.02, 100.02, 100.03, 100.05, 100.06}

    // Pass nil for dst to allocate; pass a preallocated []byte to append
    // into it (like binary.BigEndian.AppendUint64).
    data, flags, _ := numcodec.AppendEncodeFloat64s(nil, prices)
    fmt.Printf("compressed %d → %d bytes (flags=%s)\n",
        8*len(prices), len(data), flags)

    out, _ := numcodec.AppendDecodeFloat64s(nil, data, flags)
    fmt.Println(out) // [100.01 100.02 100.02 100.03 100.05 100.06]
}
```

Append style also lets callers reuse buffers across many calls:

```go
dst := make([]byte, 0, 4096)
for _, series := range everything {
    var flags numcodec.Flags
    dst, flags, _ = numcodec.AppendEncodeFloat64s(dst[:0], series)
    // ... use dst / flags ...
}
```

### Reusing scratch buffers across encode calls

For repeated encoding, hold on to an `Encoder`. It amortises the internal
working buffers (Chimp ring buffer, Simple8b staging) so the hot path is
zero-alloc:

```go
var enc numcodec.Encoder
for _, series := range everything {
    dst, flags, _ = enc.AppendEncodeFloat64s(dst[:0], series)
    // ...
}
enc.Reset() // optional; drops references but keeps capacity
```

Each `AppendEncode*` tries several codecs in priority order and picks the
smallest output. The returned `Flags` bitmask records which codec won;
pass it back to the matching `AppendDecode*`.

| Helper                                          | Try order                                                                            |
| ----------------------------------------------- | ------------------------------------------------------------------------------------ |
| `AppendEncodeInt32s`/`AppendEncodeInt64s`       | ZigZag+Simple8b → bare ZigZag → raw big-endian                                       |
| `AppendEncodeUint32s`/`AppendEncodeUint64s`     | Uvarint+Simple8b → bare Uvarint → raw big-endian                                     |
| `AppendEncodeFloat32s`/`AppendEncodeFloat64s`   | Gorilla → Chimp (N=32) → raw IEEE 754                                                |
| `AppendEncodeDelta2`                            | Delta + Simple8b (sorted) → Delta + ZigZag+Uvarint (unsorted)                        |

The `Decode` counterpart accepts a destination slice (nil to allocate)
and returns the extended slice. Malformed input returns `ErrMalformed`;
unrecognised flag bits return `ErrBadFlags`.

## Sub-packages

If you only need one codec, the per-algorithm packages have narrower APIs:

| Package                             | Algorithm                                                                                             |
| ----------------------------------- | ----------------------------------------------------------------------------------------------------- |
| [`varint`](varint/)                 | Uvarint (unsigned) and ZigZag (signed) variable-length integer encoding                               |
| [`simple8`](simple8/)               | [Simple8b](https://arxiv.org/pdf/1209.2137.pdf) bit-packing (up to 240 integers per 64-bit word)      |
| [`gorilla`](gorilla/)               | [Gorilla](http://www.vldb.org/pvldb/vol8/p1816-teller.pdf) XOR-based float / double compression       |
| [`chimp`](chimp/)                   | [Chimp](https://www.vldb.org/pvldb/vol15/p3058-liakos.pdf) — Gorilla with adaptive leading-zero table |
| [`delta2`](delta2/)                 | Delta-of-delta over Simple8b (sorted) or ZigZag+Uvarint (unsorted)                                    |

Sub-package highlights:

- **`chimp`** exposes a single knob `n int` selecting the Chimp variant:
  `n == 0` is base Chimp; `n ∈ {4,8,16,32,64,128,256}` is ChimpN with a
  ring buffer of size `n`. Use `chimp.Scratch` for zero-alloc reuse.
- **`simple8`** offers two unpack surfaces: `AppendUnpack` (into a
  caller slice) and `UnpackFunc` (callback, zero-alloc streaming).

Sentinel errors `ErrOverflow`, `ErrMalformed`, and `ErrBadFlags` are
exported from the root package. All entry points return normal Go `error`
values — no panic/recover control flow.

## Compression ratio on real market data

`go test -run TestCompressRateOnSamples -v` runs every column of the
sample datasets through the helpers and reports the ratio:

```
NBBO/kline:  raw=124800   compressed=54850    (44.0%)
NBBO/trade:  raw=14166928 compressed=3369345  (23.8%)
SEHK/kline:  raw=77720    compressed=27846    (35.8%)
SEHK/trade:  raw=1512096  compressed=326500   (21.6%)
```

Trade data (mostly timestamps and small integer sizes) compresses ~4-5×;
KLine data (mixed float columns) compresses ~2-3×.

## Benchmarks (Apple M4 Pro, 10 000-value workloads)

Encode (allocating dst each iteration):

```
BenchmarkEncodeDelta2-12               15591   14621 ns/op   81920 B/op    1 allocs/op
BenchmarkEncodeInt32s-12                4962   48702 ns/op  122883 B/op    2 allocs/op
BenchmarkEncodeUint32s-12               4864   48460 ns/op  122883 B/op    2 allocs/op
BenchmarkEncodeInt64s-12                4426   55955 ns/op  163845 B/op    2 allocs/op
BenchmarkEncodeUint64s-12               3925   60115 ns/op  163847 B/op    2 allocs/op
BenchmarkEncodeFloat32s-12              2562   92880 ns/op   40976 B/op    1 allocs/op
BenchmarkEncodeFloat64s-12              2410   99715 ns/op   81950 B/op    1 allocs/op
BenchmarkEncodeDelta2Unordered-12      18189   14058 ns/op   81921 B/op    1 allocs/op
```

Encoder-reuse (single `Encoder` amortised, dst reused):

```
BenchmarkEncoderEncodeDelta2-12        20582   11682 ns/op       3 B/op    0 allocs/op
BenchmarkEncoderEncodeInt32s-12         5244   45024 ns/op      26 B/op    0 allocs/op
BenchmarkEncoderEncodeUint32s-12        5221   44407 ns/op      26 B/op    0 allocs/op
BenchmarkEncoderEncodeInt64s-12         4714   48387 ns/op      39 B/op    0 allocs/op
BenchmarkEncoderEncodeFloat32s-12       2671   90199 ns/op      30 B/op    0 allocs/op
BenchmarkEncoderEncodeFloat64s-12       2560   95813 ns/op      60 B/op    0 allocs/op
```

Decode (append into caller slice):

```
BenchmarkDecodeDelta2-12               14936   16270 ns/op    4917 MB/s    0 allocs/op
BenchmarkDecodeDelta2Unordered-12       9123   24822 ns/op    3223 MB/s    0 allocs/op
BenchmarkDecodeInt32s-12                7080   33753 ns/op    1185 MB/s    0 allocs/op
BenchmarkDecodeUint32s-12               8905   26315 ns/op    1520 MB/s    0 allocs/op
BenchmarkDecodeInt64s-12                7203   32705 ns/op    2446 MB/s    0 allocs/op
BenchmarkDecodeUint64s-12               7366   30750 ns/op    2602 MB/s    0 allocs/op
BenchmarkDecodeFloat32s-12              2126  119035 ns/op     336 MB/s    0 allocs/op
BenchmarkDecodeFloat64s-12              1384  175747 ns/op     455 MB/s    0 allocs/op
```

Decode is zero-alloc across the board. Plain-call encode allocates one
working buffer (roughly sized to raw input); the `Encoder` type reuses
those buffers between calls.

## License

Apache-2.0, same as the upstream Java library.
