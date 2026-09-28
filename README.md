# clickup-tui

A small terminal UI for the ClickUp work you do every day: this week’s tasks, comments, time, and search. Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea). Inbox is not included — ClickUp has no public inbox API.

## Setup

You need a ClickUp personal token from [Settings → Apps](https://app.clickup.com/settings/apps).

```bash
go build -o clickup .
./clickup init
./clickup
```

`init` saves `~/.config/clickup-tui/config.yaml`. You can also set `CLICKUP_TOKEN` / `CLICKUP_WORKSPACE`, or pass `--token` and `--workspace`.

## Keys

| Where | Keys |
|---|---|
| Anywhere | `1` week · `2` time · `3` search · `tab` cycle · `r` refresh · `q` quit |
| Lists | `j`/`k` move · `enter` open · `t` log time · `/` search |
| Task | `c` comment · `t` log time · `esc` back |
| Timesheet | `e` edit duration · `a` add (`TASK-ID 1h30m`) · `d` delete |
| Comment | `ctrl+s` submit · `esc` cancel |

Durations accept `1h`, `1h30m`, `30m`, `1:30`, or a bare number of minutes.

## Week

Shows tasks assigned to you that are due this week (Monday–Sunday), grouped by day. Today is highlighted. Open a task to read the description and comments, then comment or log time without opening the browser.

Time is written as a normal ClickUp time entry ending now, so it shows up on today's timesheet.
