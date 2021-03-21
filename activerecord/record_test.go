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
		activerecord.Attr("title", activerecord.String),
		activerecord.Attr("pages", activerecord.Int),
	)

	schema.Connect(sqlite3.Open())

	book := schema.New(map[string]interface{}{
		"title": "Moby Dick", "pages": 146,
	})

	if !book.HasAttribute("title") {
		t.Fatal("expected attribute 'title'")
	}

	_, err := book.Insert(context.TODO())
	if err != nil {
		t.Fatalf("%s", err)
	}
}
