package app

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
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
			}
			tasks = append(tasks, t)
		}
		return FetchTasksMsg{Tasks: tasks}
	}
}
