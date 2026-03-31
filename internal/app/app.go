package app

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ctreminiom/go-atlassian/pkg/infra/models"
	"jira-epic-dag/internal/jira"
)

// Project holds a Jira project key and name.
type Project struct {
	Key  string
	Name string
}

// Epic holds the fields we display for each open epic.
type Epic struct {
	Key      string
	Summary  string
	Status   string
	Assignee string
	Project  string
}

// FetchProjectsMsg is sent when the project list is fetched.
type FetchProjectsMsg struct {
	Projects []Project
	Err      error
}

// Task holds the fields we display for an epic's child issue.
type Task struct {
	Key       string
	Summary   string
	Status    string
	Assignee  string
	IssueType string
	BlockedBy []string
}

// FetchEpicsMsg is sent when the epic search completes.
type FetchEpicsMsg struct {
	Epics []Epic
	Err   error
}

// FetchTasksMsg is sent when the child task search completes.
type FetchTasksMsg struct {
	Tasks []Task
	Err   error
}

// GenerateDAGMsg is sent when the DAG has been generated.
type GenerateDAGMsg struct {
	Server *DAGServer
	Err    error
}

// FetchProjects returns a Bubble Tea Cmd that fetches the user's projects.
func FetchProjects(client *jira.Client) tea.Cmd {
	return func() tea.Msg {
		keys, err := client.MyProjectKeys(context.Background())
		if err != nil {
			return FetchProjectsMsg{Err: err}
		}

		projects := make([]Project, len(keys))
		for i, k := range keys {
			projects[i] = Project{Key: k}
		}
		return FetchProjectsMsg{Projects: projects}
	}
}

// FetchEpics returns a Bubble Tea Cmd that searches for open epics in the given project.
func FetchEpics(client *jira.Client, projectKey string) tea.Cmd {
	return func() tea.Msg {
		issues, err := client.SearchOpenEpics(context.Background(), []string{projectKey})
		if err != nil {
			return FetchEpicsMsg{Err: err}
		}

		epics := make([]Epic, 0, len(issues))
		for _, issue := range issues {
			e := Epic{
				Key: issue.Key,
			}
			if f := issue.Fields; f != nil {
				e.Summary = f.Summary
				if f.Status != nil {
					e.Status = f.Status.Name
				}
				if f.Assignee != nil {
					e.Assignee = f.Assignee.DisplayName
				}
				if f.Project != nil {
					e.Project = f.Project.Key
				}
			}
			epics = append(epics, e)
		}
		return FetchEpicsMsg{Epics: epics}
	}
}

// FetchTasks returns a Bubble Tea Cmd that fetches child issues of an epic.
func FetchTasks(client *jira.Client, epicKey string) tea.Cmd {
	return func() tea.Msg {
		issues, err := client.SearchEpicChildren(context.Background(), epicKey)
		if err != nil {
			return FetchTasksMsg{Err: err}
		}

		tasks := issuesToTasks(issues)
		return FetchTasksMsg{Tasks: tasks}
	}
}

// GenerateDAG fetches tasks for an epic and renders a Mermaid DAG as ASCII.
func GenerateDAG(client *jira.Client, epic Epic) tea.Cmd {
	return func() tea.Msg {
		issues, err := client.SearchEpicChildren(context.Background(), epic.Key)
		if err != nil {
			return GenerateDAGMsg{Err: err}
		}

		allTasks := issuesToTasks(issues)
		// Filter out the epic itself from the task list
		tasks := make([]Task, 0, len(allTasks))
		for _, t := range allTasks {
			if t.Key != epic.Key && t.Status != "Cancelled" {
				tasks = append(tasks, t)
			}
		}

		// Build graph edges
		taskSet := make(map[string]bool)
		for _, t := range tasks {
			taskSet[t.Key] = true
		}
		edges := make(map[string]map[string]bool)
		for _, t := range tasks {
			for _, blocker := range t.BlockedBy {
				if taskSet[blocker] {
					if edges[blocker] == nil {
						edges[blocker] = make(map[string]bool)
					}
					edges[blocker][t.Key] = true
				}
			}
		}
		reduced := transitiveReduction(edges)
		mermaidStr := buildMermaid(epic, tasks, reduced)

		title := fmt.Sprintf("%s: %s", epic.Key, epic.Summary)
		server, err := StartDAGServer(title, mermaidStr)
		if err != nil {
			return GenerateDAGMsg{Err: err}
		}

		// Open in browser
		exec.Command("open", server.URL).Start()

		return GenerateDAGMsg{Server: server}
	}
}

