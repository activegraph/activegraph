package activesupport

type StringSlice []string

func (ss StringSlice) ToHash() Hash {
	h := make(Hash, len(ss))
	for _, s := range ss {
		h[s] = struct{}{}
	}
	return h
}
