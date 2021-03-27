package activerecord

type Predicate struct {
	Cond string
	Args []interface{}
}

type query struct {
	predicates  []Predicate
	groupValues []string
}

func (q *query) copy() *query {
	newq := query{
		predicates:  make([]Predicate, len(q.predicates)),
		groupValues: make([]string, len(q.groupValues)),
	}

	copy(newq.predicates, q.predicates)
	copy(newq.groupValues, q.groupValues)

	return &newq
}

func (q *query) where(cond string, args ...interface{}) {
	q.predicates = append(q.predicates, Predicate{cond, args})
}

func (q *query) group(values ...string) {
	q.groupValues = append(q.groupValues, values...)
}
