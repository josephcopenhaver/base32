# base32

[![Go Report Card](https://goreportcard.com/badge/github.com/josephcopenhaver/base32)](https://goreportcard.com/report/github.com/josephcopenhaver/base32)
![tests](https://github.com/josephcopenhaver/base32/actions/workflows/tests.yaml/badge.svg)
![code-coverage](https://img.shields.io/badge/code_coverage-100%25-rgb%2852%2C208%2C88%29)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)


A fast, allocation-conscious, case-insensitive Crockford-style Base32 implementation for Go.

---

## Features

- Compact API centered around `[]byte` and `string`
- No padding characters (`=`) – output length is determined by input length
- Strict decoder:
  - validates encoded length
  - rejects invalid characters
  - rejects non-canonical, non-zero tail bits in the final symbol group
- Separate:
  - safe, allocating helpers (`Encode`, `Decode`, etc.)
  - append helpers that reuse buffers
  - `Unsafe*` helpers for pre-validated hot paths
- Designed to be efficient and friendly to high-throughput code

---

## Installation

```bash
go get github.com/josephcopenhaver/base32@latest
```

Requires Go **1.22+** (uses `for range <integer>` and `unsafe.StringData`).

---

## Quick start

```go
package main

import (
	"fmt"
	"log"

	"github.com/josephcopenhaver/base32"
)

func main() {
	// Encode a byte slice.
	src := []byte("hello, base32")
	enc := base32.Encode(src)
	fmt.Println("encoded:", string(enc))

	// Encode a string directly.
	encStr := base32.EncodeString("hello, base32")
	fmt.Println("encoded string:", encStr)

	// Decode from []byte.
	dec, err := base32.Decode(enc)
	if err != nil {
		log.Fatalf("decode failed: %v", err)
	}
	fmt.Println("decoded:", string(dec))

	// Decode from string.
	dec2, err := base32.DecodeString(encStr)
	if err != nil {
		log.Fatalf("decode string failed: %v", err)
	}
	fmt.Println("decoded string:", string(dec2))
}
```

### Empty input behavior

- `Encode` → returns `nil` when `len(src) == 0`.
- `EncodeString` → returns `""` when `len(src) == 0`.
- `Decode` / `DecodeString` → return `(nil, nil)` when `len(src) == 0`.
- `Append*` functions return `dst` unchanged when `len(src) == 0`.

This makes "no input → no output" easy to handle without special cases.

---

## API overview

### Length helpers

```go
func EncodedLength(n int) int
func DecodedLength(n int) int
```

- `EncodedLength(n)`
  Returns the number of bytes required to encode `n` input bytes.
  Returns `-1` if `n` cannot produce a valid Base32 encoding for this scheme.

- `DecodedLength(n)`
  Returns the number of bytes that would be produced by decoding an encoded
  value of length `n`.
  Returns `-1` if `n` cannot be a valid encoded length.

Examples:

```go
// Planning an encode into a pre-allocated buffer.
n := base32.EncodedLength(len(src))
if n < 0 {
	return fmt.Errorf("cannot encode length %d", len(src))
}
dst := make([]byte, n)
base32.UnsafeEncode(dst, src)
```

```go
// Planning a decode into a pre-allocated buffer.
n := base32.DecodedLength(len(enc))
if n < 0 {
	return fmt.Errorf("invalid encoded length %d", len(enc))
}
dst := make([]byte, n)
if err := base32.UnsafeDecode(dst, enc); err != nil {
	return err
}
```

---

### Safe encode / decode

#### Encode

```go
func Encode(src []byte) []byte
func EncodeString(src string) string
```

- Allocate a new destination slice / string.
- Return the encoded representation.
- For zero-length input, they return `nil` or `""` respectively.

#### Decode

```go
var (
	ErrInvalidBase32Length = errors.New("invalid base32 length")
	ErrInvalidBase32Char   = errors.New("invalid base32 character")
)

func Decode(src []byte) ([]byte, error)
func DecodeString(src string) ([]byte, error)
```

- Allocate a new destination slice.
- Validate input length (structure) and characters.
- Enforce canonical tail bits — non-zero tail bits cause `ErrInvalidBase32Char`.

On error:

- `Decode` / `DecodeString` return a non-nil `error`.
- The returned `[]byte` may be **non-nil and partially filled**; do not assume it
  is empty or safe to reuse without clearing if the contents are sensitive.

---

### Append helpers

```go
func AppendEncode(dst, src []byte) []byte
func AppendEncodeString(dst []byte, src string) []byte

func AppendDecode(dst, src []byte) ([]byte, error)
func AppendDecodeString(dst []byte, src string) ([]byte, error)
```

- Encode/decode and append results to `dst`, using `slices.Grow` under the hood.
- For zero-length `src`, they return `dst` as-is (`AppendDecode*` also return `nil` error).

Example:

```go
buf := make([]byte, 0, base32.EncodedLength(len(src)))
buf = base32.AppendEncode(buf, src)

// ... later, perhaps reuse the same buffer for decoding into a fresh slice:
decoded, err := base32.AppendDecode(nil, buf)
if err != nil {
	// If the data is sensitive, consider clearing any appended region.
	return err
}
_ = decoded
```

Error handling for append decoders:

- On error, the returned slice may contain newly appended bytes that are partially
  decoded. If this matters (e.g., secrets), clear or discard the appended region.

---

### Unsafe helpers

```go
func UnsafeEncode(dst []byte, src []byte)
func UnsafeDecode(dst []byte, src []byte) error
```

Low-level, allocation-free helpers for hot paths with pre-validated sizes:

#### `UnsafeEncode`

- Preconditions:
  - `len(src) > 0`
  - `len(dst) >= EncodedLength(len(src))` (and that length is valid)
- Panics if:
  - `len(src) == 0`, or
  - `len(dst)` is too small.

```go
const idSize = 16
const encIDSize = (idSize/5)*8 + ((idSize%5)*8+4)/5 // a.k.a. 26; a.k.a. verified-valid output of EncodedLength(16)
var id [idSize]byte
var enc [encIDSize]byte

base32.UnsafeEncode(enc[:], id[:])
```

#### `UnsafeDecode`

- Preconditions:
  - `len(src) > 0`
  - `DecodedLength(len(src)) > 0`
  - `len(dst) >= DecodedLength(len(src))`
- Panics if:
  - `len(src)` is not a structurally valid encoded length, or
  - `len(dst)` is too small.

- Returns an `error` if:
  - Any character is invalid (`ErrInvalidBase32Char`), or
  - Tail bits are non-canonical / non-zero.

When `UnsafeDecode` returns a non-nil error, the contents of `dst` are
unspecified and may be partially decoded.

---

## Decoding strictness

This implementation **intentionally rejects**:

- Encoded lengths that are impossible for this Base32 scheme.
- Any character not in the supported alphabet.
- Encodings where the trailing, unused bits in the last symbol group are non-zero.

The last rule means this decoder will flag cases where:

- The encoded string is truncated.
- The extra bits were used for higher-level bit-packing but not cleared.
- There is no other length metadata, and those bits may indicate corruption.

If you are using the tail bits for your own higher-level scheme, you **must**
clear them before calling these functions.

---

## Concurrency

The package does not use mutable global state. All exported functions are safe
to call from multiple goroutines concurrently.

---

[![Go Reference](https://pkg.go.dev/badge/github.com/josephcopenhaver/base32.svg)](https://pkg.go.dev/github.com/josephcopenhaver/base32)
