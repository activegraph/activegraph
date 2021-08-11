package activesupport

import (
	"regexp"
)

type StringSlice []string

func (ss StringSlice) ToHash() Hash {
	h := make(Hash, len(ss))
	for _, s := range ss {
		h[s] = struct{}{}
	}
	return h
}

type String string

var blankStringRe = regexp.MustCompile(`\A[[:space:]]*\z`)

// IsBlank returns true when string contains only space characters, and false otherwise.
func (s String) IsBlank() bool {
	return blankStringRe.Match([]byte(s))
}
