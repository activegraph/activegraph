package activesupport

import (
	"regexp"
)

type StringSlice []string

func Strings(ss ...string) StringSlice {
	return StringSlice(ss)
}

func (ss StringSlice) ToHash() Hash {
	h := make(Hash, len(ss))
	for _, s := range ss {
		h[s] = struct{}{}
	}
	return h
}

// Contains returns true if the slice contains an element with the given value.
func (ss StringSlice) Contains(v interface{}) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}
	for i := range ss {
		if ss[i] == s {
			return true
		}
	}
	return false
}

func (ss StringSlice) Find(pred func(String) bool) String {
	for i := range ss {
		if pred(String(ss[i])) {
			return String(ss[i])
		}
	}
	return String("")
}

type String string

var blankStringRe = regexp.MustCompile(`\A[[:space:]]*\z`)

// IsBlank returns true when string contains only space characters, and false otherwise.
func (s String) IsBlank() bool {
	return blankStringRe.Match([]byte(s))
}

// IsEmpty returns true when lenght of string is zero, and false otherwise.
func (s String) IsEmpty() bool {
	return len(s) == 0
}

func (s String) IsNotEmpty() bool {
	return len(s) != 0
}
