package activerecord

type WhereChain struct {
	predicates []string
	args       []interface{}
}

func (wc *WhereChain) copy() *WhereChain {
	// TODO: implement shallow copy.
	return wc
}
