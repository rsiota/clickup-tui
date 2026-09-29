package clickup

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

const pageSize = 100

type SearchOptions struct {
	Assignees     []string
	DueAfter      time.Time
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
	if !opts.DueAfter.IsZero() {
		// due_date_gt is exclusive; subtract 1ms so the start day is included.
		params.Set("due_date_gt", strconv.FormatInt(opts.DueAfter.UnixMilli()-1, 10))
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

// WeekBounds returns Monday 00:00 and next Monday 00:00 in now's location.
func WeekBounds(now time.Time) (time.Time, time.Time) {
	start := startOfWeek(now)
	return start, start.AddDate(0, 0, 7)
}

func startOfWeek(now time.Time) time.Time {
	loc := now.Location()
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	// Monday = 0 … Sunday = 6
	offset := (int(day.Weekday()) + 6) % 7
	return day.AddDate(0, 0, -offset)
}

// WeekTasks returns open tasks assigned to the user that are due this week (Mon–Sun).
func (c *Client) WeekTasks(ctx context.Context, workspace string, userID int, now time.Time) ([]Task, error) {
	start, end := WeekBounds(now)
	var all []Task
	for page := 0; page < 10; page++ {
		tasks, err := c.SearchTasks(ctx, workspace, SearchOptions{
			Assignees: []string{strconv.Itoa(userID)},
			DueAfter:  start,
			DueBefore: end,
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

type GetTaskOptions struct {
	// IncludeMarkdown asks ClickUp for markdown_description (large; usually unnecessary in the TUI).
	IncludeMarkdown bool
}

func (c *Client) GetTask(ctx context.Context, workspace, id string, opts ...GetTaskOptions) (*Task, error) {
	path := fmt.Sprintf("/v2/task/%s", id) + taskScopeQuery(id, workspace)
	includeMD := false
	if len(opts) > 0 && opts[0].IncludeMarkdown {
		includeMD = true
	}
	q := url.Values{}
	if includeMD {
		q.Set("include_markdown_description", "true")
	}
	if enc := q.Encode(); enc != "" {
		if strings.Contains(path, "?") {
			path += "&" + enc
		} else {
			path += "?" + enc
		}
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

// ListStatusesResponse is the subset of GET /list/{id} we need for status pickers.
type ListStatusesResponse struct {
	ID       string       `json:"id"`
	Statuses []ListStatus `json:"statuses"`
}

type ListStatus struct {
	Status     string  `json:"status"`
	Color      string  `json:"color"`
	Type       string  `json:"type"`
	OrderIndex float64 `json:"orderindex"`
}

func (s ListStatus) TaskStatus() TaskStatus {
	return TaskStatus{Status: s.Status, Color: s.Color, Type: s.Type}
}

// GetListStatuses returns the status pipeline for a list, ordered by orderindex.
func (c *Client) GetListStatuses(ctx context.Context, listID string) ([]ListStatus, error) {
	if listID == "" {
		return nil, fmt.Errorf("empty list id")
	}
	var resp ListStatusesResponse
	if err := c.do(ctx, http.MethodGet, "/v2/list/"+listID, nil, &resp); err != nil {
		return nil, err
	}
	statuses := append([]ListStatus(nil), resp.Statuses...)
	sort.SliceStable(statuses, func(i, j int) bool {
		return statuses[i].OrderIndex < statuses[j].OrderIndex
	})
	return statuses, nil
}

// UpdateTaskStatus sets a task's status and returns the updated task.
func (c *Client) UpdateTaskStatus(ctx context.Context, workspace, taskID, status string) (*Task, error) {
	path := fmt.Sprintf("/v2/task/%s", taskID) + taskScopeQuery(taskID, workspace)
	body := map[string]string{"status": status}
	var task Task
	if err := c.do(ctx, http.MethodPut, path, body, &task); err != nil {
		return nil, err
	}
	return &task, nil
}
