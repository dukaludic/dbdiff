package main

import (
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
}

func (m *Model) initLists(width, height int) {
	snapshotlist := list.New([]list.Item{}, list.NewDefaultDelegate(), width, height/2)
	snapshotlist.Title = "Snapshots"
	tableList := list.New([]list.Item{}, list.NewDefaultDelegate(), width, height/2)
	tableList.Title = "Tables"

	m.lists = []list.Model{snapshotlist, tableList}
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
	return &Model{}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if !m.loaded {
			m.width = msg.Width
			m.height = msg.Height

			leftWidth := msg.Width / 3
			// rightWidth := msg.Width - leftWidth

			halfHeight := msg.Height / 2

			m.initLists(msg.Width, msg.Height)
			m.initTable()

			m.lists[0].SetSize(leftWidth, halfHeight-2)
			m.lists[1].SetSize(leftWidth, msg.Height-halfHeight-2)

			m.loaded = true
		}

		// m.initLists(msg.Width, msg.Height)

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
			style.Render(m.lists[1].View()),
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

	listen()

	// m := New()
	// p := tea.NewProgram(m)

	// if err := p.Start(); err != nil {
	// 	fmt.Println(err)
	// 	os.Exit(1)
	// }
}
