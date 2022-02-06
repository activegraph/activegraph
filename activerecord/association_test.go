package activerecord

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/activegraph/activegraph/activesupport"
)

func TestActiveRecord_Persisted_AssignAssociation(t *testing.T) {
	EstablishConnection(DatabaseConfig{
		Adapter: "sqlite3", Database: t.Name() + ".db",
	})

	defer os.Remove(t.Name() + ".db")
	defer RemoveConnection("primary")

	Migrate(t.Name(), func(m *M) {
		m.CreateTable("owners", func(t *Table) { t.String("name") })
		m.CreateTable("targets", func(t *Table) { t.Int64("value") })
		m.AddForeignKey("targets", "owners")
	})

	Owner := New("owner", func(r *R) { r.HasOne("target") })
	Target := New("target", func(r *R) { r.BelongsTo("owner") })

	// Insert an owner into the database, then create a new unpersisted target.
	owner := Owner.Create(Hash{"name": "Kaneman"})
	target := Target.New(Hash{"value": 42})

	owner = owner.AssignAssociation("target", target)
	owner.Expect("failed to assign target to the persisted owner")

	assoc := target.Association("owner")
	assoc.Expect("failed to access owner association")
	require.Equal(t, owner.Unwrap().ID(), assoc.Unwrap().ID())
}
