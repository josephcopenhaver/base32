package base32

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_b32EncodedLen(t *testing.T) {
	is := assert.New(t)

	is.PanicsWithValue("base32: invalid encode source length", func() {
		b32EncodedLen(5 + (math.MaxInt / 8 * 5))
	})

	is.NotPanics(func() {
		input := 3 + (math.MaxInt / 8 * 5)
		resp := b32EncodedLen(input)
		is.Equal((input/5)*8+((input%5)*8+4)/5, resp)
	})
}
