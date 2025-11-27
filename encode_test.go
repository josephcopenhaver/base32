package base32

import (
	"math"
	"slices"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_encodedLen(t *testing.T) {
	t.Parallel()

	is := assert.New(t)

	is.PanicsWithValue("base32: invalid encode source length", func() {
		encodedLen(5 + (math.MaxInt / 8 * 5))
	})

	is.NotPanics(func() {
		input := (math.MaxInt / 8 * 5)
		resp := encodedLen(input)
		is.Equal((input/5)*8+((input%5)*8+4)/5, resp)
	})
}

type eCall uint8

const (
	unsafeEncCall eCall = iota + 1
	encCall
	encAppendCall
)

type encoderTestCase struct {
	// given describes initial configurations in a BDD style
	given func(*testing.T, encoderTestCase) (string, encoderTestCase, func(func(*testing.T)) func(*testing.T))
	// when describes the action being taken under the initial conditions of given in a BDD style
	when string
	// then describes the desired outcome from the action taken in a BDD style
	then string
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

func (tc encoderTestCase) clone() encoderTestCase {
	ctc := tc

	ctc.dst = slices.Clone(tc.dst)

	return ctc
}

func (tc encoderTestCase) runTI(t *testing.T, tci int) {
	t.Helper()

	f := func(tc encoderTestCase, extraCfg string) func(*testing.T) {
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

				then := tc.then
				if then == "" {
					if tc.expPanic != nil {
						then = "a panic should occur"
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

	f(tc, "")(t)

	if tc.call == encCall && tc.expPanic == nil {
		{
			tc := tc.clone()

			dst := []byte(`test_`)
			tc.expStr = string(dst) + tc.expStr
			tc.dst = dst
			tc.call = encAppendCall
			f(tc, "encCall2encAppendCall")(t)
		}

		{
			tc := tc.clone()

			tc.call = encAppendCall
			f(tc, "encCall2encAppendCall-nil-dst")(t)
		}

		if len(tc.src) > 0 {
			tc := tc.clone()

			tc.dst = make([]byte, len(tc.expStr))
			tc.call = unsafeEncCall
			f(tc, "encCall2unsafeEncCall")(t)
		}
	}
}

func (tc encoderTestCase) run(t *testing.T) {
	is := assert.New(t)

	length := tc.srcLen
	if length == 0 {
		length = len(tc.src)
	}
	var src []byte
	if length > 0 {
		src = []byte(tc.src[:length])
	}

	switch tc.call {
	case unsafeEncCall:
		if tc.expPanic != nil {
			is.PanicsWithValue(tc.expPanic, func() {
				UnsafeEncode(tc.dst, src)
			})
			is.Empty(tc.expStr)
			break
		}

		is.NotPanics(func() {
			UnsafeEncode(tc.dst, src)
		})
		is.Equal(tc.expStr, string(tc.dst))
	case encCall:
		is.Nil(tc.dst)

		resp := Encode(src)

		if tc.expStr == "" {
			is.Nil(resp)
			break
		}

		is.Equal(tc.expStr, string(resp))
	case encAppendCall:
		resp := AppendEncode(tc.dst, src)

		if len(src) == 0 && tc.dst == nil {
			is.Nil(resp)
			break
		}

		is.Equal(tc.expStr, string(resp))
	default:
		panic("misconfigured test case")
	}
}

func TestEncode(t *testing.T) {
	t.Parallel()

	tcs := []encoderTestCase{
		{
			when:   "19 bytes",
			call:   encCall,
			src:    "1234567890123456789",
			srcLen: 19,
			expStr: "64S36D1N6RVKGE9G64S36D1N6RVKGE8",
		},
		{
			when:   "18 bytes",
			call:   encCall,
			src:    "1234567890123456789",
			srcLen: 18,
			expStr: "64S36D1N6RVKGE9G64S36D1N6RVKG",
		},
		{
			when:   "17 bytes",
			call:   encCall,
			src:    "1234567890123456789",
			srcLen: 17,
			expStr: "64S36D1N6RVKGE9G64S36D1N6RVG",
		},
		{
			when:   "16 bytes",
			call:   encCall,
			src:    "1234567890123456789",
			srcLen: 16,
			expStr: "64S36D1N6RVKGE9G64S36D1N6R",
		},
		{
			when:   "15 bytes",
			call:   encCall,
			src:    "1234567890123456789",
			srcLen: 15,
			expStr: "64S36D1N6RVKGE9G64S36D1N",
		},
		{
			when: "0 bytes",
			call: encCall,
		},
		{
			when:     "unsafe-encode destination has no capacity and source is not empty",
			call:     unsafeEncCall,
			src:      "1",
			dst:      []byte{},
			expPanic: "base32: encode destination too short",
		},
		{
			when:     "unsafe-encode src is empty",
			call:     unsafeEncCall,
			src:      "",
			expPanic: "base32: invalid encode source length",
		},
	}

	for i, tc := range tcs {
		tc.runTI(t, i)
	}
}
