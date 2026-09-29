package tui

import (
	"testing"

	"clickup-tui/internal/clickup"
)

func TestTaskHasBody(t *testing.T) {
	if taskHasBody(clickup.Task{Description: "  "}) {
		t.Fatal("whitespace only")
	}
	if !taskHasBody(clickup.Task{Description: "hi"}) {
		t.Fatal("expected body")
	}
}

func TestBumpPrefetchPrioritizesCursor(t *testing.T) {
	m := &Model{prefetchQueue: []string{"a", "b", "c"}}
	m.bumpPrefetch("b")
	if len(m.prefetchQueue) != 3 || m.prefetchQueue[0] != "b" {
		t.Fatalf("queue=%v", m.prefetchQueue)
	}
}
