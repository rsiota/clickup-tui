package clickup

import (
	"encoding/json"
	"strings"
	"time"
)

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

type UserResponse struct {
	User User `json:"user"`
}

type Workspace struct {
	ID   FlexString `json:"id"`
	Name string     `json:"name"`
}

type WorkspacesResponse struct {
	Teams []Workspace `json:"teams"`
}

type TaskStatus struct {
	Status string `json:"status"`
	Color  string `json:"color"`
	Type   string `json:"type"`
}

type TaskPriority struct {
	ID       FlexString `json:"id"`
	Priority string     `json:"priority"`
	Color    string     `json:"color"`
}

type Task struct {
	ID                  string        `json:"id"`
	CustomID            string        `json:"custom_id,omitempty"`
	Name                string        `json:"name"`
	Description         string        `json:"description,omitempty"`
	MarkdownDescription string        `json:"markdown_description,omitempty"`
	Status              TaskStatus    `json:"status"`
	DateCreated         FlexInt64     `json:"date_created"`
	DateUpdated         FlexInt64     `json:"date_updated"`
	Assignees           []User        `json:"assignees"`
	Priority            *TaskPriority `json:"priority"`
	DueDate             FlexInt64     `json:"due_date"`
	TimeSpent           FlexInt64     `json:"time_spent"`
	URL                 string        `json:"url"`
	List                struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"list"`
	Folder struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Hidden bool   `json:"hidden"`
	} `json:"folder"`
	Space struct {
		ID string `json:"id"`
	} `json:"space"`
}

func (t Task) Ref() string {
	if t.CustomID != "" {
		return t.CustomID
	}
	return t.ID
}

func (t Task) Body() string {
	if strings.TrimSpace(t.MarkdownDescription) != "" {
		return t.MarkdownDescription
	}
	return t.Description
}

func (t Task) DueTime() time.Time {
	if t.DueDate == 0 {
		return time.Time{}
	}
	return time.UnixMilli(t.DueDate.Int64())
}

type TasksResponse struct {
	Tasks []Task `json:"tasks"`
}

type Comment struct {
	ID          FlexString `json:"id"`
	CommentText string     `json:"comment_text"`
	User        User       `json:"user"`
	Date        FlexInt64  `json:"date"`
}

func (c Comment) Time() time.Time {
	if c.Date == 0 {
		return time.Time{}
	}
	return time.UnixMilli(c.Date.Int64())
}

type CommentsResponse struct {
	Comments []Comment `json:"comments"`
}

type CreateCommentRequest struct {
	CommentText string `json:"comment_text"`
	NotifyAll   bool   `json:"notify_all,omitempty"`
}

type TimeEntryTask struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (t *TimeEntryTask) UnmarshalJSON(b []byte) error {
	if string(b) == "null" || string(b) == `""` {
		return nil
	}
	if len(b) > 0 && b[0] == '"' {
		var id string
		if err := json.Unmarshal(b, &id); err != nil {
			return err
		}
		t.ID = id
		return nil
	}
	type alias TimeEntryTask
	return json.Unmarshal(b, (*alias)(t))
}

type TimeEntry struct {
	ID          string        `json:"id"`
	Task        TimeEntryTask `json:"task"`
	Billable    bool          `json:"billable"`
	Start       FlexInt64     `json:"start"`
	End         FlexInt64     `json:"end"`
	Duration    FlexInt64     `json:"duration"`
	Description string        `json:"description"`
	TaskURL     string        `json:"task_url,omitempty"`
}

func (e TimeEntry) StartTime() time.Time {
	if e.Start == 0 {
		return time.Time{}
	}
	return time.UnixMilli(e.Start.Int64())
}

func (e TimeEntry) Running() bool {
	return e.Duration.Int64() < 0
}

type TimeEntriesResponse struct {
	Data []TimeEntry `json:"data"`
}

type SingleTimeEntryResponse struct {
	Data TimeEntry `json:"data"`
}

type CreateTimeEntryRequest struct {
	Description string `json:"description,omitempty"`
	Start       int64  `json:"start"`
	Duration    int64  `json:"duration"`
	Tid         string `json:"tid,omitempty"`
	Billable    bool   `json:"billable,omitempty"`
}

type UpdateTimeEntryRequest struct {
	Duration *int64 `json:"duration,omitempty"`
	Start    *int64 `json:"start,omitempty"`
	Tid      string `json:"tid,omitempty"`
}
