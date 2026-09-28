package tui

import (
	"testing"

	"clickup-tui/internal/clickup"
)

func TestLookupTaskFromCache(t *testing.T) {
	m := Model{
		taskCache: map[string]clickup.Task{
			"abc": {ID: "abc", Name: "Cached"},
		},
	}
	tasks, ok := m.lookupTask("abc")
	if !ok || tasks.Name != "Cached" {
		t.Fatalf("lookup cache failed")
	}
}

func TestRebuildTaskCache(t *testing.T) {
	m := Model{}
	m.rebuildTaskCache([]clickup.Task{
		{ID: "1", Name: "A"},
		{ID: "2", Name: "B"},
	})
	if len(m.taskCache) != 2 {
		t.Fatalf("cache size %d", len(m.taskCache))
	}
}
