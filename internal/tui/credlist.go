package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ripnet/shellodex/internal/model"
)

type CredListModel struct {
	cfg    *model.Config
	cursor int
	width  int
	height int
}

func NewCredListModel(cfg *model.Config) CredListModel {
	return CredListModel{cfg: cfg}
}

func (m *CredListModel) SetConfig(cfg *model.Config) {
	m.cfg = cfg
	if m.cursor >= len(m.cfg.Credentials) {
		m.cursor = max(0, len(m.cfg.Credentials)-1)
	}
}

func (m CredListModel) Update(msg tea.Msg) (CredListModel, tea.Cmd) {
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
			if m.cursor < len(m.cfg.Credentials)-1 {
				m.cursor++
			}
		}
	}
	return m, nil
}

func (m CredListModel) SelectedCred() *model.Credential {
	if len(m.cfg.Credentials) == 0 {
		return nil
	}
	c := m.cfg.Credentials[m.cursor]
	return &c
}

func (m CredListModel) View() string {
	if m.width == 0 {
		return ""
	}
	w := m.width

	title := styleHeader.Width(w).Render("  Credentials")
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

func (m CredListModel) renderRows(maxRows, w int) string {
	if len(m.cfg.Credentials) == 0 {
		return styleMuted.Width(w).Padding(1, 2).Render("No credentials. Press n to add one.")
	}

	start := 0
	if m.cursor >= maxRows {
		start = m.cursor - maxRows + 1
	}
	end := start + maxRows
	if end > len(m.cfg.Credentials) {
		end = len(m.cfg.Credentials)
	}

	var sb strings.Builder
	for i := start; i < end; i++ {
		c := m.cfg.Credentials[i]
		selected := i == m.cursor

		cursor := "  "
		if selected {
			cursor = styleCursor.Render("▶ ")
		}

		label := c.Name
		sub := c.Username
		if c.KeyPath != "" {
			sub += " (key)"
		}
		line := "    " + cursor + label + "  " + styleGroupPath.Render(sub)

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
