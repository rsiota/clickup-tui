package tui

import (
	"html"
	"regexp"
	"strings"
)

const maxBodyRunes = 12_000

var (
	htmlMediaTag   = regexp.MustCompile(`(?is)<(img|video|audio|source|iframe|embed|object)[^>]*>`)
	htmlFigure     = regexp.MustCompile(`(?is)<figure[^>]*>.*?</figure>`)
	htmlAttachment = regexp.MustCompile(`(?is)<(?:span|div)[^>]*class="[^"]*attachment[^"]*"[^>]*>.*?</(?:span|div)>`)
	mdImage        = regexp.MustCompile(`!\[[^\]]*\]\([^)]+\)`)
	longURL        = regexp.MustCompile(`https?://[^\s<>"]{120,}`)
)

// sanitizeTaskContent removes embedded media and trims huge HTML for terminal display.
func sanitizeTaskContent(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if looksLikeHTML(s) {
		s = htmlMediaTag.ReplaceAllString(s, "")
		s = htmlFigure.ReplaceAllString(s, "")
		s = htmlAttachment.ReplaceAllString(s, "")
		s = stripHTML(s)
	} else {
		s = mdImage.ReplaceAllString(s, "")
	}
	s = longURL.ReplaceAllString(s, "[link]")
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	runes := []rune(s)
	if len(runes) > maxBodyRunes {
		s = string(runes[:maxBodyRunes]) + "\n\n… (description truncated)"
	}
	return s
}

func sanitizeComment(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	if looksLikeHTML(s) {
		s = htmlMediaTag.ReplaceAllString(s, "")
		s = htmlFigure.ReplaceAllString(s, "")
		s = stripHTML(s)
	} else {
		s = mdImage.ReplaceAllString(s, "")
	}
	return strings.TrimSpace(html.UnescapeString(s))
}
