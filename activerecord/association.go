package activerecord

type Association interface {
	AssociationName() string
}

type BelongsToAssoc struct {
	Name string
}

func (a *BelongsToAssoc) AssociationName() string {
	return a.Name
}

type HasManyAssoc struct {
	Name string
}

func (a *HasManyAssoc) AssociationName() string {
	return a.Name
}

type associationsMap map[string]Association

func (m associationsMap) Copy() associationsMap {
	mm := make(associationsMap, len(m))
	for name, assoc := range m {
		mm[name] = assoc
	}
	return mm
}

type associations struct {
}

func (a *associations) AssignAssociation(assocName string, ar *ActiveRecord) error {
	return nil
}

func (a *associations) AccessAssociation(assocName string) (*ActiveRecord, error) {
	return nil, nil
}
