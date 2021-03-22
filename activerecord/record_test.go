package activerecord_test

import (
	"context"
	"testing"

	"github.com/activegraph/activegraph/activerecord"
	"github.com/activegraph/activegraph/activerecord/sqlite3"
)

func TestActiveRecord_Insert(t *testing.T) {
	schema := activerecord.New(
		"book",
		activerecord.StringAttr{Name: "title"},
		activerecord.IntAttr{Name: "pages"},
	)

	schema.Connect(sqlite3.Open())

	book := schema.New(map[string]interface{}{
		"title": "Moby Dick", "pages": 146,
	})

	t.Logf("id is %v", book.ID())

	if !book.HasAttribute("title") {
		t.Fatal("expected attribute 'title'")
	}

	_, err := book.Insert(context.TODO())
	if err != nil {
		t.Fatalf("%s", err)
	}
}
