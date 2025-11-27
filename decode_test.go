package base32

import (
	"math"
	"slices"
	"strconv"
	"testing"

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
		n := decodedLen((math.MaxInt-7)/8*8 + int(i))

		if invalidRemainders[i] {
			is.Equal(-1, n)
			continue
		}

		is.NotEqual(-1, n)
		is.Greater(n, 0)
	}
}

type dCall uint8

const (
	unsafeDecCall dCall = iota + 1
	decCall
	appendDecCall
)

type decoderTestCase struct {
	// given describes initial configurations in a BDD style
	given func(*testing.T, decoderTestCase) (string, decoderTestCase, func(func(*testing.T)) func(*testing.T))
	// when describes the action being taken under the initial conditions of given in a BDD style
	when string
	// then describes the desired outcome from the action taken in a BDD style
	then string
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

func (tc decoderTestCase) clone() decoderTestCase {
	ctc := tc

	ctc.dst = slices.Clone(tc.dst)

	return ctc
}

func (tc decoderTestCase) runTI(t *testing.T, tci int) {
	t.Helper()

	f := func(tc decoderTestCase, extraCfg string) func(*testing.T) {
		tc = tc.clone()

		var givenStr string
		var given func(func(*testing.T)) func(*testing.T)
		if f := tc.given; f != nil {
			givenStr, tc, given = f(t, tc)
		}

		f := func(t *testing.T) {
			t.Helper()

			t.Run("when "+tc.when, func(t *testing.T) {
				t.Helper()
				if tc.expErr != nil && tc.expPanic != nil {
					t.Fatal("found invalid test case config")
				}

				then := tc.then
				if then == "" {
					if tc.expPanic != nil {
						then = "a panic should occur"
					} else if tc.expErr != nil {
						then = "an error should occur"
					} else {
						then = "no error should occur"
					}
				}
				t.Run("then "+then, func(t *testing.T) {
					t.Helper()

					tc.run(t)
				})
			})
		}

		if given != nil {
			if givenStr == "" {
				givenStr = "context unspecified"
			}
			nf := given(f)
			f = func(t *testing.T) {
				t.Helper()

				t.Run("given "+givenStr, nf)
			}
		}

		{
			var prefix string

			if tci >= 0 {
				prefix = strconv.Itoa(tci)
			}

			if extraCfg != "" {
				if prefix != "" {
					prefix += "/"
				}
				prefix += extraCfg
			}

			if prefix != "" {
				nf := f
				f = func(t *testing.T) {
					t.Helper()

					t.Run(prefix, nf)
				}
			}
		}

		return f
	}

	tc.runVariants(t, f)
}

func (tc decoderTestCase) runVariants(t *testing.T, f func(decoderTestCase, string) func(*testing.T)) {
	t.Helper()

	f(tc, "")(t)

	if tc.call == decCall && tc.expPanic == nil && tc.expErr == nil && tc.expErrStr == "" {
		{
			tc := tc.clone()

			dst := []byte(`test_`)
			tc.expStr = string(dst) + tc.expStr
			tc.dst = dst
			tc.call = appendDecCall
			f(tc, "decCall2appendDecCall")(t)
		}

		{
			tc := tc.clone()

			tc.call = appendDecCall
			f(tc, "decCall2appendDecCall-nil-dst")(t)
		}

		if len(tc.src) > 0 {
			tc := tc.clone()

			tc.dst = make([]byte, len(tc.expStr))
			tc.call = unsafeDecCall
			f(tc, "decCall2unsafeDecCall")(t)
		}
	}
}

func (tc decoderTestCase) run(t *testing.T) {
	t.Helper()

	var src []byte
	if len(tc.src) > 0 {
		src = []byte(tc.src)
	}

	switch tc.call {
	case unsafeDecCall:
		tc.testUnsafeDec(t, src)
	case decCall:
		tc.testDec(t, src)
	case appendDecCall:
		tc.testAppendDec(t, src)
	default:
		panic("misconfigured test case")
	}
}

func (tc decoderTestCase) testUnsafeDec(t *testing.T, src []byte) {
	t.Helper()

	is := assert.New(t)

	if tc.expPanic != nil {
		is.PanicsWithValue(tc.expPanic, func() {
			UnsafeDecode(tc.dst, src)
		})
		is.Empty(tc.expStr)
		is.Empty(tc.expErr)
		is.Empty(tc.expErrStr)
		return
	}

	var errResp error
	is.NotPanics(func() {
		errResp = UnsafeDecode(tc.dst, src)
	})

	if tc.expErr != nil {
		is.ErrorIs(errResp, tc.expErr)
	}

	if tc.expErrStr != "" {
		is.Equal(tc.expErrStr, errResp.Error())
	}

	if tc.expErr == nil && tc.expErrStr == "" {
		is.Nil(errResp)
		is.Equal(tc.expStr, string(tc.dst))
	}
	// otherwise dst could be dirty, out of scope to evaluate
}

func (tc decoderTestCase) testDec(t *testing.T, src []byte) {
	t.Helper()

	is := assert.New(t)

	is.Nil(tc.dst)

	resp, errResp := Decode(src)

	if tc.expErr != nil {
		is.ErrorIs(errResp, tc.expErr)
	}

	if tc.expErrStr != "" {
		is.Equal(tc.expErrStr, errResp.Error())
	}

	if tc.expErr == nil && tc.expErrStr == "" {
		is.Nil(errResp)
		is.Equal(tc.expStr, string(resp))
	} else if src == nil || errResp == ErrInvalidBase32Length {
		is.Nil(resp)
	}
	// otherwise resp could be dirty, out of scope to evaluate
}

func (tc decoderTestCase) testAppendDec(t *testing.T, src []byte) {
	t.Helper()

	is := assert.New(t)

	resp, errResp := AppendDecode(tc.dst, src)

	if tc.expErr != nil {
		is.ErrorIs(errResp, tc.expErr)
	}

	if tc.expErrStr != "" {
		is.Equal(tc.expErrStr, errResp.Error())
	}

	if tc.expErr == nil && tc.expErrStr == "" {
		is.Nil(errResp)
		is.Equal(tc.expStr, string(resp))
	} else if src == nil || errResp == ErrInvalidBase32Length {
		is.Nil(resp)
	}
	// otherwise resp could be dirty, out of scope to evaluate
}

func TestDecode(t *testing.T) {
	t.Parallel()

	tcs := []decoderTestCase{
		{
			when:   "8 bytes",
			call:   decCall,
			src:    "64S36D1N",
			expStr: "12345",
		},
		{
			when:      "8 bytes where last is invalid",
			call:      decCall,
			src:       "64S36D1U",
			expErr:    ErrInvalidBase32Char,
			expErrStr: ErrInvalidBase32Char.Error(),
		},
		{
			when:   "31 bytes",
			call:   decCall,
			src:    "64S36D1N6RVKGE9G64S36D1N6RVKGE8",
			expStr: "1234567890123456789",
		},
		{
			when:      "31 bytes where last is invalid",
			call:      decCall,
			src:       "64S36D1N6RVKGE9G64S36D1N6RVKGEU",
			expErr:    ErrInvalidBase32Char,
			expErrStr: ErrInvalidBase32Char.Error(),
		},
		{
			when:      "31 bytes with invalid tail bits",
			call:      decCall,
			src:       "64S36D1N6RVKGE9G64S36D1N6RVKGE4",
			expErr:    ErrInvalidBase32Char,
			expErrStr: ErrInvalidBase32Char.Error(),
		},
		{
			when:      "30 bytes",
			call:      decCall,
			src:       "64S36D1N6RVKGE9G64S36D1N6RVKGE",
			expErr:    ErrInvalidBase32Length,
			expErrStr: ErrInvalidBase32Length.Error(),
		},
		{
			when:   "29 bytes",
			call:   decCall,
			src:    "64S36D1N6RVKGE9G64S36D1N6RVKG",
			expStr: "123456789012345678",
		},
		{
			when:      "29 bytes where last is invalid",
			call:      decCall,
			src:       "64S36D1N6RVKGE9G64S36D1N6RVKU",
			expErr:    ErrInvalidBase32Char,
			expErrStr: ErrInvalidBase32Char.Error(),
		},
		{
			when:      "29 bytes with invalid tail bits",
			call:      decCall,
			src:       "64S36D1N6RVKGE9G64S36D1N6RVK1",
			expErr:    ErrInvalidBase32Char,
			expErrStr: ErrInvalidBase32Char.Error(),
		},
		{
			when:      "28 bytes where last is invalid",
			call:      decCall,
			src:       "64S36D1N6RVKGE9G64S36D1N6RVU",
			expErr:    ErrInvalidBase32Char,
			expErrStr: ErrInvalidBase32Char.Error(),
		},
		{
			when:      "28 bytes with invalid tail bits",
			call:      decCall,
			src:       "64S36D1N6RVKGE9G64S36D1N6RV8",
			expErr:    ErrInvalidBase32Char,
			expErrStr: ErrInvalidBase32Char.Error(),
		},
		{
			when:   "28 bytes",
			call:   decCall,
			src:    "64S36D1N6RVKGE9G64S36D1N6RVG",
			expStr: "12345678901234567",
		},
		{
			when:      "27 bytes",
			call:      decCall,
			src:       "64S36D1N6RVKGE9G64S36D1N6RV",
			expErr:    ErrInvalidBase32Length,
			expErrStr: ErrInvalidBase32Length.Error(),
		},
		{
			when:      "26 bytes where last is invalid",
			call:      decCall,
			src:       "64S36D1N6RVKGE9G64S36D1N6U",
			expErr:    ErrInvalidBase32Char,
			expErrStr: ErrInvalidBase32Char.Error(),
		},
		{
			when:      "26 bytes with invalid tail bits",
			call:      decCall,
			src:       "64S36D1N6RVKGE9G64S36D1N62",
			expErr:    ErrInvalidBase32Char,
			expErrStr: ErrInvalidBase32Char.Error(),
		},
		{
			when:   "26 bytes",
			call:   decCall,
			src:    "64S36D1N6RVKGE9G64S36D1N6R",
			expStr: "1234567890123456",
		},
		{
			when:      "25 bytes",
			call:      decCall,
			src:       "64S36D1N6RVKGE9G64S36D1N6",
			expErr:    ErrInvalidBase32Length,
			expErrStr: ErrInvalidBase32Length.Error(),
		},
		{
			when:   "24 bytes",
			call:   decCall,
			src:    "64S36D1N6RVKGE9G64S36D1N",
			expStr: "123456789012345",
		},
		{
			when: "0 bytes",
			call: decCall,
		},
		{
			when:     "unsafe-decode destination has no capacity and source is not empty",
			call:     unsafeDecCall,
			src:      "00",
			dst:      []byte{},
			expPanic: "base32: decode destination too short",
		},
		{
			when:     "unsafe-decode src is empty",
			call:     unsafeDecCall,
			src:      "",
			expPanic: "base32: invalid decode source length",
		},
		{
			when:      "append-decode source is invalid length",
			call:      appendDecCall,
			src:       "0",
			expErr:    ErrInvalidBase32Length,
			expErrStr: ErrInvalidBase32Length.Error(),
		},
		{
			when:      "append-decode source has an invalid char",
			call:      appendDecCall,
			src:       "0U",
			expErr:    ErrInvalidBase32Char,
			expErrStr: ErrInvalidBase32Char.Error(),
		},
	}

	for i, tc := range tcs {
		tc.runTI(t, i)
	}
}
