package base32

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTables(t *testing.T) {
	t.Parallel()

	const (
		b32Chars         = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
		invalidDecodeVal = byte(b32Invalid)
	)

	is := assert.New(t)

	validChar := func(c byte) (byte, int8) {
		if c >= 'a' && c <= 'z' {
			c -= ('a' - 'A')
		}
		switch c {
		case 'O':
			c = '0'
		case 'I':
			c = '1'
		case 'L':
			c = '1'
		}
		return c, int8(strings.IndexByte(b32Chars, c))
	}

	for i := range 256 {
		c := byte(i)

		uc, i := validChar(c)
		if i == -1 {
			is.Equal(invalidDecodeVal, decodeTab[c])
			continue
		}

		is.Equal(i, int8(decodeTab[c]))
		is.Equal(uc, encodeTab[i])
	}

	// verify hardcoded alias values
	is.Equal(uint8(0), decodeTab['0'])
	is.Equal(uint8(1), decodeTab['1'])
}
