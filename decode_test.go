package base32

import (
	"iter"
	"math"
	"slices"
	"testing"

	"github.com/josephcopenhaver/tbdd-go"
	"github.com/stretchr/testify/assert"
)

func Test_decodedLen(t *testing.T) {
	t.Parallel()

	is := assert.New(t)

	invalidRemainders := [8]bool{}
	invalidRemainders[1] = true
	invalidRemainders[3] = true
	invalidRemainders[6] = true

	for i := range uint8(8) {
		v := (math.MaxInt-7)/8*8 + int(i)

		// decodedLen
		{
			n := decodedLen(v)

			if invalidRemainders[i] {
				is.Equal(-1, n)
				continue
			}

			is.NotEqual(-1, n)
			is.Greater(n, 0)
		}

		// DecodedLength
		{
			n := DecodedLength(v)

			if invalidRemainders[i] {
				is.Equal(-1, n)
				continue
			}

			is.NotEqual(-1, n)
			is.Greater(n, 0)

			n = DecodedLength(-v)
			is.Equal(-1, n)
		}
	}

	// DecodedLength (zero case)
	{
		n := DecodedLength(0)
		is.Equal(0, n)
	}
}

type dCall uint8

const (
	unsafeDecCall dCall = iota + 1
	decCall
	appendDecCall
)

func (c dCall) canHaveNilDst() bool {
	switch c {
	case decCall, appendDecCall:
		return true
	case unsafeDecCall:
		return false
	default:
		panic("invalid dCall value")
	}
}

type decodeTC struct {
	// the function operation to call
	call dCall
	// src is the source data to encode
	src string
	// dst is where encoded data will be placed
	dst []byte

	// expectations

	expStr    string
	expErrStr string
	expErr    error
	expPanic  any
}

type decodeTCR struct {
	str    string
	err    error
	nilDst bool
}

func (tc decodeTC) clone() decodeTC {
	ctc := tc

	ctc.dst = slices.Clone(tc.dst)

	return ctc
}

func (tc decodeTC) runUnsafeDec(t *testing.T, src []byte) decodeTCR {
	t.Helper()

	is := assert.New(t)

	if tc.expPanic != nil {
		is.PanicsWithValue(tc.expPanic, func() {
			UnsafeDecode(tc.dst, src)
		})

		return decodeTCR{string(tc.dst), nil, tc.dst == nil}
	}

	err := UnsafeDecode(tc.dst, src)
	return decodeTCR{string(tc.dst), err, tc.dst == nil}
}

func (tc decodeTC) runDec(t *testing.T, src []byte) decodeTCR {
	t.Helper()

	dst, err := Decode(src)
	return decodeTCR{string(dst), err, dst == nil}
}

func (tc decodeTC) runAppendDec(t *testing.T, src []byte) decodeTCR {
	t.Helper()

	dst, err := AppendDecode(tc.dst, src)
	return decodeTCR{string(dst), err, dst == nil}
}

func cloneDecodeTC(tc decodeTC) decodeTC {
	return tc.clone()
}

func descDecodeTC(t *testing.T, cfg tbdd.Describe[decodeTC]) tbdd.DescribeResponse {
	t.Helper()

	is := assert.New(t)

	tc := cfg.TC
	when := cfg.When
	then := cfg.Then

	is.NotEmpty(when)
	// Infer 'then' if not already defined.
	if then == "" {
		if tc.expPanic != nil {
			then = "should panic"
		} else if tc.expErr != nil || tc.expErrStr != "" {
			then = "should error"
		} else {
			then = "should succeed"
		}
	}

	return tbdd.DescribeResponse{
		When: when,
		Then: then,
	}
}

func runDecodeTC(t *testing.T, tc decodeTC) decodeTCR {
	t.Helper()

	is := assert.New(t)

	// verify TC configuration expectations makes sense
	if tc.expPanic != nil {
		// individual checks before potential unified failure
		is.Nil(tc.expErr)
		is.Empty(tc.expErrStr)
		is.Empty(tc.expStr)

		if tc.expErr != nil || tc.expErrStr != "" || tc.expStr != "" {
			t.Fatal("invalid test case config: when a panic is expected, nothing else should be expected")
		}
	} else if tc.expErr == nil && tc.expErrStr == "" && len(tc.src) > 0 && tc.expStr == "" {
		t.Fatal("invalid test case config: test case expects an empty result when input is non-zero and no panics nor errors are expected")
	}

	var src []byte
	if len(tc.src) > 0 {
		src = []byte(tc.src)
	}

	switch tc.call {
	case unsafeDecCall:
		return tc.runUnsafeDec(t, src)
	case decCall:
		return tc.runDec(t, src)
	case appendDecCall:
		return tc.runAppendDec(t, src)
	default:
		panic("misconfigured test case")
	}
}

