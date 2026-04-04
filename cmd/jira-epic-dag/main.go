package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"jira-epic-dag/internal/jira"
	"jira-epic-dag/internal/tui"
)

func main() {
	host := os.Getenv("JIRA_HOST")
	email := os.Getenv("JIRA_EMAIL")
	token := os.Getenv("JIRA_API_TOKEN")

	if host == "" || email == "" || token == "" {
		fmt.Fprintln(os.Stderr, "Set JIRA_HOST, JIRA_EMAIL, and JIRA_API_TOKEN environment variables.")
		fmt.Fprintln(os.Stderr, "  JIRA_HOST=https://yoursite.atlassian.net")
		fmt.Fprintln(os.Stderr, "  JIRA_EMAIL=you@example.com")
		fmt.Fprintln(os.Stderr, "  JIRA_API_TOKEN=your-api-token")
		os.Exit(1)
	}

	client, err := jira.NewClient(host, email, token)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create Jira client: %v\n", err)
		os.Exit(1)
	}

	model := tui.NewModel(client)
	p := tea.NewProgram(model)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}
}
