package base32

import (
	"iter"
	"math"
	"slices"
	"testing"

	"github.com/josephcopenhaver/tbdd-go"
	"github.com/stretchr/testify/assert"
)

func Test_encodedLen(t *testing.T) {
	t.Parallel()

	is := assert.New(t)

	const inputTooBig = 5 + (math.MaxInt / 8 * 5)
	const inputOK = math.MaxInt / 8 * 5
	const outputOK = (inputOK/5)*8 + ((inputOK%5)*8+4)/5

	is.PanicsWithValue("base32: invalid encode source length", func() {
		encodedLen(inputTooBig)
	})
	is.Equal(-1, EncodedLength(inputTooBig))

	is.Equal(outputOK, encodedLen(inputOK))
	is.Equal(outputOK, EncodedLength(inputOK))
	is.Equal(0, EncodedLength(0))
	is.Equal(-1, EncodedLength(-inputOK))
}

type eCall uint8

const (
	unsafeEncCall eCall = iota + 1
	encCall
	appendEncCall
	encStrCall
	appendEncStrCall
)

type encodeTC struct {
	// the function operation to call
	call eCall
	// srcLen determines the source byte length to test
	srcLen int
	// src is the source data to encode
	src string
	// dst is where encoded data will be placed
	dst []byte

	// expectations

	expStr   string
	expPanic any
}

type encodeTCR struct {
	str    string
	nilDst bool
}

func (tc encodeTC) clone() encodeTC {
	ctc := tc

	ctc.dst = slices.Clone(tc.dst)

	return ctc
}

func cloneEncodeTC(tc encodeTC) encodeTC {
	return tc.clone()
}

func descEncodeTC(t *testing.T, cfg tbdd.Describe[encodeTC]) tbdd.DescribeResponse {
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
		} else {
			then = "should succeed"
		}
	}

	return tbdd.DescribeResponse{
		When: when,
		Then: then,
	}
}

func runEncodeTC(t *testing.T, tc encodeTC) encodeTCR {
	t.Helper()

	is := assert.New(t)

	// verify TC configuration expectations makes sense
	if tc.expPanic != nil {
		// individual checks before potential unified failure
		is.Empty(tc.expStr)

		if tc.expStr != "" {
			t.Fatal("invalid test case config: when a panic is expected, nothing else should be expected")
		}
	} else if len(tc.src) > 0 && tc.expStr == "" {
		t.Fatal("invalid test case config: test case expects an empty result when input is non-zero and no panics are expected")
	}

	var src []byte
	{
		length := tc.srcLen
		if length == 0 {
			length = len(tc.src)
		}
		if length > 0 {
			src = []byte(tc.src[:length])
		}
	}

	switch tc.call {
	case unsafeEncCall:
		if tc.expPanic != nil {
			is.PanicsWithValue(tc.expPanic, func() {
				UnsafeEncode(tc.dst, src)
			})
			return encodeTCR{}
		}

		UnsafeEncode(tc.dst, src)

		return encodeTCR{string(tc.dst), false}
	case encCall:
		is.Nil(tc.dst)

		resp := Encode(src)

		return encodeTCR{string(resp), resp == nil}
	case appendEncCall:
		resp := AppendEncode(tc.dst, src)

		return encodeTCR{string(resp), resp == nil}
	case encStrCall:
		resp := EncodeString(string(src))

		return encodeTCR{resp, false}
	case appendEncStrCall:
		resp := AppendEncodeString(tc.dst, string(src))

		return encodeTCR{string(resp), resp == nil}
	default:
		panic("misconfigured test case")
	}
}

func checkEncodeTCR(t *testing.T, cfg tbdd.Assert[encodeTC, encodeTCR]) {
	t.Helper()

	is := assert.New(t)

	tc := cfg.TC
	r := cfg.Result

	if tc.expPanic != nil {
		return
	}

	switch tc.call {
	case unsafeEncCall, encStrCall:
	case encCall:
		if tc.expStr == "" {
			is.True(r.nilDst)
		}
	case appendEncCall, appendEncStrCall:
		if len(tc.src) == 0 && tc.dst == nil {
			is.True(r.nilDst)
		}
	default:
		panic("misconfigured test case")
	}

	is.Equal(tc.expStr, string(r.str))
}

