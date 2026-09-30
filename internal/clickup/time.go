package clickup

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func (c *Client) TimeEntries(ctx context.Context, workspace string, start, end time.Time, taskID string) ([]TimeEntry, error) {
	q := url.Values{}
	if !start.IsZero() {
		q.Set("start_date", strconv.FormatInt(start.UnixMilli(), 10))
	}
	if !end.IsZero() {
		q.Set("end_date", strconv.FormatInt(end.UnixMilli(), 10))
	}
	if taskID != "" {
		q.Set("task_id", taskID)
		if LooksCustomID(taskID) {
			q.Set("custom_task_ids", "true")
			q.Set("team_id", workspace)
		}
	}
	path := fmt.Sprintf("/v2/team/%s/time_entries", workspace)
	if enc := q.Encode(); enc != "" {
		path += "?" + enc
	}
	var resp TimeEntriesResponse
	if err := c.do(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func (c *Client) DayEntries(ctx context.Context, workspace string, day time.Time) ([]TimeEntry, error) {
	start, end := dayBounds(day)
	return c.TimeEntries(ctx, workspace, start, end, "")
}

// CreateTimeEntry logs duration on taskID. If start is zero, the entry ends now
// (start = now − duration). Otherwise start is used as the entry start time.
func (c *Client) CreateTimeEntry(ctx context.Context, workspace, taskID string, duration time.Duration, start time.Time, description string) (*TimeEntry, error) {
	if duration <= 0 {
		return nil, fmt.Errorf("duration must be positive")
	}
	if start.IsZero() {
		start = time.Now().Add(-duration)
	}
	path := fmt.Sprintf("/v2/team/%s/time_entries", workspace)
	if LooksCustomID(taskID) {
		q := url.Values{}
		q.Set("custom_task_ids", "true")
		q.Set("team_id", workspace)
		path += "?" + q.Encode()
	}
	req := CreateTimeEntryRequest{
		Description: strings.TrimSpace(description),
		Start:       start.UnixMilli(),
		Duration:    duration.Milliseconds(),
		Tid:         taskID,
	}
	var raw json.RawMessage
	if err := c.do(ctx, http.MethodPost, path, req, &raw); err != nil {
		return nil, err
	}
	var wrapped SingleTimeEntryResponse
	if json.Unmarshal(raw, &wrapped) == nil && wrapped.Data.ID != "" {
		return &wrapped.Data, nil
	}
	var entry TimeEntry
	if json.Unmarshal(raw, &entry) == nil && entry.ID != "" {
		return &entry, nil
	}
	return nil, fmt.Errorf("clickup: empty time entry response")
}

// TimeEntryUpdate is a partial update for an existing time entry.
type TimeEntryUpdate struct {
	Duration    *time.Duration
	Start       *time.Time
	Description *string
}

func (c *Client) UpdateTimeEntry(ctx context.Context, workspace, entryID string, upd TimeEntryUpdate) error {
	req := UpdateTimeEntryRequest{}
	if upd.Duration != nil {
		ms := upd.Duration.Milliseconds()
		req.Duration = &ms
	}
	if upd.Start != nil {
		ms := upd.Start.UnixMilli()
		req.Start = &ms
	}
	if upd.Description != nil {
		req.Description = upd.Description
	}
	if req.Duration == nil && req.Start == nil && req.Description == nil {
		return fmt.Errorf("nothing to update")
	}
	return c.do(ctx, http.MethodPut, fmt.Sprintf("/v2/team/%s/time_entries/%s", workspace, entryID), req, nil)
}

func (c *Client) DeleteTimeEntry(ctx context.Context, workspace, entryID string) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/v2/team/%s/time_entries/%s", workspace, entryID), nil, nil)
}

func dayBounds(day time.Time) (time.Time, time.Time) {
	loc := day.Location()
	start := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, loc)
	return start, start.Add(24 * time.Hour)
}
