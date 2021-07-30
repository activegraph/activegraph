package activesupport

type Hash map[string]interface{}

func (h Hash) HasKey(k string) bool {
	_, ok := h[k]
	return ok
}

func (h Hash) IsEmpty() bool {
	return len(h) == 0
}

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
