package activerecord

type Predicate struct {
	Cond string
	Args []interface{}
}

type Dependency struct {
	TableName  string
	ForeignKey string
	PrimaryKey string
}

type query struct {
	predicates   []Predicate
	groupValues  []string
	dependencies []Dependency
}

func (q *query) copy() *query {
	newq := query{
		predicates:   make([]Predicate, len(q.predicates)),
		groupValues:  make([]string, len(q.groupValues)),
		dependencies: make([]Dependency, len(q.dependencies)),
	}

	copy(newq.predicates, q.predicates)
	copy(newq.groupValues, q.groupValues)
	copy(newq.dependencies, q.dependencies)

	return &newq
}

func (q *query) where(cond string, args ...interface{}) {
	q.predicates = append(q.predicates, Predicate{cond, args})
}

func (q *query) group(values ...string) {
	q.groupValues = append(q.groupValues, values...)
}

func (q *query) join(name, fk, pk string) {
	q.dependencies = append(q.dependencies, Dependency{name, fk, pk})
}
