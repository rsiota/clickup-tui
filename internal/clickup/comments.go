package clickup

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"unicode"
)

func (c *Client) ListComments(ctx context.Context, workspace, taskID string) ([]Comment, error) {
	path := fmt.Sprintf("/v2/task/%s/comment", taskID) + taskScopeQuery(taskID, workspace)
	var resp CommentsResponse
	if err := c.do(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Comments, nil
}

func (c *Client) CreateComment(ctx context.Context, workspace, taskID, text string) error {
	return c.CreateCommentRich(ctx, workspace, taskID, []CommentSegment{{Text: text}}, text)
}

// CreateCommentRich posts a structured comment (supports @tags).
func (c *Client) CreateCommentRich(ctx context.Context, workspace, taskID string, segments []CommentSegment, plain string) error {
	path := fmt.Sprintf("/v2/task/%s/comment", taskID) + taskScopeQuery(taskID, workspace)
	if plain == "" {
		plain = PlainCommentText(segments)
	}
	req := CreateCommentRequest{CommentText: plain}
	for _, s := range segments {
		if s.Type == "tag" {
			req.Comment = segments
			break
		}
	}
	return c.do(ctx, http.MethodPost, path, req, nil)
}

// MentionToken is the @handle inserted in the TUI for a user (no spaces).
func MentionToken(u User) string {
	name := strings.TrimSpace(u.Username)
	if name == "" {
		name = strings.TrimSpace(u.Email)
		if i := strings.IndexByte(name, '@'); i > 0 {
			name = name[:i]
		}
	}
	var b strings.Builder
	for _, r := range name {
		if unicode.IsSpace(r) {
			continue
		}
		b.WriteRune(r)
	}
	if b.Len() == 0 {
		return fmt.Sprintf("user%d", u.ID)
	}
	return b.String()
}

// ResolveMention maps an @token to a user id using explicit bindings, then members.
func ResolveMention(token string, bindings map[string]int, members []User) (int, bool) {
	token = strings.TrimSpace(token)
	if token == "" {
		return 0, false
	}
	key := strings.ToLower(token)
	if id, ok := bindings[key]; ok && id != 0 {
		return id, true
	}
	var prefix *User
	for i := range members {
		u := &members[i]
		if u.ID == 0 {
			continue
		}
		if strings.EqualFold(MentionToken(*u), token) ||
			strings.EqualFold(strings.TrimSpace(u.Username), token) ||
			strings.EqualFold(strings.TrimSpace(u.Email), token) {
			return u.ID, true
		}
		emailLocal := u.Email
		if i := strings.IndexByte(emailLocal, '@'); i > 0 {
			emailLocal = emailLocal[:i]
		}
		if strings.EqualFold(emailLocal, token) {
			return u.ID, true
		}
		compact := strings.ToLower(MentionToken(*u))
		if strings.HasPrefix(compact, key) || strings.HasPrefix(strings.ToLower(u.Username), key) {
			if prefix != nil && prefix.ID != u.ID {
				prefix = nil // ambiguous
				break
			}
			if prefix == nil {
				prefix = u
			}
		}
	}
	if prefix != nil {
		return prefix.ID, true
	}
	return 0, false
}

// BuildCommentSegments turns plain text with @handles into ClickUp comment segments.
func BuildCommentSegments(text string, bindings map[string]int, members []User) []CommentSegment {
	if text == "" {
		return nil
	}
	var segs []CommentSegment
	var plain strings.Builder
	runes := []rune(text)
	flush := func() {
		if plain.Len() == 0 {
			return
		}
		segs = append(segs, CommentSegment{Text: plain.String()})
		plain.Reset()
	}
	for i := 0; i < len(runes); {
		if runes[i] != '@' {
			plain.WriteRune(runes[i])
			i++
			continue
		}
		if i > 0 && (unicode.IsLetter(runes[i-1]) || unicode.IsDigit(runes[i-1])) {
			plain.WriteRune('@')
			i++
			continue
		}
		j := i + 1
		for j < len(runes) && !unicode.IsSpace(runes[j]) && runes[j] != '@' {
			j++
		}
		if j == i+1 {
			plain.WriteRune('@')
			i++
			continue
		}
		token := string(runes[i+1 : j])
		if id, ok := ResolveMention(token, bindings, members); ok {
			flush()
			segs = append(segs, CommentSegment{
				Type: "tag",
				User: &CommentUser{ID: id},
			})
			i = j
			continue
		}
		plain.WriteString(string(runes[i:j]))
		i = j
	}
	flush()
	if len(segs) == 0 {
		return []CommentSegment{{Text: text}}
	}
	return segs
}

func PlainCommentText(segs []CommentSegment) string {
	var b strings.Builder
	for _, s := range segs {
		if s.Type == "tag" && s.User != nil {
			b.WriteString("@user")
			continue
		}
		b.WriteString(s.Text)
	}
	return b.String()
}

// WorkspaceMembers returns users for a workspace id from Get Authorized Teams.
func (c *Client) WorkspaceMembers(ctx context.Context, workspaceID string) ([]User, error) {
	teams, err := c.ListWorkspaces(ctx)
	if err != nil {
		return nil, err
	}
	for _, t := range teams {
		if t.ID.String() == workspaceID {
			return usersFromMembers(t.Members), nil
		}
	}
	return nil, fmt.Errorf("workspace %s not found", workspaceID)
}

func usersFromMembers(members []WorkspaceMember) []User {
	out := make([]User, 0, len(members))
	seen := map[int]bool{}
	for _, m := range members {
		u := m.User
		if u.ID == 0 || seen[u.ID] {
			continue
		}
		seen[u.ID] = true
		out = append(out, u)
	}
	return out
}
