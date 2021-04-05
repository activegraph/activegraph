package activerecord

import (
	"fmt"
	"strings"
)

type ErrUnknownAssociation struct {
	RecordName string
	Assoc      string
}

func (e *ErrUnknownAssociation) Error() string {
	return fmt.Sprintf("unknown association %q for %s", e.Assoc, e.RecordName)
}

type Association interface {
	AssociationName() string
	AssociationForeignKey() string
}

type BelongsTo struct {
	name       string
	foreignKey string
}

func (a *BelongsTo) AssociationName() string {
	return a.name
}

// ForeignKey sets the foreign key used for the association. By default this is
// guessed to be the name of this relation in lower-case and "_id" suffixed.
//
// So a relation that defines a BelongsTo("person") association will use "person_id"
// as the default foreign key.
func (a *BelongsTo) ForeignKey(name string) {
	a.foreignKey = name
}

func (a *BelongsTo) AssociationForeignKey() string {
	if a.foreignKey != "" {
		return a.foreignKey
	}
	return strings.ToLower(a.name) + "_" + defaultPrimaryKeyName
}

type HasMany struct {
	name       string
	foreignKey string
}

func (a *HasMany) AssociationName() string {
	return a.name
}

func (a *HasMany) AssociationForeignKey() string {
	if a.foreignKey != "" {
		return a.foreignKey
	}
	return strings.ToLower(a.name) + "_" + defaultPrimaryKeyName
}

type associationsMap map[string]Association

func (m associationsMap) copy() associationsMap {
	mm := make(associationsMap, len(m))
	for name, assoc := range m {
		mm[name] = assoc
	}
	return mm
}

type associations struct {
	recordName string
	reflection *Reflection
	keys       associationsMap
	values     map[string]*ActiveRecord
}

func newAssociations(recordName string, assocs associationsMap, reflection *Reflection) *associations {
	return &associations{
		recordName: recordName,
		reflection: reflection,
		keys:       assocs,
		values:     make(map[string]*ActiveRecord),
	}
}

func (a *associations) copy() *associations {
	values := make(map[string]*ActiveRecord, len(a.values))
	for k, v := range a.values {
		values[k] = v
	}
	return &associations{
		recordName: a.recordName,
		reflection: a.reflection,
		keys:       a.keys.copy(),
		values:     values,
	}
}

func (a *associations) HasAssociation(assocName string) bool {
	_, ok := a.keys[assocName]
	return ok
}

func (a *associations) HasAssociations(assocNames ...string) bool {
	for _, assocName := range assocNames {
		if !a.HasAssociation(assocName) {
			return false
		}
	}
	return true
}

func (a *associations) get(assocName string) Association {
	if !a.HasAssociation(assocName) {
		return nil
	}
	return a.keys[assocName]
}

func (a *associations) AccessAssociation(assocName string) *ActiveRecord {
	if !a.HasAssociation(assocName) {
		return nil
	}
	return a.values[assocName]
}

func (a *associations) AssignAssociation(assocName string, val *ActiveRecord) error {
	_, ok := a.keys[assocName]
	if !ok {
		return &ErrUnknownAssociation{RecordName: a.recordName, Assoc: assocName}
	}

	a.values[assocName] = val
	return nil
}
