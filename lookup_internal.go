package uax14

import "unicode/utf8"

// lookupProperty normalizes trie lookup results for internal algorithm use.
// Valid UTF-8 scalars that are unmapped in LineBreak data default to XX.
func lookupProperty[T ~string | ~[]byte](in T) property {
	v, sz := lookup(in)
	if v != 0 || sz == 0 || len(in) < sz {
		return v
	}

	if isValidUTF8Prefix(in, sz) {
		return _XX
	}
	return 0
}

func isValidUTF8Prefix[T ~string | ~[]byte](in T, sz int) bool {
	switch x := any(in).(type) {
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
