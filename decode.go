// FILE: github.com/josephcopenhaver/base32/decode.go

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

	// Only these remainders are possible for valid un-padded base32:
	// 0, 2, 4, 5, 7. Others imply bad input.

	validDecodeRemainder = uint8((1 << 0) | (1 << 2) | (1 << 4) | (1 << 5) | (1 << 7))
)

var (
	ErrInvalidBase32Length = errors.New("invalid base32 length")
	ErrInvalidBase32Char   = errors.New("invalid base32 character")
)

// decodedLen returns the base32 encoded length of
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
func decodedLen(n int) int {
	rem := n % 8

	if (validDecodeRemainder & (uint8(1) << rem)) == 0 {
		return -1
	}

	return (n/8)*5 + (rem*5)/8
}

func decode(dst []byte, src []byte) error {
	n := len(src)

	srcPtr := unsafe.Pointer(&src[0])
	dstPtr := unsafe.Pointer(&dst[0])

	for range n / 8 {
		c0 := decodeTab[*(*byte)(srcPtr)]
		c1 := decodeTab[*(*byte)(unsafe.Add(srcPtr, 1))]
		c2 := decodeTab[*(*byte)(unsafe.Add(srcPtr, 2))]
		c3 := decodeTab[*(*byte)(unsafe.Add(srcPtr, 3))]
		c4 := decodeTab[*(*byte)(unsafe.Add(srcPtr, 4))]
		c5 := decodeTab[*(*byte)(unsafe.Add(srcPtr, 5))]
		c6 := decodeTab[*(*byte)(unsafe.Add(srcPtr, 6))]
		c7 := decodeTab[*(*byte)(unsafe.Add(srcPtr, 7))]

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
		c0 := decodeTab[*(*byte)(srcPtr)]
		c1 := decodeTab[*(*byte)(unsafe.Add(srcPtr, 1))]

		// last 2 LSBs of last decoded value must be zero for remainder=2
		if (c0|c1) == b32Invalid || (c1&0x03) != 0 {
			return ErrInvalidBase32Char
		}

		*(*byte)(dstPtr) = (c0<<3 | c1>>2)
	case 4:
		c0 := decodeTab[*(*byte)(srcPtr)]
		c1 := decodeTab[*(*byte)(unsafe.Add(srcPtr, 1))]
		c2 := decodeTab[*(*byte)(unsafe.Add(srcPtr, 2))]
		c3 := decodeTab[*(*byte)(unsafe.Add(srcPtr, 3))]

		// last 4 LSBs of last decoded value must be zero for remainder=4
		if (c0|c1|c2|c3) == b32Invalid || (c3&0x0F) != 0 {
			return ErrInvalidBase32Char
		}

		*(*byte)(dstPtr) = (c0<<3 | c1>>2)
		*(*byte)(unsafe.Add(dstPtr, 1)) = ((c1&3)<<6 | c2<<1 | c3>>4)
	case 5:
		c0 := decodeTab[*(*byte)(srcPtr)]
		c1 := decodeTab[*(*byte)(unsafe.Add(srcPtr, 1))]
		c2 := decodeTab[*(*byte)(unsafe.Add(srcPtr, 2))]
		c3 := decodeTab[*(*byte)(unsafe.Add(srcPtr, 3))]
		c4 := decodeTab[*(*byte)(unsafe.Add(srcPtr, 4))]

		// last 1 LSB of last decoded value must be zero for remainder=5
		if (c0|c1|c2|c3|c4) == b32Invalid || (c4&0x01) != 0 {
			return ErrInvalidBase32Char
		}

		*(*byte)(dstPtr) = (c0<<3 | c1>>2)
		*(*byte)(unsafe.Add(dstPtr, 1)) = ((c1&0x03)<<6 | c2<<1 | c3>>4)
		*(*byte)(unsafe.Add(dstPtr, 2)) = ((c3&0x0F)<<4 | c4>>1)
	case 7:
		c0 := decodeTab[*(*byte)(srcPtr)]
		c1 := decodeTab[*(*byte)(unsafe.Add(srcPtr, 1))]
		c2 := decodeTab[*(*byte)(unsafe.Add(srcPtr, 2))]
		c3 := decodeTab[*(*byte)(unsafe.Add(srcPtr, 3))]
		c4 := decodeTab[*(*byte)(unsafe.Add(srcPtr, 4))]
		c5 := decodeTab[*(*byte)(unsafe.Add(srcPtr, 5))]
		c6 := decodeTab[*(*byte)(unsafe.Add(srcPtr, 6))]

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

// UnsafeDecode decodes the source slice into the destination slice.
//
// It should generally only be used when working with pre-validated
// sizes of data like in the case of data types with known byte-lengths.
//
// This function panics if the source is empty or if the destination
// does not have enough space in the slice for the decoded form of src.
//
// It is the parent context's responsibility to clear the dst slice
// should an error be returned and that be the ideal rollback state.
//
// Knowing the length of the slice now occupied by the decoded form of src
// is the responsibility of the caller. It can easily be computed by the
// expression ` (n/8)*5 + ((n%8)*5)/8` where n is the length of src.
//
// invariants:
//
// - len(src) > 0
//
// - len(dst) >=  decodedLen(len(src))
//
// - len(src) is a valid base32 encoded value length
func UnsafeDecode(dst []byte, src []byte) error {
	// guard statements forcing panics rather than letting next call
	// lead to undefined behaviors

	if n := decodedLen(len(src)); n <= 0 {
		panic("base32: invalid decode source length")
	} else if len(dst) < n {
		panic("base32: decode destination too short")
	}

	return decode(dst, src)
}

// Decode returns the decoded form of src if src is not empty. If src is
// empty nil is returned.
//
// If an error occurs during decoding then an error will be returned.
//
// If an error is returned the caller must not assume the returned slice
// is nil. It is the caller's responsibility to choose how to handle a
// non-nil result in such a case. If the data is not sensitive simply
// ignore it. If it is sensitive consider clearing the slice of
// contents. There is no guarantee about the contents of the slice when a
// non-nil error is returned. It could be partially decoded or contain
// empty bytes.
func Decode(src []byte) ([]byte, error) {
	n := len(src)
	if n == 0 {
		return nil, nil
	}

	n = decodedLen(n)
	if n < 0 {
		return nil, ErrInvalidBase32Length
	}

	dst := make([]byte, n)

	err := decode(dst, src)
	return dst, err
}

// AppendDecode returns the decoded form of src appended to dst
// if src is not empty. If src is empty dst is returned as-is.
//
// If an error occurs during decoding then an error will be returned.
//
// If an error is returned the caller must not assume the returned slice
// is nil. It is the caller's responsibility to choose how to handle a
// non-nil result in such a case. If the data is not sensitive simply
// ignore it. If it is sensitive consider clearing the slice of
// newly appended contents. There is no guarantee about the contents of
// the appended slice when a non-nil error is returned. It could be
// partially decoded or contain empty bytes.
func AppendDecode(dst, src []byte) ([]byte, error) {
	n := len(src)
	if n == 0 {
		return dst, nil
	}

	n = decodedLen(n)
	if n < 0 {
		return nil, ErrInvalidBase32Length
	}
	orig := len(dst)

	dst = slices.Grow(dst, n)
	dst = dst[:orig+n]

	err := decode(dst[orig:], src)
	return dst, err
}
