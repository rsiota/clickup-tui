package tui

import (
	"testing"

	"clickup-tui/internal/clickup"
)

func TestTaskBrowserURL(t *testing.T) {
	if got := taskBrowserURL(clickup.Task{ID: "abc", URL: "https://app.clickup.com/t/abc"}); got != "https://app.clickup.com/t/abc" {
		t.Fatalf("got %q", got)
	}
	if got := taskBrowserURL(clickup.Task{ID: "abc"}); got != "https://app.clickup.com/t/abc" {
		t.Fatalf("fallback got %q", got)
	}
	if got := taskBrowserURL(clickup.Task{}); got != "" {
		t.Fatalf("empty got %q", got)
	}
}

func TestTimeEntryBrowserURL(t *testing.T) {
	e := clickup.TimeEntry{TaskURL: "https://app.clickup.com/t/from-entry", Task: clickup.TimeEntryTask{ID: "x"}}
	if got := timeEntryBrowserURL(e, nil); got != "https://app.clickup.com/t/from-entry" {
		t.Fatalf("entry url got %q", got)
	}
	e = clickup.TimeEntry{Task: clickup.TimeEntryTask{ID: "x"}}
	lookup := func(id string) (clickup.Task, bool) {
		if id == "x" {
			return clickup.Task{ID: "x", URL: "https://app.clickup.com/t/cached"}, true
		}
		return clickup.Task{}, false
	}
	if got := timeEntryBrowserURL(e, lookup); got != "https://app.clickup.com/t/cached" {
		t.Fatalf("cache url got %q", got)
	}
	if got := timeEntryBrowserURL(e, nil); got != "https://app.clickup.com/t/x" {
		t.Fatalf("id fallback got %q", got)
	}
}
