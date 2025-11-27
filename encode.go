// FILE: github.com/josephcopenhaver/base32/encode.go

package base32

import (
	"slices"
	"unsafe"
)

func encodedLen(n int) int {
	result := (n/5)*8 + ((n%5)*8+4)/5
	if result <= n {
		panic("base32: invalid encode source length")
	}

	return result
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

		*(*byte)(dstPtr) = encodeTab[b0>>3]
		*(*byte)(unsafe.Add(dstPtr, 1)) = encodeTab[((b0<<2)|(b1>>6))&31]
		*(*byte)(unsafe.Add(dstPtr, 2)) = encodeTab[(b1>>1)&31]
		*(*byte)(unsafe.Add(dstPtr, 3)) = encodeTab[((b1<<4)|(b2>>4))&31]
		*(*byte)(unsafe.Add(dstPtr, 4)) = encodeTab[((b2<<1)|(b3>>7))&31]
		*(*byte)(unsafe.Add(dstPtr, 5)) = encodeTab[(b3>>2)&31]
		*(*byte)(unsafe.Add(dstPtr, 6)) = encodeTab[((b3<<3)|(b4>>5))&31]
		*(*byte)(unsafe.Add(dstPtr, 7)) = encodeTab[b4&31]

		srcPtr = unsafe.Add(srcPtr, 5)
		dstPtr = unsafe.Add(dstPtr, 8)
	}

	// Tail (no padding).
	switch n % 5 {
	case 1:
		b0 := *(*byte)(srcPtr)

		*(*byte)(dstPtr) = encodeTab[b0>>3]
		*(*byte)(unsafe.Add(dstPtr, 1)) = encodeTab[(b0<<2)&31]
	case 2:
		b0 := *(*byte)(srcPtr)
		b1 := *(*byte)(unsafe.Add(srcPtr, 1))

		*(*byte)(dstPtr) = encodeTab[b0>>3]
		*(*byte)(unsafe.Add(dstPtr, 1)) = encodeTab[((b0<<2)|(b1>>6))&31]
		*(*byte)(unsafe.Add(dstPtr, 2)) = encodeTab[(b1>>1)&31]
		*(*byte)(unsafe.Add(dstPtr, 3)) = encodeTab[(b1<<4)&31]
	case 3:
		b0 := *(*byte)(srcPtr)
		b1 := *(*byte)(unsafe.Add(srcPtr, 1))
		b2 := *(*byte)(unsafe.Add(srcPtr, 2))

		*(*byte)(dstPtr) = encodeTab[b0>>3]
		*(*byte)(unsafe.Add(dstPtr, 1)) = encodeTab[((b0<<2)|(b1>>6))&31]
		*(*byte)(unsafe.Add(dstPtr, 2)) = encodeTab[(b1>>1)&31]
		*(*byte)(unsafe.Add(dstPtr, 3)) = encodeTab[((b1<<4)|(b2>>4))&31]
		*(*byte)(unsafe.Add(dstPtr, 4)) = encodeTab[(b2<<1)&31]
	case 4:
		b0 := *(*byte)(srcPtr)
		b1 := *(*byte)(unsafe.Add(srcPtr, 1))
		b2 := *(*byte)(unsafe.Add(srcPtr, 2))
		b3 := *(*byte)(unsafe.Add(srcPtr, 3))

		*(*byte)(dstPtr) = encodeTab[b0>>3]
		*(*byte)(unsafe.Add(dstPtr, 1)) = encodeTab[((b0<<2)|(b1>>6))&31]
		*(*byte)(unsafe.Add(dstPtr, 2)) = encodeTab[(b1>>1)&31]
		*(*byte)(unsafe.Add(dstPtr, 3)) = encodeTab[((b1<<4)|(b2>>4))&31]
		*(*byte)(unsafe.Add(dstPtr, 4)) = encodeTab[((b2<<1)|(b3>>7))&31]
		*(*byte)(unsafe.Add(dstPtr, 5)) = encodeTab[(b3>>2)&31]
		*(*byte)(unsafe.Add(dstPtr, 6)) = encodeTab[(b3<<3)&31]
	}
}

// UnsafeEncode fills dst with the encoded form of src.
//
// It should generally only be used when working with pre-validated
// sizes of data like in the case of data types with known byte-lengths.
//
// This function panics if the source is empty or if the destination
// does not have enough space in the slice for the encoded form of src.
//
// Knowing the length of the slice now occupied by the encoded form of src
// is the responsibility of the caller. It can easily be computed by the
// expression ` (n/5)*8 + ((n%5)*8+4)/5 ` where n is the length of src.
//
// invariants:
//
// - len(src) > 0
//
// - len(dst) >= encodedLen(len(src))
func UnsafeEncode(dst []byte, src []byte) {
	// guard statements forcing panics rather than letting next call
	// lead to undefined behaviors

	if n := encodedLen(len(src)); len(dst) < n {
		panic("base32: encode destination too short")
	}

	encode(dst, src)
}

// Encode returns nil if src is empty, otherwise it returns the
// encoded form of src.
func Encode(src []byte) []byte {
	n := len(src)
	if n == 0 {
		return nil
	}

	n = encodedLen(n)
	dst := make([]byte, n)

	encode(dst, src)

	return dst
}

// AppendEncode returns the encoded form of src appended to dst
// if src is not empty. If src is empty dst is returned as-is.
func AppendEncode(dst, src []byte) []byte {
	n := len(src)
	if n == 0 {
		return dst
	}

	n = encodedLen(n)
	orig := len(dst)

	dst = slices.Grow(dst, n)
	dst = dst[:orig+n]

	encode(dst[orig:], src)

	return dst
}
