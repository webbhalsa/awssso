package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

type Item struct {
	title, desc string
	Value       any
}

func (i Item) Title() string       { return i.title }
func (i Item) Description() string { return i.desc }
func (i Item) FilterValue() string { return i.title + " " + i.desc }

func NewItem(title, desc string, value any) Item {
	return Item{title: title, desc: desc, Value: value}
}

type pickModel struct {
	list   list.Model
	choice *Item
}

func (m pickModel) Init() tea.Cmd { return nil }

func (m pickModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if i, ok := m.list.SelectedItem().(Item); ok {
				m.choice = &i
				return m, tea.Quit
			}
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m pickModel) View() string {
	return docStyle.Render(m.list.View())
}

// Pick shows an interactive list and returns the selected item, or nil if cancelled.
func Pick(title string, items []Item) (*Item, error) {
	listItems := make([]list.Item, len(items))
	for i, it := range items {
		listItems[i] = it
	}
	l := list.New(listItems, list.NewDefaultDelegate(), 0, 0)
	l.Title = title
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)

	tty, err := openTTY()
	if err != nil {
		return nil, err
	}
	defer tty.Close()

	m := pickModel{list: l}
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithInput(tty), tea.WithOutput(tty))
	result, err := p.Run()
	if err != nil {
		return nil, err
	}
	final := result.(pickModel)
	return final.choice, nil
}

// Confirm asks a simple yes/no question in the terminal.
func Confirm(prompt string) bool {
	var answer string
	fmt.Printf("%s [y/N]: ", prompt)
	fmt.Scanln(&answer)
	return answer == "y" || answer == "Y" || answer == "yes"
}
