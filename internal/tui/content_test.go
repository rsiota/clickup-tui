package tui

import (
	"strings"
	"testing"
)

func TestSanitizeTaskContentStripsMedia(t *testing.T) {
	in := `<p>Hello</p><img src="https://example.com/huge.png"/><video src="x"></video><p>Bye</p>`
	got := sanitizeTaskContent(in)
	if strings.Contains(got, "img") || strings.Contains(got, "<video") {
		t.Fatalf("media leaked: %q", got)
	}
	if !strings.Contains(got, "Hello") || !strings.Contains(got, "Bye") {
		t.Fatalf("text lost: %q", got)
	}
}

func TestSanitizeMarkdownImage(t *testing.T) {
	in := "See ![screenshot](https://example.com/a.png) for details"
	got := sanitizeTaskContent(in)
	if strings.Contains(got, "![") {
		t.Fatalf("markdown image remained: %q", got)
	}
	if !strings.Contains(got, "See") {
		t.Fatalf("text lost: %q", got)
	}
}