func issuesToTasks(issues []*models.IssueSchemeV2) []Task {
	tasks := make([]Task, 0, len(issues))
	for _, issue := range issues {
		t := Task{
			Key: issue.Key,
		}
		if f := issue.Fields; f != nil {
			t.Summary = f.Summary
			if f.Status != nil {
				t.Status = f.Status.Name
			}
			if f.Assignee != nil {
				t.Assignee = f.Assignee.DisplayName
			}
			if f.IssueType != nil {
				t.IssueType = f.IssueType.Name
			}
			for _, link := range f.IssueLinks {
				if link.Type != nil && link.Type.Inward == "is blocked by" && link.InwardIssue != nil {
					t.BlockedBy = append(t.BlockedBy, link.InwardIssue.Key)
				}
			}
		}
		tasks = append(tasks, t)
	}
	return tasks
}

func buildMermaid(epic Epic, tasks []Task, reduced map[string]map[string]bool) string {
	// Find tasks with no in-epic blockers after reduction
	hasBlocker := make(map[string]bool)
	for _, targets := range reduced {
		for v := range targets {
			hasBlocker[v] = true
		}
	}

	// Generate mermaid
	var b strings.Builder
	b.WriteString("graph TD\n")

	epicID := sanitizeMermaidID(epic.Key)
	b.WriteString(fmt.Sprintf("    %s[\"%s: %s\"]\n", epicID, epic.Key, escapeMermaid(epic.Summary)))
	for _, t := range tasks {
		tid := sanitizeMermaidID(t.Key)
		b.WriteString(fmt.Sprintf("    %s[\"%s: %s\"]\n", tid, t.Key, escapeMermaid(t.Summary)))
	}

	b.WriteString("\n")

	// Epic -> unblocked tasks
	for _, t := range tasks {
		if !hasBlocker[t.Key] && t.Key != epic.Key {
			b.WriteString(fmt.Sprintf("    %s --> %s\n", epicID, sanitizeMermaidID(t.Key)))
		}
	}

	// Blocker -> blocked edges
	for _, t := range tasks {
		if targets, ok := reduced[t.Key]; ok {
			for v := range targets {
				b.WriteString(fmt.Sprintf("    %s --> %s\n", sanitizeMermaidID(t.Key), sanitizeMermaidID(v)))
			}
		}
	}

	// Style classes by status
	b.WriteString("\n")
	b.WriteString("    classDef done fill:#a6e3a1,stroke:#40a02b,color:#1e1e2e\n")
	b.WriteString("    classDef todo fill:#89b4fa,stroke:#1e66f5,color:#1e1e2e\n")
	b.WriteString("    classDef inprogress fill:#fab387,stroke:#fe640b,color:#1e1e2e\n")

	for _, t := range tasks {
		tid := sanitizeMermaidID(t.Key)
		switch t.Status {
		case "Done":
			b.WriteString(fmt.Sprintf("    class %s done\n", tid))
		case "To Do":
			b.WriteString(fmt.Sprintf("    class %s todo\n", tid))
		default:
			b.WriteString(fmt.Sprintf("    class %s inprogress\n", tid))
		}
	}

	return b.String()
}

func transitiveReduction(edges map[string]map[string]bool) map[string]map[string]bool {
	reduced := make(map[string]map[string]bool)
	for u, targets := range edges {
		reduced[u] = make(map[string]bool)
		for v := range targets {
			reduced[u][v] = true
		}
	}

	for u, targets := range edges {
		for v := range targets {
			if reachableWithout(edges, u, v) {
				delete(reduced[u], v)
			}
		}
	}

	return reduced
}

// reachableWithout checks if target is reachable from start without using the direct edge.
func reachableWithout(edges map[string]map[string]bool, start, target string) bool {
	visited := map[string]bool{start: true}
	queue := make([]string, 0)
	for next := range edges[start] {
		if next != target {
			queue = append(queue, next)
			visited[next] = true
		}
	}
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		if curr == target {
			return true
		}
		for next := range edges[curr] {
			if !visited[next] {
				visited[next] = true
				queue = append(queue, next)
			}
		}
	}
	return false
}

func sanitizeMermaidID(key string) string {
	return strings.ReplaceAll(key, "-", "_")
}

func escapeMermaid(s string) string {
	s = strings.ReplaceAll(s, "\"", "#quot;")
	return s
}
