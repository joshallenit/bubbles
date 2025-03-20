package table

import (
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	lipglosstable "github.com/charmbracelet/lipgloss/table"
)

// Model defines a state for the table widget.
type Model struct {
	KeyMap KeyMap
	Help   help.Model

	cols      []Column
	rows      []Row
	cursor    int
	focus     bool
	styleFunc StyleFunc

	// -1 to be fit height (all rows)
	manualHeight int
	// -1 to be fit width (all data)
	manualWidth int
	start       int
}

// Row represents one line in the table.
type Row []string

// Column defines the table structure.
type Column struct {
	Title string
	Width int
}

// KeyMap defines keybindings. It satisfies to the help.KeyMap interface, which
// is used to render the help menu.
type KeyMap struct {
	LineUp       key.Binding
	LineDown     key.Binding
	PageUp       key.Binding
	PageDown     key.Binding
	HalfPageUp   key.Binding
	HalfPageDown key.Binding
	GotoTop      key.Binding
	GotoBottom   key.Binding
}

// ShortHelp implements the KeyMap interface.
func (km KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{km.LineUp, km.LineDown}
}

// FullHelp implements the KeyMap interface.
func (km KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{km.LineUp, km.LineDown, km.GotoTop, km.GotoBottom},
		{km.PageUp, km.PageDown, km.HalfPageUp, km.HalfPageDown},
	}
}

// DefaultKeyMap returns a default set of keybindings.
func DefaultKeyMap() KeyMap {
	const spacebar = " "
	return KeyMap{
		LineUp: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		LineDown: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("b", "pgup"),
			key.WithHelp("b/pgup", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("f", "pgdown", spacebar),
			key.WithHelp("f/pgdn", "page down"),
		),
		HalfPageUp: key.NewBinding(
			key.WithKeys("u", "ctrl+u"),
			key.WithHelp("u", "½ page up"),
		),
		HalfPageDown: key.NewBinding(
			key.WithKeys("d", "ctrl+d"),
			key.WithHelp("d", "½ page down"),
		),
		GotoTop: key.NewBinding(
			key.WithKeys("home", "g"),
			key.WithHelp("g/home", "go to start"),
		),
		GotoBottom: key.NewBinding(
			key.WithKeys("end", "G"),
			key.WithHelp("G/end", "go to end"),
		),
	}
}

// Styles contains style definitions for this list component. By default, these
// values are generated by DefaultStyles.
type Styles struct {
	Header   lipgloss.Style
	Cell     lipgloss.Style
	Selected lipgloss.Style
}

type StyleFunc func(m Model, row int, col int) lipgloss.Style

// DefaultStyles returns a set of default style definitions for this table.
func DefaultStyles() Styles {
	return Styles{
		Selected: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212")),
		Header:   lipgloss.NewStyle().Bold(true).Padding(0, 1),
		Cell:     lipgloss.NewStyle().Padding(0, 1),
	}
}

// SetStyles sets the table styles.
func (m *Model) SetStyles(s Styles) {
	m.styleFunc = stylesToStyleFunc(s)
}

func stylesToStyleFunc(s Styles) StyleFunc {
	return func(m Model, row int, col int) lipgloss.Style {
		if row == lipglosstable.HeaderRow {
			return s.Header
		} else if row == m.Cursor() {
			return s.Selected
		} else {
			return s.Cell
		}
	}
}

// Option is used to set options in New. For example:
//
//	table := New(WithColumns([]Column{{Title: "ID", Width: 10}}))
type Option func(*Model)

// New creates a new model for the table widget.
func New(opts ...Option) Model {
	m := Model{
		cursor:       0,
		manualHeight: -1,
		manualWidth:  -1,
		KeyMap:       DefaultKeyMap(),
		Help:         help.New(),
	}
	m.styleFunc = stylesToStyleFunc(DefaultStyles())

	for _, opt := range opts {
		opt(&m)
	}
	return m
}

// WithColumns sets the table columns (headers).
func WithColumns(cols []Column) Option {
	return func(m *Model) {
		m.cols = cols
	}
}

// WithRows sets the table rows (data).
func WithRows(rows []Row) Option {
	return func(m *Model) {
		m.rows = rows
	}
}

// WithHeight sets the height of the table.
func WithHeight(h int) Option {
	return func(m *Model) {
		m.manualHeight = h
	}
}

// WithWidth sets the width of the table.
func WithWidth(w int) Option {
	return func(m *Model) {
		m.manualWidth = w
	}
}

// WithFocused sets the focus state of the table.
func WithFocused(f bool) Option {
	return func(m *Model) {
		m.focus = f
	}
}

// WithStyles sets the table styles.
func WithStyles(s Styles) Option {
	return func(m *Model) {
		m.styleFunc = stylesToStyleFunc(s)
	}
}

func WithStyleFunc(styleFunc StyleFunc) Option {
	return func(m *Model) {
		m.styleFunc = styleFunc
	}
}

// SetRows sets a new rows state.
func (m *Model) SetStyleFunc(styleFunc StyleFunc) {
	WithStyleFunc(styleFunc)(m)
}

// WithKeyMap sets the key map.
func WithKeyMap(km KeyMap) Option {
	return func(m *Model) {
		m.KeyMap = km
	}
}

