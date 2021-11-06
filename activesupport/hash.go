package activesupport

type Hash map[string]interface{}

func (h Hash) HasKey(k string) bool {
	_, ok := h[k]
	return ok
}

func (h Hash) IsEmpty() bool {
	return len(h) == 0
}

func (h Hash) ToHash() Hash {
	return h
}

func (h Hash) Copy() Hash {
	hcopy := make(Hash, len(h))
	for k, v := range h {
		hcopy[k] = v
	}
	return hcopy
}

// Slice returns a new Hash containing the entries for the given keys.
// Any given keys that are not found are ignored.
func (h Hash) Slice(keys ...string) Hash {
	hcopy := make(Hash, len(keys))
	for i := range keys {
		if v, ok := h[keys[i]]; ok {
			hcopy[keys[i]] = v
		}
	}
	return hcopy
}

func (h Hash) Merged(others ...Hash) Hash {
	hcopy := h.Copy()
	for _, other := range others {
		for k, v := range other {
			hcopy[k] = v
		}
	}
	return hcopy
}

func (h Hash) Merge(others ...Hash) Hash {
	for _, other := range others {
		for k, v := range other {
			h[k] = v
		}
	}
	return h
}

type HashConverter interface {
	ToHash() Hash
}

type HashArrayConverter interface {
	ToHashArray() []Hash
}
