package main

import (
	"context"
	"fmt"
	"os"
	"slices"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/table"
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

	events chan RowEvent
	tables []string

	selectedTable int
}

type UpdatedRowData struct {
	columns  []string
	previous table.Row
	current  table.Row
}

var tableColumnDiffMap = make(map[string]UpdatedRowData)

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
	m.table = bubbletable.New(bubbletable.WithColumns([]table.Column{}))
}

func (m *Model) loadSelectedTableRows(table string) {
	tableDiff := tableColumnDiffMap[table]

	rows := []bubbletable.Row{
		tableDiff.previous,
		tableDiff.current,
	}

	m.table.SetRows(rows)
}

func New() *Model {
	return &Model{
		events:        make(chan RowEvent, 100),
		selectedTable: -1,
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

func columnsFromNames(names []string) []bubbletable.Column {
	cols := make([]bubbletable.Column, len(names))
	for i, name := range names {
		cols[i] = bubbletable.Column{Title: name, Width: 15}
	}
	return cols
}

func (m *Model) loadSelectedTableColumns() {
	items := m.lists[0].Items()
	if len(items) == 0 || m.selectedTable < 0 || m.selectedTable >= len(items) {
		return
	}

	selectedTableName := items[m.selectedTable].FilterValue()
	m.table.SetRows([]table.Row{})
	m.table.SetColumns(columnsFromNames(tableColumnMap[selectedTableName]))
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if !m.loaded {
			m.width = msg.Width
			m.height = msg.Height

			leftWidth := msg.Width / 3

			m.initList(msg.Width, msg.Height)
			m.initTable()

			m.table.SetHeight(m.height - 2)
			m.table.SetWidth(m.width - 60)

			m.lists[0].SetSize(leftWidth, msg.Height-2)

			m.loaded = true
		}
	case RowEvent:

		cols := tableColumnMap[msg.table]

		prevMap := msg.data[0].(map[string]interface{})
		currMap := msg.data[1].(map[string]interface{})

		prevRow := make([]string, len(cols))
		currRow := make([]string, len(cols))

		for i, col := range cols {
			prevRow[i] = toString(prevMap[col])
			currRow[i] = toString(currMap[col])
		}

		previous := table.Row(prevRow)
		current := table.Row(currRow)

		tableColumnDiffMap[msg.table] = UpdatedRowData{
			columns:  cols,
			previous: previous,
			current:  current,
		}

		tableExists := slices.Contains(m.tables, msg.table)

		if !tableExists {
			m.tables = append(m.tables, msg.table)
		}

		items := []list.Item{}
		for _, v := range m.tables {
			items = append(items, item(v))
		}

		m.lists[0].SetItems(items)

		if len(m.lists[0].Items()) > 0 {
			if m.selectedTable == -1 {
				m.selectedTable = 0
				m.lists[0].Select(0)
				m.loadSelectedTableColumns()
				m.loadSelectedTableRows(m.tables[m.selectedTable])
			}
		}

		return m, waitForEvent(m.events) // Keep listening

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up":
			if m.selectedTable > 0 {
				m.selectedTable--
				m.lists[0].Select(m.selectedTable)
				m.loadSelectedTableColumns()
				m.loadSelectedTableRows(m.tables[m.selectedTable])
			}
		case "down":
			if m.selectedTable < len(m.lists[0].Items())-1 {
				m.selectedTable++
				m.lists[0].Select(m.selectedTable)
				m.loadSelectedTableColumns()
				m.loadSelectedTableRows(m.tables[m.selectedTable])
			}
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
	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
	)

	if err := p.Start(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