func checkDecodeTCR(t *testing.T, cfg tbdd.Assert[decodeTC, decodeTCR]) {
	t.Helper()

	is := assert.New(t)

	tc := cfg.TC
	r := cfg.Result

	if tc.expPanic != nil {
		return
	}

	if tc.expErr == nil && tc.expErrStr == "" {
		is.Nil(r.err)
		is.Equal(tc.expStr, r.str)
	} else if tc.call.canHaveNilDst() && (len(tc.src) == 0 || r.err == ErrInvalidBase32Length) {
		is.True(r.nilDst)
	}

	if tc.expErr != nil || tc.expErrStr != "" {
		is.NotNil(r.err)

		if exp := tc.expErr; exp != nil {
			is.ErrorIs(r.err, exp)
		}

		if exp := tc.expErrStr; exp != "" {
			is.Equal(exp, r.err.Error())
		}
	} else if len(tc.src) > 0 {
		is.NotEmpty(tc.expStr)

		is.Equal(tc.expStr, r.str)
	}
}

func decodeTCVariants(t *testing.T, tc decodeTC) iter.Seq[tbdd.TestVariant[decodeTC]] {
	t.Helper()

	return func(yield func(tbdd.TestVariant[decodeTC]) bool) {
		t.Helper()

		if tc.call != decCall || tc.expPanic != nil || tc.expErr != nil || tc.expErrStr != "" {
			return
		}

		{
			tc := tc.clone()

			dst := []byte(`test_`)
			tc.expStr = string(dst) + tc.expStr
			tc.dst = dst
			tc.call = appendDecCall

			if !yield(tbdd.TestVariant[decodeTC]{
				TC:          tc,
				Kind:        "decCall2appendDecCall",
				SkipCloneTC: true,
			}) {
				return
			}
		}

		{
			tc := tc.clone()

			tc.call = appendDecCall

			if !yield(tbdd.TestVariant[decodeTC]{
				TC:          tc,
				Kind:        "decCall2appendDecCall-nil-dst",
				SkipCloneTC: true,
			}) {
				return
			}
		}

		if len(tc.src) > 0 {
			tc := tc.clone()

			tc.dst = make([]byte, len(tc.expStr))
			tc.call = unsafeDecCall

			if !yield(tbdd.TestVariant[decodeTC]{
				TC:          tc,
				Kind:        "decCall2unsafeDecCall",
				SkipCloneTC: true,
			}) {
				return
			}
		}
	}
}

