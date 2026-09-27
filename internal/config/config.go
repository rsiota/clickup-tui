package config

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"clickup-tui/internal/clickup"

	"golang.org/x/term"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Token     string `yaml:"token"`
	Workspace string `yaml:"workspace,omitempty"`
	Path      string `yaml:"-"`
}

func configPath() (string, error) {
	if p := os.Getenv("CLICKUP_TUI_CONFIG"); p != "" {
		return p, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "clickup-tui", "config.yaml"), nil
}

func Load(tokenFlag, workspaceFlag string) (*Config, error) {
	cfg := &Config{
		Token:     firstNonEmpty(tokenFlag, os.Getenv("CLICKUP_TOKEN")),
		Workspace: firstNonEmpty(workspaceFlag, os.Getenv("CLICKUP_WORKSPACE")),
	}

	path, err := configPath()
	if err == nil {
		cfg.Path = path
		if data, readErr := os.ReadFile(path); readErr == nil {
			var file Config
			if err := yaml.Unmarshal(data, &file); err != nil {
				return nil, fmt.Errorf("parse %s: %w", path, err)
			}
			if cfg.Token == "" {
				cfg.Token = file.Token
			}
			if cfg.Workspace == "" {
				cfg.Workspace = file.Workspace
			}
		}
	}

	if cfg.Token == "" {
		return nil, fmt.Errorf(`no ClickUp token found.

Get a personal token from https://app.clickup.com/settings/apps
then run:

  clickup init

or set CLICKUP_TOKEN, or pass --token`)
	}
	return cfg, nil
}

func (c *Config) Save() error {
	if c.Path == "" {
		path, err := configPath()
		if err != nil {
			return err
		}
		c.Path = path
	}
	if err := os.MkdirAll(filepath.Dir(c.Path), 0o700); err != nil {
		return err
	}
	data, err := yaml.Marshal(&Config{Token: c.Token, Workspace: c.Workspace})
	if err != nil {
		return err
	}
	return os.WriteFile(c.Path, data, 0o600)
}

func Init() error {
	fmt.Print("ClickUp API token (pk_...): ")
	raw, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return err
	}
	token := strings.TrimSpace(string(raw))
	if token == "" {
		return fmt.Errorf("token is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	client := clickup.New(token)
	user, err := client.GetUser(ctx)
	if err != nil {
		return fmt.Errorf("token check failed: %w", err)
	}
	fmt.Printf("Signed in as %s\n", displayUser(*user))

	teams, err := client.ListWorkspaces(ctx)
	if err != nil {
		return fmt.Errorf("list workspaces: %w", err)
	}
	if len(teams) == 0 {
		return fmt.Errorf("this token has no workspaces")
	}

	workspace := teams[0].ID.String()
	if len(teams) == 1 {
		fmt.Printf("Workspace: %s\n", teams[0].Name)
	} else {
		fmt.Println("Workspaces:")
		for i, t := range teams {
			fmt.Printf("  %d) %s  (%s)\n", i+1, t.Name, t.ID)
		}
		fmt.Print("Choose workspace [1]: ")
		in := bufio.NewReader(os.Stdin)
		line, _ := in.ReadString('\n')
		line = strings.TrimSpace(line)
		if line != "" {
			n, err := strconv.Atoi(line)
			if err != nil || n < 1 || n > len(teams) {
				return fmt.Errorf("invalid choice")
			}
			workspace = teams[n-1].ID.String()
			fmt.Printf("Workspace: %s\n", teams[n-1].Name)
		} else {
			fmt.Printf("Workspace: %s\n", teams[0].Name)
		}
	}

	cfg := &Config{Token: token, Workspace: workspace}
	if err := cfg.Save(); err != nil {
		return err
	}
	fmt.Printf("Saved %s\n", cfg.Path)
	return nil
}

func displayUser(u clickup.User) string {
	if u.Username != "" {
		return u.Username
	}
	return u.Email
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
