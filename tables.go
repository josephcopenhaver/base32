// FILE: github.com/josephcopenhaver/base32/tables.go

// A case insensitive Crockford style base32 implementation.

package base32

const b32Invalid = 0xFF

//
// encode and decode tables are using Crockford style case insensitive grammars
//

var encodeTab, decodeTab = func() ([32]byte, [256]byte) {
	const (
		b32Chars   = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
		b32UpToLow = ('a' - 'A')
	)

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
