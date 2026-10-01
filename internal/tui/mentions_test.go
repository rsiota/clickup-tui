package tui

import (
	"testing"

	"clickup-tui/internal/clickup"

	"github.com/charmbracelet/bubbles/textarea"
)

func TestActiveMentionQuery(t *testing.T) {
	q, ok := activeMentionQuery("hello @ada")
	if !ok || q != "ada" {
		t.Fatalf("got %q ok=%v", q, ok)
	}
	if _, ok := activeMentionQuery("hello ada"); ok {
		t.Fatal("expected no mention")
	}
	if _, ok := activeMentionQuery("email@x.com"); ok {
		t.Fatal("email local @ should not open mention")
	}
	q, ok = activeMentionQuery("@")
	if !ok || q != "" {
		t.Fatalf("bare @ got %q ok=%v", q, ok)
	}
}

func TestFilterMembers(t *testing.T) {
	members := []clickup.User{
		{ID: 1, Username: "Ada Lovelace", Email: "ada@ex.com"},
		{ID: 2, Username: "Bob", Email: "bob@ex.com"},
	}
	got := filterMembers(members, "ada")
	if len(got) != 1 || got[0].ID != 1 {
		t.Fatalf("got %+v", got)
	}
	got = filterMembers(members, "bob@")
	if len(got) != 1 || got[0].ID != 2 {
		t.Fatalf("email filter got %+v", got)
	}
}

func TestInsertMentionRewritesQuery(t *testing.T) {
	m := &Model{}
	m.comment = textarea.New()
	m.comment.SetValue("Hi @ad")
	// Cursor is at end after SetValue.
	m.insertMention(clickup.User{ID: 9, Username: "Ada Lovelace"})
	if got := m.comment.Value(); got != "Hi @AdaLovelace " {
		t.Fatalf("value=%q", got)
	}
	if m.mentionBindings["adalovelace"] != 9 {
		t.Fatalf("bindings=%v", m.mentionBindings)
	}
}
