package main

import (
	"context"
	"fmt"
	"os"
	"slices"

	"github.com/charmbracelet/bubbles/list"
	bubbletable "github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	width  int
	height int

	lists  []list.Model
	table  bubbletable.Model
	err    error
	loaded bool

	events     chan RowEvent
	tableItems []string
}

func (m *Model) initList(width, height int) {
	tableList := list.New([]list.Item{}, list.NewDefaultDelegate(), width, height)
	tableList.Title = "Tables"
	tableList.SetShowStatusBar(false)

	styles := list.DefaultStyles()
	styles.Title = styles.Title.Background(lipgloss.Color(""))
	tableList.Styles = styles

	m.lists = []list.Model{tableList}
}

func (m *Model) initTable() {
	columns := []bubbletable.Column{
		{Title: "ID", Width: 10},
		{Title: "first_name", Width: 10},
		{Title: "last_name", Width: 10},
	}
	m.table = bubbletable.New(bubbletable.WithColumns(columns))

}

func New() *Model {
	return &Model{
		events: make(chan RowEvent, 100),
	}
}

func (m Model) Init() tea.Cmd {
	ctx := context.Background()
	go listen(ctx, m.events)
	return waitForEvent(m.events)
}

func waitForEvent(events <-chan RowEvent) tea.Cmd {
	return func() tea.Msg {
		return <-events
	}
}

type item string

func (i item) Title() string       { return string(i) }
func (i item) Description() string { return "" }
func (i item) FilterValue() string { return string(i) }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if !m.loaded {
			m.width = msg.Width
			m.height = msg.Height

			leftWidth := msg.Width / 3
			// rightWidth := msg.Width - leftWidth

			halfHeight := msg.Height / 2

			m.initList(msg.Width, msg.Height)
			// m.initTable()

			m.lists[0].SetSize(leftWidth, halfHeight)

			m.loaded = true
		}
	case RowEvent:
		exists := slices.Contains(m.tableItems, msg.table)

		if !exists {
			m.tableItems = append(m.tableItems, msg.table)
		}

		items := []list.Item{}
		for _, v := range m.tableItems {
			items = append(items, item(v))
		}

		m.lists[0].SetItems(items)
		return m, waitForEvent(m.events) // Keep listening

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m Model) View() string {
	if m.loaded {
		style := lipgloss.NewStyle().Border(lipgloss.NormalBorder())

		leftColumn := lipgloss.JoinVertical(
			lipgloss.Left,
			style.Render(m.lists[0].View()),
		)

		rightColumn := style.Render(m.table.View())

		return lipgloss.JoinHorizontal(
			lipgloss.Top,
			leftColumn,
			rightColumn,
		)
	}

	return "LOADING"
}

func main() {
	m := New()
	p := tea.NewProgram(m)

	if err := p.Start(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