// TestDecode uses the tbdd.Lifecycle "test helper".
// For each entry in tcs:
//   - TC describes inputs + expectations.
//   - Act (runDecodeTC) runs the appropriate decode function based on TC.call.
//   - Assert (checkDecodeTCR) validates the result against expectations.
//   - Variants (decodeTCVariants) generate additional derived test cases.
//   - Describe (descDecodeTC) fills in the "then" string if not set.
//
// To add a new scenario, append a new tbdd.Lifecycle entry to tcs.
func TestDecode(t *testing.T) {
	t.Parallel()

	tcs := []tbdd.Lifecycle[decodeTC, decodeTCR]{
		{
			When: "8 bytes",
			TC: decodeTC{
				src:    "64S36D1N",
				expStr: "12345",
			},
		},
		{
			When: "8 bytes where last is invalid",
			TC: decodeTC{
				src:    "64S36D1U",
				expErr: ErrInvalidBase32Char,
			},
		},
		{
			When: "31 bytes",
			TC: decodeTC{
				src:    "64S36D1N6RVKGE9G64S36D1N6RVKGE8",
				expStr: "1234567890123456789",
			},
		},
		{
			When: "31 bytes where last is invalid",
			TC: decodeTC{
				src:    "64S36D1N6RVKGE9G64S36D1N6RVKGEU",
				expErr: ErrInvalidBase32Char,
			},
		},
		{
			When: "31 bytes with invalid tail bits",
			TC: decodeTC{
				src:    "64S36D1N6RVKGE9G64S36D1N6RVKGE4",
				expErr: ErrInvalidBase32Char,
			},
		},
		{
			When: "30 bytes",
			TC: decodeTC{
				src:    "64S36D1N6RVKGE9G64S36D1N6RVKGE",
				expErr: ErrInvalidBase32Length,
			},
		},
		{
			When: "29 bytes",
			TC: decodeTC{
				src:    "64S36D1N6RVKGE9G64S36D1N6RVKG",
				expStr: "123456789012345678",
			},
		},
		{
			When: "29 bytes where last is invalid",
			TC: decodeTC{
				src:    "64S36D1N6RVKGE9G64S36D1N6RVKU",
				expErr: ErrInvalidBase32Char,
			},
		},
		{
			When: "29 bytes with invalid tail bits",
			TC: decodeTC{
				src:    "64S36D1N6RVKGE9G64S36D1N6RVK1",
				expErr: ErrInvalidBase32Char,
			},
		},
		{
			When: "28 bytes where last is invalid",
			TC: decodeTC{
				src:    "64S36D1N6RVKGE9G64S36D1N6RVU",
				expErr: ErrInvalidBase32Char,
			},
		},
		{
			When: "28 bytes with invalid tail bits",
			TC: decodeTC{
				src:    "64S36D1N6RVKGE9G64S36D1N6RV8",
				expErr: ErrInvalidBase32Char,
			},
		},
		{
			When: "28 bytes",
			TC: decodeTC{
				src:    "64S36D1N6RVKGE9G64S36D1N6RVG",
				expStr: "12345678901234567",
			},
		},
		{
			When: "27 bytes",
			TC: decodeTC{
				src:    "64S36D1N6RVKGE9G64S36D1N6RV",
				expErr: ErrInvalidBase32Length,
			},
		},
		{
			When: "26 bytes where last is invalid",
			TC: decodeTC{
				src:    "64S36D1N6RVKGE9G64S36D1N6U",
				expErr: ErrInvalidBase32Char,
			},
		},
		{
			When: "26 bytes with invalid tail bits",
			TC: decodeTC{
				src:    "64S36D1N6RVKGE9G64S36D1N62",
				expErr: ErrInvalidBase32Char,
			},
		},
		{
			When: "26 bytes",
			TC: decodeTC{
				src:    "64S36D1N6RVKGE9G64S36D1N6R",
				expStr: "1234567890123456",
			},
		},
		{
			When: "25 bytes",
			TC: decodeTC{
				src:    "64S36D1N6RVKGE9G64S36D1N6",
				expErr: ErrInvalidBase32Length,
			},
		},
		{
			When: "24 bytes",
			TC: decodeTC{
				src:    "64S36D1N6RVKGE9G64S36D1N",
				expStr: "123456789012345",
			},
		},
		{
			When: "0 bytes",
			TC: decodeTC{
				call: decCall,
			},
		},
		{
			When: "unsafe-decode destination has no capacity and source is not empty",
			TC: decodeTC{
				call:     unsafeDecCall,
				src:      "00",
				dst:      []byte{},
				expPanic: "base32: decode destination too short",
			},
		},
		{
			When: "unsafe-decode src is empty",
			TC: decodeTC{
				call:     unsafeDecCall,
				src:      "",
				expPanic: "base32: invalid decode source length",
			},
		},
		{
			When: "append-decode source is invalid length",
			TC: decodeTC{
				call:   appendDecCall,
				src:    "0",
				expErr: ErrInvalidBase32Length,
			},
		},
		{
			When: "append-decode source has an invalid char",
			TC: decodeTC{
				call:   appendDecCall,
				src:    "0U",
				expErr: ErrInvalidBase32Char,
			},
		},
	}

	for i, tc := range tcs {
		tc.CloneTC = cloneDecodeTC
		tc.Variants = decodeTCVariants
		tc.Describe = descDecodeTC
		tc.Act = runDecodeTC
		tc.Assert = checkDecodeTCR

		// if no call is specified, use decCall
		if tc.TC.call == 0 {
			tc.TC.call = decCall
		}

		// expand single-error checks to string based error checking as well
		if err := tc.TC.expErr; err != nil && tc.TC.expErrStr == "" {
			tc.TC.expErrStr = err.Error()
		}

		f := tc.NewI(t, i)
		f(t)
	}
}
