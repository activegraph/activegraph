package activesupport

type Hash map[string]interface{}

func (h Hash) Copy() Hash {
	hcopy := make(Hash, len(h))
	for k, v := range h {
		hcopy[k] = v
	}
	return hcopy
}

type HashConverter interface {
	ToHash() Hash
}

type HashArrayConverter interface {
	ToHashArray() []Hash
}
