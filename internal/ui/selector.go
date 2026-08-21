package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// AccountItem represents a selectable account
type AccountItem struct {
	Name     string
	IsActive bool
}

// SelectorModel is the bubbletea model for account selection
type SelectorModel struct {
	Items    []AccountItem
	Cursor   int
	Selected int
	Agent    string
	Done     bool
	Quitting bool
}

// NewSelector creates a new account selector
func NewSelector(agent string, items []AccountItem) SelectorModel {
	selected := -1
	for i, item := range items {
		if item.IsActive {
			selected = i
		}
	}
	return SelectorModel{
		Items:    items,
		Cursor:   0,
		Selected: selected,
		Agent:    agent,
	}
}

func (m SelectorModel) Init() tea.Cmd {
	return nil
}

func (m SelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.Quitting = true
			return m, tea.Quit
		case "up", "k":
			if m.Cursor > 0 {
				m.Cursor--
			}
		case "down", "j":
			if m.Cursor < len(m.Items)-1 {
				m.Cursor++
			}
		case "enter":
			m.Selected = m.Cursor
			m.Done = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m SelectorModel) View() string {
	if m.Done || m.Quitting {
		return ""
	}

	header := SectionHeader(IconSwitch, fmt.Sprintf("Switch Account · %s", AgentName.Render(m.Agent)))

	s := "\n" + header + "\n\n"
	s += "  Select account:\n\n"

	for i, item := range m.Items {
		cursor := "  "
		style := AccountInactive
		indicator := IconDotOpen

		if m.Cursor == i {
			cursor = Accent.Render("> ")
			style = Bright
			indicator = Accent.Render(IconDot)
		}

		if item.IsActive {
			indicator = Success.Render(IconDot)
		}

		line := fmt.Sprintf("%s%s %s", cursor, indicator, style.Render(item.Name))

		if item.IsActive {
			line += Muted.Render("  ← active")
		}

		s += "  " + line + "\n"
	}

	s += "\n" + Muted.Render("  ↑/↓ navigate · enter select · q quit") + "\n"

	return BoxBorder.Render(s)
}

// GetSelectedAccount returns the selected account name
func (m SelectorModel) GetSelectedAccount() string {
	if m.Selected >= 0 && m.Selected < len(m.Items) {
		return m.Items[m.Selected].Name
	}
	return ""
}
