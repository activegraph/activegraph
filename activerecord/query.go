package activerecord

import (
	"fmt"
	"strings"
)

type Predicate struct {
	Cond string
	Args []interface{}
}

type join struct {
	Relation    *Relation
	Association Association
}

type QueryBuilder struct {
	from  string
	limit *int

	selectValues []string
	whereValues  []Predicate
	groupValues  []string
	joinValues   []join
}

func (q *QueryBuilder) copy() *QueryBuilder {
	newq := QueryBuilder{
		from:         q.from,
		limit:        q.limit,
		selectValues: make([]string, len(q.selectValues)),
		whereValues:  make([]Predicate, len(q.whereValues)),
		groupValues:  make([]string, len(q.groupValues)),
		joinValues:   make([]join, len(q.joinValues)),
	}

	copy(newq.selectValues, q.selectValues)
	copy(newq.whereValues, q.whereValues)
	copy(newq.groupValues, q.groupValues)
	copy(newq.joinValues, q.joinValues)

	return &newq
}

func (q *QueryBuilder) From(from string) {
	q.from = from
}

func (q *QueryBuilder) Select(columns ...string) {
	q.selectValues = append(q.selectValues, columns...)
}

func (q *QueryBuilder) Where(cond string, args ...interface{}) {
	q.whereValues = append(q.whereValues, Predicate{cond, args})
}

func (q *QueryBuilder) Group(values ...string) {
	q.groupValues = append(q.groupValues, values...)
}

func (q *QueryBuilder) Join(rel *Relation, assoc Association) {
	q.joinValues = append(q.joinValues, join{rel, assoc})
}

func (q *QueryBuilder) Limit(num int) {
	q.limit = &num
}

func (q *QueryBuilder) String() string {
	if q.from == "" {
		panic("from is not set")
	}

	var (
		buf          strings.Builder
		selectValues = q.selectValues
	)

	if len(selectValues) == 0 {
		selectValues = []string{"*"}
	}
	fmt.Fprintf(&buf, `SELECT %s FROM "%s"`, strings.Join(selectValues, ", "), q.from)

	for _, join := range q.joinValues {
		var (
			on = join.Relation.TableName()
			pk = join.Relation.PrimaryKey()
			fk = join.Association.AssociationForeignKey()
		)

		fmt.Fprintf(&buf, ` INNER JOIN "%s" ON `, on)
		fmt.Fprintf(&buf, `%s.%s = %s.%s `, q.from, fk, on, pk)
	}

	if len(q.whereValues) > 0 {
		fmt.Fprintf(&buf, ` WHERE`)
	}

	for i, where := range q.whereValues {
		if i < len(q.whereValues)-1 {
			fmt.Fprintf(&buf, ` AND`)
		}
		fmt.Fprintf(&buf, ` (%s)`, where.Cond)
	}

	if len(q.groupValues) > 0 {
		fmt.Fprintf(&buf, ` GROUP BY %s`, strings.Join(q.groupValues, ", "))
	}
	if q.limit != nil {
		fmt.Fprintf(&buf, ` LIMIT %d`, *q.limit)
	}

	return buf.String()
}

func (q *QueryBuilder) Args() []interface{} {
	args := make([]interface{}, 0, len(q.whereValues))
	for i := range q.whereValues {
		args = append(args, q.whereValues[i].Args...)
	}
	return args
}

func (q *QueryBuilder) Operation() *QueryOperation {
	return &QueryOperation{
		Text:    q.String(),
		Args:    q.Args(),
		Columns: q.selectValues,
	}
}
