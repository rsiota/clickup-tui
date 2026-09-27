package clickup

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const pageSize = 100

type SearchOptions struct {
	Assignees     []string
	DueBefore     time.Time
	IncludeClosed bool
	Subtasks      bool
	Page          int
}

func (c *Client) SearchTasks(ctx context.Context, workspace string, opts SearchOptions) ([]Task, error) {
	params := url.Values{}
	if opts.Page > 0 {
		params.Set("page", strconv.Itoa(opts.Page))
	}
	if opts.IncludeClosed {
		params.Set("include_closed", "true")
	}
	if opts.Subtasks {
		params.Set("subtasks", "true")
	}
	for _, a := range opts.Assignees {
		params.Add("assignees[]", a)
	}
	if !opts.DueBefore.IsZero() {
		params.Set("due_date_lt", strconv.FormatInt(opts.DueBefore.UnixMilli(), 10))
	}
	params.Set("order_by", "due_date")

	path := fmt.Sprintf("/v2/team/%s/task", workspace)
	if q := params.Encode(); q != "" {
		path += "?" + q
	}
	var resp TasksResponse
	if err := c.do(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Tasks, nil
}

// TodayTasks returns open tasks assigned to the user that are due today or overdue.
func (c *Client) TodayTasks(ctx context.Context, workspace string, userID int, now time.Time) ([]Task, error) {
	loc := now.Location()
	startOfTomorrow := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).Add(24 * time.Hour)
	var all []Task
	for page := 0; page < 10; page++ {
		tasks, err := c.SearchTasks(ctx, workspace, SearchOptions{
			Assignees: []string{strconv.Itoa(userID)},
			DueBefore: startOfTomorrow,
			Subtasks:  true,
			Page:      page,
		})
		if err != nil {
			return nil, err
		}
		all = append(all, tasks...)
		if len(tasks) < pageSize {
			break
		}
	}
	return all, nil
}

func (c *Client) GetTask(ctx context.Context, workspace, id string) (*Task, error) {
	path := fmt.Sprintf("/v2/task/%s", id) + taskScopeQuery(id, workspace)
	if !strings.Contains(path, "?") {
		path += "?include_markdown_description=true"
	} else {
		path += "&include_markdown_description=true"
	}
	var task Task
	if err := c.do(ctx, http.MethodGet, path, nil, &task); err != nil {
		return nil, err
	}
	return &task, nil
}

// FindTasks returns assigned open tasks whose name, id, or custom id contains query.
// If query looks like a task id, GetTask is tried first.
func (c *Client) FindTasks(ctx context.Context, workspace string, userID int, query string) ([]Task, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("empty search")
	}

	if task, err := c.GetTask(ctx, workspace, query); err == nil {
		return []Task{*task}, nil
	}

	needle := strings.ToLower(query)
	var matches []Task
	for page := 0; page < 5; page++ {
		tasks, err := c.SearchTasks(ctx, workspace, SearchOptions{
			Assignees: []string{strconv.Itoa(userID)},
			Subtasks:  true,
			Page:      page,
		})
		if err != nil {
			return nil, err
		}
		for _, t := range tasks {
			if taskMatches(t, needle) {
				matches = append(matches, t)
			}
		}
		if len(tasks) < pageSize {
			break
		}
	}
	return matches, nil
}

func taskMatches(t Task, needle string) bool {
	if strings.Contains(strings.ToLower(t.Name), needle) {
		return true
	}
	if strings.Contains(strings.ToLower(t.ID), needle) {
		return true
	}
	return strings.Contains(strings.ToLower(t.CustomID), needle)
}
