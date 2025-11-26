// FILE: github.com/josephcopenhaver/base32/base32.go

// A case insensitive Crockford style base32 implementation.

// This base32 decoding implementation rejects inputs that contain non-canonical
// tail bits that are non-zero. Other implementations may ignore them as useless
// noise but this algorithm strictly interprets them as a signal to fail decoding.
// If you are bit packing at a higher level to utilize these empty bits you are
// required to clear them before passing bytes to these functions. It is unsafe to
// assume the contents are noise as it could indicate a failure to preserve the
// encoded value full length. If there was concrete length metadata present as
// part of the standard decoding process I would feel differently but I leave that
// up to the caller to implement as they choose and to clear the tail bits as
// needed.

package base32

import (
	"errors"
	"slices"
	"unsafe"
)

const (
	b32Invalid = 0xFF
	b32Chars   = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
	b32UpToLow = ('a' - 'A')

	// Only these remainders are possible for valid un-padded base32:
	// 0, 2, 4, 5, 7. Others imply bad input.

	validDecodeRemainder = uint8((1 << 0) | (1 << 2) | (1 << 4) | (1 << 5) | (1 << 7))
)

var (
	ErrInvalidBase32Length = errors.New("invalid base32 length")
	ErrInvalidBase32Char   = errors.New("invalid base32 character")
)

//
// encode and decode tables are using Crockford style case insensitive grammars
//

var b32Encode, b32Decode = func() ([32]byte, [256]byte) {
	var enc [32]byte
	var dec [256]byte

	for i := range dec {
		dec[i] = b32Invalid
	}

	upLetter := func(v, i byte) {
		dec[v] = i
		dec[v+b32UpToLow] = i
	}

	for i := range b32Chars {
		i := byte(i)
		v := b32Chars[i]

		enc[i] = v
		if v > '9' {
			upLetter(v, i)
			continue
		}

		dec[v] = i
	}

	// char aliases
	upLetter('O', dec['0'])
	upLetter('I', dec['1'])
	upLetter('L', dec['1'])

	return enc, dec
}()

func b32EncodedLen(n int) int {
	result := (n/5)*8 + ((n%5)*8+4)/5
	if result <= n {
		panic("base32: invalid encode source length")
	}

	return result
}

// b32DecodedLen returns the base32 encoded length of
// base32 bytes with the provided length.
//
// If the input is zero the output will be zero. It is up
// to the calling context to choose how to handle the zero
// output case appropriately.
//
// If the input is invalid then -1 will be returned.
//
// invariants:
//
// - n must not be negative
func b32DecodedLen(n int) int {
	rem := n % 8

	if (validDecodeRemainder & (uint8(1) << rem)) == 0 {
		return -1
	}

	// decodedLen = floor(5*n/8) for valid lengths
	decodedLen := (n / 8) * 5
	switch rem {
	case 2:
		decodedLen += 1
	case 4:
		decodedLen += 2
	case 5:
		decodedLen += 3
	case 7:
		decodedLen += 4
	}

	return decodedLen
}

