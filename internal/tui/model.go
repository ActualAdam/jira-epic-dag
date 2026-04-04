package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"jira-epic-dag/internal/app"
	"jira-epic-dag/internal/jira"
)

type phase int

const (
	phaseLoadingProjects phase = iota
	phaseSelectProject
	phaseLoadingEpics
	phaseSelectEpic
	phaseLoadingTasks
	phaseShowTasks
)

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	keyStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

// Model holds all TUI state.
type Model struct {
	client      *jira.Client
	phase       phase
	projects    []app.Project
	projCursor  int
	epics       []app.Epic
	epicCursor  int
	tasks       []app.Task
	err         error
}

// NewModel creates the initial model.
func NewModel(client *jira.Client) Model {
	return Model{
		client: client,
		phase:  phaseLoadingProjects,
	}
}

func (m Model) Init() tea.Cmd {
	return app.FetchProjects(m.client)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc":
			return m.goBack()
		case "up", "k":
			m.moveCursor(-1)
		case "down", "j":
			m.moveCursor(1)
		case "enter":
			return m.selectItem()
		}

	case app.FetchProjectsMsg:
		if msg.Err != nil {
			m.err = msg.Err
		} else {
			m.projects = msg.Projects
			m.phase = phaseSelectProject
		}

	case app.FetchEpicsMsg:
		if msg.Err != nil {
			m.err = msg.Err
		} else {
			m.epics = msg.Epics
			m.epicCursor = 0
			m.phase = phaseSelectEpic
		}

	case app.FetchTasksMsg:
		if msg.Err != nil {
			m.err = msg.Err
		} else {
			m.tasks = msg.Tasks
			m.phase = phaseShowTasks
		}
	}

	return m, nil
}

func (m *Model) moveCursor(delta int) {
	switch m.phase {
	case phaseSelectProject:
		m.projCursor = clamp(m.projCursor+delta, 0, len(m.projects)-1)
	case phaseSelectEpic:
		m.epicCursor = clamp(m.epicCursor+delta, 0, len(m.epics)-1)
	}
}

func (m Model) selectItem() (tea.Model, tea.Cmd) {
	switch m.phase {
	case phaseSelectProject:
		if len(m.projects) > 0 {
			m.phase = phaseLoadingEpics
			return m, app.FetchEpics(m.client, m.projects[m.projCursor].Key)
		}
	case phaseSelectEpic:
		if len(m.epics) > 0 {
			m.phase = phaseLoadingTasks
			return m, app.FetchTasks(m.client, m.epics[m.epicCursor].Key)
		}
	}
	return m, nil
}

func (m Model) goBack() (tea.Model, tea.Cmd) {
	switch m.phase {
	case phaseSelectEpic:
		m.phase = phaseSelectProject
		m.epics = nil
		m.epicCursor = 0
	case phaseShowTasks:
		m.phase = phaseSelectEpic
		m.tasks = nil
	}
	return m, nil
}

func (m Model) View() string {
	if m.err != nil {
		return errorStyle.Render(fmt.Sprintf("Error: %v", m.err)) + "\n\nPress q to quit.\n"
	}

	switch m.phase {
	case phaseLoadingProjects:
		return "Loading projects...\n"

	case phaseSelectProject:
		var b strings.Builder
		b.WriteString(titleStyle.Render("Select a project") + "\n\n")
		for i, p := range m.projects {
			cursor := "  "
			style := dimStyle
			if i == m.projCursor {
				cursor = "> "
				style = selectedStyle
			}
			b.WriteString(style.Render(fmt.Sprintf("%s%s", cursor, p.Key)) + "\n")
		}
		b.WriteString("\n" + dimStyle.Render("↑/↓ move, enter select, q quit") + "\n")
		return b.String()

	case phaseLoadingEpics:
		return fmt.Sprintf("Loading epics for %s...\n", m.projects[m.projCursor].Key)

	case phaseSelectEpic:
		var b strings.Builder
		project := m.projects[m.projCursor].Key
		b.WriteString(titleStyle.Render(fmt.Sprintf("Open Epics — %s", project)) + fmt.Sprintf(" (%d)\n\n", len(m.epics)))

		if len(m.epics) == 0 {
			b.WriteString("No open epics found.\n")
		} else {
			for i, e := range m.epics {
				cursor := "  "
				style := lipgloss.NewStyle()
				if i == m.epicCursor {
					cursor = "> "
					style = selectedStyle
				}
				assignee := "Unassigned"
				if e.Assignee != "" {
					assignee = e.Assignee
				}
				b.WriteString(style.Render(fmt.Sprintf("%s%s  %s  [%s]  %s",
					cursor,
					keyStyle.Render(e.Key),
					e.Summary,
					e.Status,
					assignee,
				)) + "\n")
			}
		}
		b.WriteString("\n" + dimStyle.Render("↑/↓ move, enter select, esc back, q quit") + "\n")
		return b.String()

	case phaseLoadingTasks:
		return fmt.Sprintf("Loading tasks for %s...\n", m.epics[m.epicCursor].Key)

	case phaseShowTasks:
		var b strings.Builder
		epic := m.epics[m.epicCursor]
		b.WriteString(titleStyle.Render(fmt.Sprintf("%s — %s", epic.Key, epic.Summary)) + fmt.Sprintf(" (%d tasks)\n\n", len(m.tasks)))

		if len(m.tasks) == 0 {
			b.WriteString("No child tasks found.\n")
		} else {
			for _, t := range m.tasks {
				assignee := "Unassigned"
				if t.Assignee != "" {
					assignee = t.Assignee
				}
				b.WriteString(fmt.Sprintf("%s  %s  [%s]  %s  %s\n",
					keyStyle.Render(t.Key),
					t.Summary,
					t.Status,
					t.IssueType,
					assignee,
				))
			}
		}
		b.WriteString("\n" + dimStyle.Render("esc back, q quit") + "\n")
		return b.String()
	}

	return ""
}

func clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
