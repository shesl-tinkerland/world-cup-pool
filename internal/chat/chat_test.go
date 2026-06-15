package chat

import (
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

func chatMessageRecord(text string) *core.Record {
	collection := core.NewBaseCollection(messagesCollection)
	collection.Fields.Add(&core.TextField{Name: "text", Max: 1000})
	collection.Fields.Add(&core.TextField{Name: "origText", Max: 1000, Hidden: true})
	collection.Fields.Add(&core.BoolField{Name: "deleted"})
	collection.Fields.Add(&core.TextField{Name: "deletedBy", Max: 32})
	collection.Fields.Add(&core.DateField{Name: "deletedAt"})

	rec := core.NewRecord(collection)
	rec.Set("text", text)
	return rec
}

func TestSoftDeleteMessageStoresOriginalAndPlaceholderState(t *testing.T) {
	rec := chatMessageRecord("  trash talk  ")
	now := time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC)

	softDeleteMessage(rec, "u1", now)

	if !rec.GetBool("deleted") {
		t.Fatal("deleted flag was not set")
	}
	if rec.GetString("text") != "" {
		t.Fatalf("text = %q, want blank placeholder text", rec.GetString("text"))
	}
	if rec.GetString("origText") != "  trash talk  " {
		t.Fatalf("origText = %q, want original text", rec.GetString("origText"))
	}
	if rec.GetString("deletedBy") != "u1" {
		t.Fatalf("deletedBy = %q, want u1", rec.GetString("deletedBy"))
	}
	if !rec.GetDateTime("deletedAt").Time().Equal(now) {
		t.Fatalf("deletedAt = %v, want %v", rec.GetDateTime("deletedAt").Time(), now)
	}
}

func TestRestoreMessageMovesOriginalTextBack(t *testing.T) {
	rec := chatMessageRecord("trash talk")
	softDeleteMessage(rec, "u1", time.Now())

	if err := restoreMessage(rec); err != nil {
		t.Fatalf("restoreMessage returned error: %v", err)
	}

	if rec.GetBool("deleted") {
		t.Fatal("deleted flag remained set")
	}
	if rec.GetString("text") != "trash talk" {
		t.Fatalf("text = %q, want original text", rec.GetString("text"))
	}
	if rec.GetString("origText") != "" {
		t.Fatalf("origText = %q, want blank", rec.GetString("origText"))
	}
	if rec.GetString("deletedBy") != "" {
		t.Fatalf("deletedBy = %q, want blank", rec.GetString("deletedBy"))
	}
	if !rec.GetDateTime("deletedAt").Time().IsZero() {
		t.Fatal("deletedAt was not cleared")
	}
}

func TestCleanMessageAllowsDeletedBlankPlaceholder(t *testing.T) {
	rec := chatMessageRecord("")
	rec.Set("deleted", true)

	if err := cleanMessage(rec); err != nil {
		t.Fatalf("cleanMessage returned error for deleted placeholder: %v", err)
	}
}

func TestCleanMessageRejectsBlankActiveMessage(t *testing.T) {
	rec := chatMessageRecord("   ")

	if err := cleanMessage(rec); err == nil {
		t.Fatal("cleanMessage accepted blank active message")
	}
}