// Update is the Bubble Tea update loop.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.focus {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.KeyMap.LineUp):
			m.MoveUp(1)
		case key.Matches(msg, m.KeyMap.LineDown):
			m.MoveDown(1)
		case key.Matches(msg, m.KeyMap.PageUp):
			m.MoveUp(m.Height())
		case key.Matches(msg, m.KeyMap.PageDown):
			m.MoveDown(m.Height())
		case key.Matches(msg, m.KeyMap.HalfPageUp):
			m.MoveUp(m.Height() / 2) //nolint:mnd
		case key.Matches(msg, m.KeyMap.HalfPageDown):
			m.MoveDown(m.Height() / 2) //nolint:mnd
		case key.Matches(msg, m.KeyMap.GotoTop):
			m.GotoTop()
		case key.Matches(msg, m.KeyMap.GotoBottom):
			m.GotoBottom()
		}
	}

	return m, nil
}

// Focused returns the focus state of the table.
func (m Model) Focused() bool {
	return m.focus
}

// Focus focuses the table, allowing the user to move around the rows and
// interact.
func (m *Model) Focus() {
	m.focus = true
}

// Blur blurs the table, preventing selection or movement.
func (m *Model) Blur() {
	m.focus = false
}

// View renders the component.
func (m Model) View() string {
	renderTable := lipglosstable.New()

	renderTable.StyleFunc(func(row, col int) lipgloss.Style {
		style := m.styleFunc(m, row, col)
		if row == lipglosstable.HeaderRow && m.cols[col].Width != 0 && style.GetWidth() == 0 {
			return style.Width(m.cols[col].Width)
		} else {
			return style
		}
	})
	renderTable.Data(tableData{m})
	if m.manualHeight != -1 {
		renderTable.Height(m.manualHeight)
	}
	if m.manualWidth != -1 {
		renderTable.Width(m.manualWidth)
	}
	return renderTable.Render()
}

// HelpView is a helper method for rendering the help menu from the keymap.
// Note that this view is not rendered by default and you must call it
// manually in your application, where applicable.
func (m Model) HelpView() string {
	return m.Help.View(m.KeyMap)
}

// SelectedRow returns the selected row.
// You can cast it to your own implementation.
func (m Model) SelectedRow() Row {
	if m.cursor < 0 || m.cursor >= len(m.rows) {
		return nil
	}

	return m.rows[m.cursor]
}

// Rows returns the current rows.
func (m Model) Rows() []Row {
	return m.rows
}

// Columns returns the current columns.
func (m Model) Columns() []Column {
	return m.cols
}

// SetRows sets a new rows state.
func (m *Model) SetRows(r []Row) {
	m.rows = r
}

// SetColumns sets a new columns state.
func (m *Model) SetColumns(c []Column) {
	m.cols = c
}

// SetWidth sets the width of the viewport of the table.
func (m *Model) SetWidth(w int) {
	WithWidth(w)(m)
}

// SetHeight sets the height of the viewport of the table.
func (m *Model) SetHeight(h int) {
	WithHeight(h)(m)
}

// Height returns the viewport height of the table.
func (m Model) Height() int {
	if m.manualHeight != -1 {
		return m.manualHeight
	} else {
		return len(m.rows)
	}
}

// Cursor returns the index of the selected row.
func (m Model) Cursor() int {
	return m.cursor
}

// SetCursor sets the cursor position in the table.
func (m *Model) SetCursor(n int) {
	m.cursor = clamp(n, 0, len(m.rows)-1)
	m.start = clamp(clamp(m.start, m.cursor-m.Height()-1, m.cursor+m.Height()-1), 0, len(m.rows)-1)
	println("Set cursor to ", m.cursor)
}

// MoveUp moves the selection up by any number of rows.
// It can not go above the first row.
func (m *Model) MoveUp(n int) {
	m.cursor = clamp(m.cursor-n, 0, len(m.rows)-1)
	m.start = clamp(clamp(m.start, m.cursor-m.Height()-1, m.cursor+m.Height()-1), 0, len(m.rows)-1)
}

// MoveDown moves the selection down by any number of rows.
// It can not go below the last row.
func (m *Model) MoveDown(n int) {
	m.cursor = clamp(m.cursor+n, 0, len(m.rows)-1)
	m.start = clamp(clamp(m.start, m.cursor-m.Height()-1, m.cursor+m.Height()-1), 0, len(m.rows)-1)
}

// GotoTop moves the selection to the first row.
func (m *Model) GotoTop() {
	m.MoveUp(m.cursor)
}

// GotoBottom moves the selection to the last row.
func (m *Model) GotoBottom() {
	m.MoveDown(len(m.rows))
}

// FromValues create the table rows from a simple string. It uses `\n` by
// default for getting all the rows and the given separator for the fields on
// each row.
func (m *Model) FromValues(value, separator string) {
	rows := []Row{}
	for _, line := range strings.Split(value, "\n") {
		r := Row{}
		for _, field := range strings.Split(line, separator) {
			r = append(r, field)
		}
		rows = append(rows, r)
	}

	m.SetRows(rows)
}

func max(a, b int) int {
	if a > b {
		return a
	}

	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}

	return b
}

func clamp(v, low, high int) int {
	return min(max(v, low), high)
}

type tableData struct {
	m Model
}

var _ lipglosstable.Data = tableData{}

func (t tableData) At(row, col int) string {
	if row == lipglosstable.HeaderRow {
		return t.m.cols[col].Title
	}
	return t.m.rows[t.m.start+row][col]
}

func (t tableData) Rows() int {
	return t.m.Height()
}

func (t tableData) Columns() int {
	return len(t.m.cols)
}
