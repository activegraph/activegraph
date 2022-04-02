package activerecord

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/activegraph/activegraph/activesupport"
)

func TestMigrate_AddForeignKey(t *testing.T) {
	EstablishConnection(DatabaseConfig{
		Adapter: "sqlite3", Database: t.Name(),
	})

	defer os.Remove(t.Name())
	defer RemoveConnection("primary")

	// Create two tables with a reference.
	Migrate(t.Name()+"_1", func(m *M) {
		m.CreateTable("owners", func(t *Table) {
			t.String("name")
		})
		m.CreateTable("targets", func(t *Table) {
			t.Int64("value")
			t.References("owners")
		})
	})

	Owner := New("owner", func(r *R) { r.HasOne("target") })
	Target := New("target", func(r *R) { r.BelongsTo("owner") })

	owner := Owner.Create(Hash{"name": "Thom"})
	owner.Expect("owner was not created")
	target := Target.Create(Hash{"owner_id": owner.Unwrap().ID(), "value": 43})
	target.Expect("target was not created")

	// In the next migration we add a foreign key, so the database will
	// maintain the data integrity.
	Migrate(t.Name()+"_2", func(m *M) {
		m.AddForeignKey("targets", "owners")
	})

	// Ensure that records are still there.
	targets, err := Target.All().ToA()
	require.NoError(t, err)
	require.Len(t, targets, 1)
	require.Equal(t, targets[0].Attribute("value"), int64(43))
}