func encodeTCVariants(t *testing.T, tc encodeTC) iter.Seq[tbdd.TestVariant[encodeTC]] {
	t.Helper()

	return func(yield func(tbdd.TestVariant[encodeTC]) bool) {
		t.Helper()

		if tc.call != encCall || tc.expPanic != nil {
			return
		}

		{
			tc := tc.clone()

			tc.call = encStrCall

			if !yield(tbdd.TestVariant[encodeTC]{
				TC:          tc,
				Kind:        "encCall2encStringCall",
				SkipCloneTC: true,
			}) {
				return
			}
		}

		{
			tc := tc.clone()

			dst := []byte(`test_`)
			tc.expStr = string(dst) + tc.expStr
			tc.dst = dst
			tc.call = appendEncCall

			if !yield(tbdd.TestVariant[encodeTC]{
				TC:          tc,
				Kind:        "encCall2appendEncCall",
				SkipCloneTC: true,
			}) {
				return
			}
		}

		{
			tc := tc.clone()

			dst := []byte(`test_`)
			tc.expStr = string(dst) + tc.expStr
			tc.dst = dst
			tc.call = appendEncStrCall

			if !yield(tbdd.TestVariant[encodeTC]{
				TC:          tc,
				Kind:        "encCall2appendEncStrCall",
				SkipCloneTC: true,
			}) {
				return
			}
		}

		{
			tc := tc.clone()

			tc.call = appendEncCall

			if !yield(tbdd.TestVariant[encodeTC]{
				TC:          tc,
				Kind:        "encCall2appendEncCall-nil-dst",
				SkipCloneTC: true,
			}) {
				return
			}
		}

		{
			tc := tc.clone()

			tc.call = appendEncStrCall

			if !yield(tbdd.TestVariant[encodeTC]{
				TC:          tc,
				Kind:        "encCall2appendEncStringCall",
				SkipCloneTC: true,
			}) {
				return
			}
		}

		if len(tc.src) > 0 {
			tc := tc.clone()

			tc.dst = make([]byte, len(tc.expStr))
			tc.call = unsafeEncCall

			if !yield(tbdd.TestVariant[encodeTC]{
				TC:          tc,
				Kind:        "encCall2unsafeEncCall",
				SkipCloneTC: true,
			}) {
				return
			}
		}
	}
}

// TestEncode uses the tbdd.Lifecycle "test helper".
// For each entry in tcs:
//   - TC describes inputs + expectations.
//   - Act (runEncodeTC) runs the appropriate encode function based on TC.call.
//   - Assert (checkEncodeTCR) validates the result against expectations.
//   - Variants (encodeTCVariants) generate additional derived test cases.
//   - Describe (descEncodeTC) fills in the "then" string if not set.
//
// To add a new scenario, append a new tbdd.Lifecycle entry to tcs.
func TestEncode(t *testing.T) {
	t.Parallel()

	tcs := []tbdd.Lifecycle[encodeTC, encodeTCR]{
		{
			When: "19 bytes",
			TC: encodeTC{
				src:    "1234567890123456789",
				srcLen: 19,
				expStr: "64S36D1N6RVKGE9G64S36D1N6RVKGE8",
			},
		},
		{
			When: "18 bytes",
			TC: encodeTC{
				src:    "1234567890123456789",
				srcLen: 18,
				expStr: "64S36D1N6RVKGE9G64S36D1N6RVKG",
			},
		},
		{
			When: "17 bytes",
			TC: encodeTC{
				src:    "1234567890123456789",
				srcLen: 17,
				expStr: "64S36D1N6RVKGE9G64S36D1N6RVG",
			},
		},
		{
			When: "16 bytes",
			TC: encodeTC{
				src:    "1234567890123456789",
				srcLen: 16,
				expStr: "64S36D1N6RVKGE9G64S36D1N6R",
			},
		},
		{
			When: "15 bytes",
			TC: encodeTC{
				src:    "1234567890123456789",
				srcLen: 15,
				expStr: "64S36D1N6RVKGE9G64S36D1N",
			},
		},
		{
			When: "0 bytes",
			TC: encodeTC{
				call: encCall,
			},
		},
		{
			When: "unsafe-encode destination has no capacity and source is not empty",
			TC: encodeTC{
				call:     unsafeEncCall,
				src:      "1",
				dst:      []byte{},
				expPanic: "base32: encode destination too short",
			},
		},
		{
			When: "unsafe-encode src is empty",
			TC: encodeTC{
				call:     unsafeEncCall,
				src:      "",
				expPanic: "base32: invalid encode source length",
			},
		},
	}

	for i, tc := range tcs {
		tc.CloneTC = cloneEncodeTC
		tc.Variants = encodeTCVariants
		tc.Describe = descEncodeTC
		tc.Act = runEncodeTC
		tc.Assert = checkEncodeTCR

		// if no call is specified, use encCall
		if tc.TC.call == 0 {
			tc.TC.call = encCall
		}

		f := tc.NewI(t, i)
		f(t)
	}
}
