package clickup

import (
	"context"
	"fmt"
	"net/http"
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
	path := fmt.Sprintf("/v2/task/%s/comment", taskID) + taskScopeQuery(taskID, workspace)
	req := CreateCommentRequest{CommentText: text}
	return c.do(ctx, http.MethodPost, path, req, nil)
}
