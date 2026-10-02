# clickup-tui

A terminal UI for everyday ClickUp work. Browse this week’s assigned tasks, open a task to read the description and comments, leave comments (with `@` mentions), change status, log time, and manage today’s timesheet — without leaving the terminal. Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

Inbox is not included — ClickUp has no public inbox API.

## Setup

You need a ClickUp personal token from [Settings → Apps](https://app.clickup.com/settings/apps).

```bash
go build -o clickup .
./clickup init
./clickup
```

`init` saves `~/.config/clickup-tui/config.yaml`. You can also set `CLICKUP_TOKEN` / `CLICKUP_WORKSPACE`, or pass `--token` and `--workspace`.

## What it does

**Week** — tasks assigned to you that are due this week (Monday–Sunday), grouped by day. Jump weeks with `[` / `]`, return to the current week with `T`, sort columns with `f`.

**Task detail** — open a row to read the description and comments. From there you can comment (`c`, `@` to mention), change status (`s`), log time (`t`), or open the task in the browser (`o`).

**Time** — a timesheet for the day. Edit durations, add or delete entries, yank/paste rows (`yy` / `p`), and step days with `[` / `]`.

## Keys

| Where | Keys |
|---|---|
| Anywhere | `1` week · `2` time · `tab` cycle · `r` refresh · `q` quit |
| Lists | `↑`/`↓`/`←`/`→` move · `enter` open · `f` sort · `s` status · `t` log time · `o` browser |
| Week | `[`/`]` week · `T` this week |
| Task | `c` comment · `t` log time · `s` status · `o` browser · `esc` close |
| Timesheet | `[`/`]` day · `t` today · `i`/`enter` edit · `a` add · `d` delete · `yy` yank · `p` paste |
| Comment | `ctrl+s` submit · `@` mention · `esc` cancel |

Durations accept `1h`, `1h30m`, `30m`, `1:30`, or a bare number of minutes.
Optional start time: `9:30 1h30m` (today at 09:30). Alone, `1:30` is still a duration ending now.
