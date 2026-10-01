package clickup

import (
	"strings"
	"testing"
)

func TestMentionTokenStripsSpaces(t *testing.T) {
	if got := MentionToken(User{Username: "Ada Lovelace", ID: 1}); got != "AdaLovelace" {
		t.Fatalf("got %q", got)
	}
	if got := MentionToken(User{Email: "bob@example.com", ID: 2}); got != "bob" {
		t.Fatalf("email token got %q", got)
	}
}

func TestBuildCommentSegmentsTagsUsers(t *testing.T) {
	members := []User{
		{ID: 10, Username: "Ada Lovelace"},
		{ID: 11, Username: "Bob"},
	}
	bindings := map[string]int{"adalovelace": 10}
	segs := BuildCommentSegments("Hi @AdaLovelace and @Bob please look", bindings, members)
	if len(segs) < 3 {
		t.Fatalf("segs=%+v", segs)
	}
	var tags int
	for _, s := range segs {
		if s.Type == "tag" {
			tags++
			if s.User == nil || (s.User.ID != 10 && s.User.ID != 11) {
				t.Fatalf("bad tag %+v", s)
			}
		}
	}
	if tags != 2 {
		t.Fatalf("want 2 tags, got %d in %+v", tags, segs)
	}
	plain := PlainCommentText(segs)
	if !strings.Contains(plain, "Hi") || !strings.Contains(plain, "please look") {
		t.Fatalf("plain=%q", plain)
	}
}

func TestBuildCommentSegmentsLeavesUnknownAt(t *testing.T) {
	segs := BuildCommentSegments("see @nobody here", nil, nil)
	if len(segs) != 1 || segs[0].Text != "see @nobody here" {
		t.Fatalf("segs=%+v", segs)
	}
}
