package uax14

import "unicode/utf8"

// lookupProperty normalizes trie lookup results for internal algorithm use.
// Valid UTF-8 scalars that are unmapped in LineBreak data default to XX.
func lookupProperty[T ~string | ~[]byte](data T) (property, int) {
	v, sz := lookup(data)
	if v != 0 || sz == 0 || len(data) < sz {
		return v, sz
	}

	if isValidUTF8Prefix(data, sz) {
		return _AL, sz
	}
	return 0, 0
}

func isValidUTF8Prefix[T ~string | ~[]byte](data T, sz int) bool {
	switch x := any(data).(type) {
	case string:
		r, w := utf8.DecodeRuneInString(x[:sz])
		return w == sz && (r != utf8.RuneError || sz > 1)
	case []byte:
		r, w := utf8.DecodeRune(x[:sz])
		return w == sz && (r != utf8.RuneError || sz > 1)
	default:
		return false
	}
}
