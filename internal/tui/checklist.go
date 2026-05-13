package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	checkStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	uncheckStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	cursorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
	titleStyle    = lipgloss.NewStyle().Bold(true).MarginBottom(1)
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).MarginTop(1)
)

type CheckItem struct {
	Label   string
	Value   any
	Checked bool
}

type checkModel struct {
	title   string
	items   []CheckItem
	cursor  int
	filter  string
	indices []int // indices into items that match filter
}

func newCheckModel(title string, items []CheckItem) checkModel {
	idx := make([]int, len(items))
	for i := range items {
		idx[i] = i
	}
	return checkModel{title: title, items: items, indices: idx}
}

func (m checkModel) Init() tea.Cmd { return nil }

func (m checkModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.items = nil
			m.indices = nil
			return m, tea.Quit
		case "enter":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.indices)-1 {
				m.cursor++
			}
		case " ":
			if len(m.indices) > 0 {
				m.items[m.indices[m.cursor]].Checked = !m.items[m.indices[m.cursor]].Checked
			}
		case "a":
			allChecked := true
			for _, i := range m.indices {
				if !m.items[i].Checked {
					allChecked = false
					break
				}
			}
			for _, i := range m.indices {
				m.items[i].Checked = !allChecked
			}
		case "backspace":
			if len(m.filter) > 0 {
				m.filter = m.filter[:len(m.filter)-1]
				m.rebuildIndices()
			}
		default:
			if len(msg.String()) == 1 {
				m.filter += msg.String()
				m.rebuildIndices()
				if m.cursor >= len(m.indices) {
					m.cursor = max(0, len(m.indices)-1)
				}
			}
		}
	}
	return m, nil
}

func (m *checkModel) rebuildIndices() {
	m.indices = nil
	f := strings.ToLower(m.filter)
	for i, it := range m.items {
		if f == "" || strings.Contains(strings.ToLower(it.Label), f) {
			m.indices = append(m.indices, i)
		}
	}
	m.cursor = 0
}

func (m checkModel) View() string {
	var sb strings.Builder
	sb.WriteString(titleStyle.Render(m.title))
	sb.WriteString("\n")

	if m.filter != "" {
		sb.WriteString(fmt.Sprintf("filter: %s\n\n", m.filter))
	}

	for pos, i := range m.indices {
		item := m.items[i]
		checkbox := uncheckStyle.Render("[ ]")
		if item.Checked {
			checkbox = checkStyle.Render("[✓]")
		}
		label := item.Label
		if pos == m.cursor {
			label = cursorStyle.Render("> " + label)
		} else {
			label = "  " + label
		}
		sb.WriteString(fmt.Sprintf("%s %s\n", checkbox, label))
	}

	sb.WriteString(helpStyle.Render("\nspace: toggle  a: all/none  type to filter  enter: confirm  ctrl+c: cancel"))
	return docStyle.Render(sb.String())
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// MultiSelect shows an interactive checklist and returns the selected items.
func MultiSelect(title string, items []CheckItem) ([]CheckItem, error) {
	tty, err := openTTY()
	if err != nil {
		return nil, err
	}
	defer tty.Close()

	m := newCheckModel(title, items)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithInput(tty), tea.WithOutput(tty))
	result, err := p.Run()
	if err != nil {
		return nil, err
	}
	final := result.(checkModel)
	if final.items == nil {
		return nil, nil
	}
	var selected []CheckItem
	for _, it := range final.items {
		if it.Checked {
			selected = append(selected, it)
		}
	}
	return selected, nil
}
