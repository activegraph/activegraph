package activerecord

type Predicate struct {
	Cond string
	Args []interface{}
}

type joinDependency struct {
	Relation    *Relation
	Association Association
}

type query struct {
	predicates  []Predicate
	groupValues []string
	joinDeps    []joinDependency
}

func (q *query) copy() *query {
	newq := query{
		predicates:  make([]Predicate, len(q.predicates)),
		groupValues: make([]string, len(q.groupValues)),
		joinDeps:    make([]joinDependency, len(q.joinDeps)),
	}

	copy(newq.predicates, q.predicates)
	copy(newq.groupValues, q.groupValues)
	copy(newq.joinDeps, q.joinDeps)

	return &newq
}

func (q *query) where(cond string, args ...interface{}) {
	q.predicates = append(q.predicates, Predicate{cond, args})
}

func (q *query) group(values ...string) {
	q.groupValues = append(q.groupValues, values...)
}

func (q *query) join(rel *Relation, assoc Association) {
	q.joinDeps = append(q.joinDeps, joinDependency{rel, assoc})
}

func (q *query) Dependencies() (dd []Dependency) {
	dd = make([]Dependency, 0, len(q.joinDeps))
	for _, dep := range q.joinDeps {
		dd = append(dd, Dependency{
			TableName:  dep.Relation.TableName(),
			PrimaryKey: dep.Relation.PrimaryKey(),
			ForeignKey: dep.Association.AssociationForeignKey(),
		})
	}
	return dd
}