func encode(dst []byte, src []byte) {
	n := len(src)

	srcPtr := unsafe.Pointer(&src[0])
	dstPtr := unsafe.Pointer(&dst[0])

	for range n / 5 {
		b0 := *(*byte)(srcPtr)
		b1 := *(*byte)(unsafe.Add(srcPtr, 1))
		b2 := *(*byte)(unsafe.Add(srcPtr, 2))
		b3 := *(*byte)(unsafe.Add(srcPtr, 3))
		b4 := *(*byte)(unsafe.Add(srcPtr, 4))

		*(*byte)(dstPtr) = b32Encode[b0>>3]
		*(*byte)(unsafe.Add(dstPtr, 1)) = b32Encode[((b0<<2)|(b1>>6))&31]
		*(*byte)(unsafe.Add(dstPtr, 2)) = b32Encode[(b1>>1)&31]
		*(*byte)(unsafe.Add(dstPtr, 3)) = b32Encode[((b1<<4)|(b2>>4))&31]
		*(*byte)(unsafe.Add(dstPtr, 4)) = b32Encode[((b2<<1)|(b3>>7))&31]
		*(*byte)(unsafe.Add(dstPtr, 5)) = b32Encode[(b3>>2)&31]
		*(*byte)(unsafe.Add(dstPtr, 6)) = b32Encode[((b3<<3)|(b4>>5))&31]
		*(*byte)(unsafe.Add(dstPtr, 7)) = b32Encode[b4&31]

		srcPtr = unsafe.Add(srcPtr, 5)
		dstPtr = unsafe.Add(dstPtr, 8)
	}

	// Tail (no padding).
	switch n % 5 {
	case 1:
		b0 := *(*byte)(srcPtr)

		*(*byte)(dstPtr) = b32Encode[b0>>3]
		*(*byte)(unsafe.Add(dstPtr, 1)) = b32Encode[(b0<<2)&31]
	case 2:
		b0 := *(*byte)(srcPtr)
		b1 := *(*byte)(unsafe.Add(srcPtr, 1))

		*(*byte)(dstPtr) = b32Encode[b0>>3]
		*(*byte)(unsafe.Add(dstPtr, 1)) = b32Encode[((b0<<2)|(b1>>6))&31]
		*(*byte)(unsafe.Add(dstPtr, 2)) = b32Encode[(b1>>1)&31]
		*(*byte)(unsafe.Add(dstPtr, 3)) = b32Encode[(b1<<4)&31]
	case 3:
		b0 := *(*byte)(srcPtr)
		b1 := *(*byte)(unsafe.Add(srcPtr, 1))
		b2 := *(*byte)(unsafe.Add(srcPtr, 2))

		*(*byte)(dstPtr) = b32Encode[b0>>3]
		*(*byte)(unsafe.Add(dstPtr, 1)) = b32Encode[((b0<<2)|(b1>>6))&31]
		*(*byte)(unsafe.Add(dstPtr, 2)) = b32Encode[(b1>>1)&31]
		*(*byte)(unsafe.Add(dstPtr, 3)) = b32Encode[((b1<<4)|(b2>>4))&31]
		*(*byte)(unsafe.Add(dstPtr, 4)) = b32Encode[(b2<<1)&31]
	case 4:
		b0 := *(*byte)(srcPtr)
		b1 := *(*byte)(unsafe.Add(srcPtr, 1))
		b2 := *(*byte)(unsafe.Add(srcPtr, 2))
		b3 := *(*byte)(unsafe.Add(srcPtr, 3))

		*(*byte)(dstPtr) = b32Encode[b0>>3]
		*(*byte)(unsafe.Add(dstPtr, 1)) = b32Encode[((b0<<2)|(b1>>6))&31]
		*(*byte)(unsafe.Add(dstPtr, 2)) = b32Encode[(b1>>1)&31]
		*(*byte)(unsafe.Add(dstPtr, 3)) = b32Encode[((b1<<4)|(b2>>4))&31]
		*(*byte)(unsafe.Add(dstPtr, 4)) = b32Encode[((b2<<1)|(b3>>7))&31]
		*(*byte)(unsafe.Add(dstPtr, 5)) = b32Encode[(b3>>2)&31]
		*(*byte)(unsafe.Add(dstPtr, 6)) = b32Encode[(b3<<3)&31]
	}
}

// invariants:
//
// - len(src) > 0
//
// - len(dst) >= b32EncodedLen(len(src))
func UnsafeEncode(dst []byte, src []byte) {
	// guard statements forcing panics rather than letting next call
	// lead to undefined behaviors

	if n := b32EncodedLen(len(src)); len(dst) < n {
		panic("base32: encode destination too short")
	}

	encode(dst, src)
}

func Encode(src []byte) []byte {
	n := len(src)
	if n == 0 {
		return nil
	}

	n = b32EncodedLen(n)
	dst := make([]byte, n)

	encode(dst, src)

	return dst
}

func AppendEncode(dst, src []byte) []byte {
	n := len(src)
	if n == 0 {
		return dst
	}

	n = b32EncodedLen(n)
	orig := len(dst)

	dst = slices.Grow(dst, n)
	dst = dst[:orig+n]

	encode(dst[orig:], src)

	return dst
}

