package activerecord

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/activegraph/activegraph/activesupport"
)

func TestActiveRecord_HasOne_AssignAssociation(t *testing.T) {
	EstablishConnection(DatabaseConfig{
		Adapter: "sqlite3", Database: t.Name(),
	})

	defer os.Remove(t.Name())
	defer RemoveConnection("primary")

	Migrate(t.Name(), func(m *M) {
		m.CreateTable("owners", func(t *Table) { t.String("name") })
		m.CreateTable("targets", func(t *Table) { t.Int64("value") })
		m.AddForeignKey("targets", "owners")
	})

	Owner := New("owner", func(r *R) { r.HasOne("target") })
	Target := New("target", func(r *R) { r.BelongsTo("owner") })

	// Ensure that relations has required associations.
	require.True(t, Owner.HasAssociation("target"))
	require.True(t, Target.HasAssociation("owner"))

	// Assert all parameters of HasOne association type.
	hasOne := Owner.ReflectOnAssociation("target")
	require.NotNil(t, hasOne)
	require.Equal(t, "target", hasOne.AssociationName())
	require.Equal(t, "owner_id", hasOne.AssociationForeignKey())

	// Assert all parameters of BelongsTo association type.
	belongsTo := Target.ReflectOnAssociation("owner")
	require.NotNil(t, belongsTo)
	require.Equal(t, "owner", belongsTo.AssociationName())
	require.Equal(t, "owner_id", belongsTo.AssociationForeignKey())

	// Insert an owner into the database, then create a new unpersisted target.
	owner := Owner.Create(Hash{"name": "Kaneman"})
	target := Target.New(Hash{"value": 42})

	// As long as owner is persisted, target should be inserted into
	// the database after association assignment. Check that foreign
	// key of the target was updated respectively.
	owner = owner.AssignAssociation("target", target)
	owner.Expect("failed to assign target to the persisted owner")

	require.Equal(t, owner.Unwrap().ID(), target.Unwrap().Attribute("owner_id"))

	assoc := target.Association("owner")
	assoc.Expect("failed to access owner association")
	require.Equal(t, owner.Unwrap().ID(), assoc.Unwrap().ID())
}

func TestActiveRecord_HasMany_AssignAssociation(t *testing.T) {
	EstablishConnection(DatabaseConfig{
		Adapter: "sqlite3", Database: t.Name(),
	})

	defer os.Remove(t.Name())
	defer RemoveConnection("primary")

	Migrate(t.Name(), func(m *M) {
		m.CreateTable("owners", func(t *Table) { t.String("name") })
		m.CreateTable("targets", func(t *Table) { t.Int64("value") })
		m.AddForeignKey("targets", "owners")
	})

	Owner := New("owner", func(r *R) { r.HasMany("targets") })
	Target := New("target", func(r *R) { r.BelongsTo("owner") })

	require.True(t, Owner.HasAssociation("targets"))
	require.True(t, Target.HasAssociation("owner"))

	// Assert all parameters of HasMany association type.
	hasMany := Owner.ReflectOnAssociation("targets")
	require.NotNil(t, hasMany)
	require.Equal(t, "target", hasMany.AssociationName())
	require.Equal(t, "owner_id", hasMany.AssociationForeignKey())

	// Assert all parameters of BelongsTo association type.
	belongsTo := Target.ReflectOnAssociation("owner")
	require.NotNil(t, belongsTo)
	require.Equal(t, "owner", belongsTo.AssociationName())
	require.Equal(t, "owner_id", belongsTo.AssociationForeignKey())

	owner := Owner.Create(Hash{"name": "Taleb"})
	target1 := Target.New(Hash{"value": 3})
	target2 := Target.New(Hash{"value": 4})
	target3 := Target.New(Hash{"value": 5})

	owner = owner.AssignCollection("targets", target1, target2)
	owner.Expect("failed to assign targets to the persisted owner")

	targets, err := owner.Collection("targets").ToA()
	require.NoError(t, err)
	require.Len(t, targets, 2)

	for i := 0; i < len(targets); i++ {
		require.Equal(t, owner.Unwrap().ID(), targets[i].Attribute("owner_id"))
	}

	owner = owner.AssignCollection("targets", target3)
	owner.Expect("failed to update targets of the owner")

	targets, err = owner.Collection("targets").ToA()
	require.NoError(t, err)
	require.Len(t, targets, 1)
}
