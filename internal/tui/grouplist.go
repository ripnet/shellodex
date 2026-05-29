package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ripnet/shellodex/internal/model"
)

type GroupListModel struct {
	cfg    *model.Config
	cursor int
	width  int
	height int
}

func NewGroupListModel(cfg *model.Config) GroupListModel {
	return GroupListModel{cfg: cfg}
}

func (m *GroupListModel) SetConfig(cfg *model.Config) {
	m.cfg = cfg
	if m.cursor >= len(m.cfg.Groups) {
		m.cursor = max(0, len(m.cfg.Groups)-1)
	}
}

func (m GroupListModel) Update(msg tea.Msg) (GroupListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "ctrl+p":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "ctrl+n":
			if m.cursor < len(m.cfg.Groups)-1 {
				m.cursor++
			}
		}
	}
	return m, nil
}

func (m GroupListModel) SelectedGroup() *model.Group {
	if len(m.cfg.Groups) == 0 {
		return nil
	}
	g := m.cfg.Groups[m.cursor]
	return &g
}

func (m GroupListModel) View() string {
	if m.width == 0 {
		return ""
	}
	w := m.width

	title := styleHeader.Width(w).Render("  Groups")
	div := styleDivider.Render(strings.Repeat("─", w))

	listH := m.height - 3
	rows := m.renderRows(listH, w)

	status := styleStatusBar.Width(w).Render(
		fmt.Sprintf("%s new  %s edit  %s delete  %s back",
			styleStatusKey.Render("n"),
			styleStatusKey.Render("e"),
			styleStatusKey.Render("d"),
			styleStatusKey.Render("esc"),
		),
	)
	return lipgloss.JoinVertical(lipgloss.Left, title, div, rows, status)
}

func (m GroupListModel) renderRows(maxRows, w int) string {
	if len(m.cfg.Groups) == 0 {
		return styleMuted.Width(w).Padding(1, 2).Render("No groups. Press n to add one.")
	}

	start := 0
	if m.cursor >= maxRows {
		start = m.cursor - maxRows + 1
	}
	end := start + maxRows
	if end > len(m.cfg.Groups) {
		end = len(m.cfg.Groups)
	}

	var sb strings.Builder
	for i := start; i < end; i++ {
		g := m.cfg.Groups[i]
		selected := i == m.cursor

		cursor := "  "
		if selected {
			cursor = styleCursor.Render("▶ ")
		}

		// Show the full breadcrumb so nesting is visible at a glance.
		path := m.cfg.GroupPath(g.ID)
		if path == "" {
			path = g.Name
		}
		line := "    " + cursor + path + "  " + styleGroupPath.Render(fmt.Sprintf("(%d hosts)", m.hostCount(g.ID)))

		if selected {
			sb.WriteString(styleSelected.Width(w).Render(line))
		} else {
			sb.WriteString(styleNormal.Width(w).Render(line))
		}
		if i < end-1 {
			sb.WriteRune('\n')
		}
	}
	for i := end - start; i < maxRows; i++ {
		sb.WriteRune('\n')
	}
	return sb.String()
}

func (m GroupListModel) hostCount(groupID string) int {
	n := 0
	for _, h := range m.cfg.Hosts {
		if h.GroupID == groupID {
			n++
		}
	}
	return n
}