func decode(dst []byte, src []byte) error {
	n := len(src)

	srcPtr := unsafe.Pointer(&src[0])
	dstPtr := unsafe.Pointer(&dst[0])

	for range n / 8 {
		c0 := b32Decode[*(*byte)(srcPtr)]
		c1 := b32Decode[*(*byte)(unsafe.Add(srcPtr, 1))]
		c2 := b32Decode[*(*byte)(unsafe.Add(srcPtr, 2))]
		c3 := b32Decode[*(*byte)(unsafe.Add(srcPtr, 3))]
		c4 := b32Decode[*(*byte)(unsafe.Add(srcPtr, 4))]
		c5 := b32Decode[*(*byte)(unsafe.Add(srcPtr, 5))]
		c6 := b32Decode[*(*byte)(unsafe.Add(srcPtr, 6))]
		c7 := b32Decode[*(*byte)(unsafe.Add(srcPtr, 7))]

		if (c0 | c1 | c2 | c3 | c4 | c5 | c6 | c7) == b32Invalid {
			return ErrInvalidBase32Char
		}

		*(*byte)(dstPtr) = (c0<<3 | c1>>2)
		*(*byte)(unsafe.Add(dstPtr, 1)) = ((c1&0x03)<<6 | c2<<1 | c3>>4)
		*(*byte)(unsafe.Add(dstPtr, 2)) = ((c3&0x0F)<<4 | c4>>1)
		*(*byte)(unsafe.Add(dstPtr, 3)) = ((c4&0x01)<<7 | c5<<2 | c6>>3)
		*(*byte)(unsafe.Add(dstPtr, 4)) = ((c6&0x07)<<5 | c7)

		srcPtr = unsafe.Add(srcPtr, 8)
		dstPtr = unsafe.Add(dstPtr, 5)
	}

	// Tail.
	switch n % 8 {
	case 2:
		c0 := b32Decode[*(*byte)(srcPtr)]
		c1 := b32Decode[*(*byte)(unsafe.Add(srcPtr, 1))]

		// last 2 LSBs of last decoded value must be zero for remainder=2
		if (c0|c1) == b32Invalid || (c1&0x03) != 0 {
			return ErrInvalidBase32Char
		}

		*(*byte)(dstPtr) = (c0<<3 | c1>>2)
	case 4:
		c0 := b32Decode[*(*byte)(srcPtr)]
		c1 := b32Decode[*(*byte)(unsafe.Add(srcPtr, 1))]
		c2 := b32Decode[*(*byte)(unsafe.Add(srcPtr, 2))]
		c3 := b32Decode[*(*byte)(unsafe.Add(srcPtr, 3))]

		// last 4 LSBs of last decoded value must be zero for remainder=4
		if (c0|c1|c2|c3) == b32Invalid || (c3&0x0F) != 0 {
			return ErrInvalidBase32Char
		}

		*(*byte)(dstPtr) = (c0<<3 | c1>>2)
		*(*byte)(unsafe.Add(dstPtr, 1)) = ((c1&3)<<6 | c2<<1 | c3>>4)
	case 5:
		c0 := b32Decode[*(*byte)(srcPtr)]
		c1 := b32Decode[*(*byte)(unsafe.Add(srcPtr, 1))]
		c2 := b32Decode[*(*byte)(unsafe.Add(srcPtr, 2))]
		c3 := b32Decode[*(*byte)(unsafe.Add(srcPtr, 3))]
		c4 := b32Decode[*(*byte)(unsafe.Add(srcPtr, 4))]

		// last 1 LSB of last decoded value must be zero for remainder=5
		if (c0|c1|c2|c3|c4) == b32Invalid || (c4&0x01) != 0 {
			return ErrInvalidBase32Char
		}

		*(*byte)(dstPtr) = (c0<<3 | c1>>2)
		*(*byte)(unsafe.Add(dstPtr, 1)) = ((c1&0x03)<<6 | c2<<1 | c3>>4)
		*(*byte)(unsafe.Add(dstPtr, 2)) = ((c3&0x0F)<<4 | c4>>1)
	case 7:
		c0 := b32Decode[*(*byte)(srcPtr)]
		c1 := b32Decode[*(*byte)(unsafe.Add(srcPtr, 1))]
		c2 := b32Decode[*(*byte)(unsafe.Add(srcPtr, 2))]
		c3 := b32Decode[*(*byte)(unsafe.Add(srcPtr, 3))]
		c4 := b32Decode[*(*byte)(unsafe.Add(srcPtr, 4))]
		c5 := b32Decode[*(*byte)(unsafe.Add(srcPtr, 5))]
		c6 := b32Decode[*(*byte)(unsafe.Add(srcPtr, 6))]

		// last 3 LSBs of last decoded value must be zero for remainder=7
		if (c0|c1|c2|c3|c4|c5|c6) == b32Invalid || (c6&0x07) != 0 {
			return ErrInvalidBase32Char
		}

		*(*byte)(dstPtr) = (c0<<3 | c1>>2)
		*(*byte)(unsafe.Add(dstPtr, 1)) = ((c1&0x03)<<6 | c2<<1 | c3>>4)
		*(*byte)(unsafe.Add(dstPtr, 2)) = ((c3&0x0F)<<4 | c4>>1)
		*(*byte)(unsafe.Add(dstPtr, 3)) = ((c4&0x01)<<7 | c5<<2 | c6>>3)
	}

	return nil
}

// invariants:
//
// - len(src) > 0
//
// - len(dst) >=  b32DecodedLen(len(src))
//
// - len(src) is a valid base32 encoded value length
func UnsafeDecode(dst []byte, src []byte) error {
	// guard statements forcing panics rather than letting next call
	// lead to undefined behaviors

	if n := b32DecodedLen(len(src)); n <= 0 {
		panic("base32: invalid decode source length")
	} else if len(dst) < n {
		panic("base32: decode destination too short")
	}

	return decode(dst, src)
}

func Decode(src []byte) ([]byte, error) {
	n := len(src)
	if n == 0 {
		return nil, nil
	}

	n = b32DecodedLen(n)
	if n < 0 {
		return nil, ErrInvalidBase32Length
	}

	dst := make([]byte, n)

	err := decode(dst, src)

	return dst, err
}

func AppendDecode(dst, src []byte) ([]byte, error) {
	n := len(src)
	if n == 0 {
		return dst, nil
	}

	n = b32DecodedLen(n)
	if n <= 0 {
		return nil, ErrInvalidBase32Length
	}
	orig := len(dst)

	dst = slices.Grow(dst, n)
	dst = dst[:orig+n]

	err := decode(dst[orig:], src)

	return dst, err
}
