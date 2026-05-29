package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/ripnet/shellodex/internal/model"
)

type settingsTab int

const (
	tabGeneral     settingsTab = iota
	tabCredentials settingsTab = iota
	tabGroups      settingsTab = iota
)

// settingsFormData is heap-allocated and referenced via pointer so huh's
// value bindings survive Bubble Tea's value copies of SettingsModel.
type settingsFormData struct {
	remote    string
	direction string
	clipboard string
}

type SettingsModel struct {
	activeTab settingsTab
	form      *huh.Form
	data      *settingsFormData
	credList  CredListModel
	groupList GroupListModel
	width     int
	height    int
}

func NewSettingsModel(cfg *model.Config) SettingsModel {
	m := SettingsModel{
		data: &settingsFormData{
			remote:    cfg.Sync.Remote,
			direction: cfg.Sync.Direction,
			clipboard: cfg.ClipboardMode,
		},
		credList:  NewCredListModel(cfg),
		groupList: NewGroupListModel(cfg),
	}
	if m.data.direction == "" {
		m.data.direction = "push"
	}
	if m.data.clipboard == "" {
		m.data.clipboard = "auto"
	}
	m.form = m.buildForm()
	return m
}

func (m SettingsModel) Init() tea.Cmd {
	return m.form.Init()
}

func (m SettingsModel) Update(msg tea.Msg) (SettingsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.credList.width = msg.Width
		m.credList.height = msg.Height
		m.groupList.width = msg.Width
		m.groupList.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		key := msg.String()
		switch key {
		case "[":
			if m.activeTab > tabGeneral {
				m.activeTab--
			}
			return m, nil
		case "]":
			if m.activeTab < tabGroups {
				m.activeTab++
			}
			return m, nil
		}
		switch m.activeTab {
		case tabCredentials:
			var cmd tea.Cmd
			m.credList, cmd = m.credList.Update(msg)
			return m, cmd
		case tabGroups:
			var cmd tea.Cmd
			m.groupList, cmd = m.groupList.Update(msg)
			return m, cmd
		}
	}
	// General tab (and all non-key messages) route to the huh form.
	if m.activeTab == tabGeneral {
		form, cmd := m.form.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.form = f
		}
		return m, cmd
	}
	return m, nil
}

func (m SettingsModel) View() string {
	if m.width == 0 {
		return ""
	}
	w, h := m.width, m.height
	tabBar := m.renderTabBar(w)
	div := styleDivider.Render(strings.Repeat("─", w))

	switch m.activeTab {
	case tabGeneral:
		box := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(colorMauve)).
			Padding(1, 2).
			Render(m.form.View())
		bodyH := h - 3 // tabBar + div + status
		if bodyH < 1 {
			bodyH = 1
		}
		body := lipgloss.Place(w, bodyH, lipgloss.Center, lipgloss.Center, box)
		status := styleStatusBar.Width(w).Render(
			fmt.Sprintf("%s%s switch tabs  %s close",
				styleStatusKey.Render("["),
				styleStatusKey.Render("]"),
				styleStatusKey.Render("esc"),
			),
		)
		return lipgloss.JoinVertical(lipgloss.Left, tabBar, div, body, status)

	case tabCredentials:
		listH := h - 3 // tabBar + div + status
		if listH < 1 {
			listH = 1
		}
		body := m.credList.renderRows(listH, w)
		status := m.renderListStatus(w)
		return lipgloss.JoinVertical(lipgloss.Left, tabBar, div, body, status)

	case tabGroups:
		listH := h - 3
		if listH < 1 {
			listH = 1
		}
		body := m.groupList.renderRows(listH, w)
		status := m.renderListStatus(w)
		return lipgloss.JoinVertical(lipgloss.Left, tabBar, div, body, status)
	}
	return ""
}

func (m SettingsModel) renderTabBar(w int) string {
	tabs := []struct {
		label string
		tab   settingsTab
	}{
		{"General", tabGeneral},
		{"Credentials", tabCredentials},
		{"Groups", tabGroups},
	}
	var parts []string
	for _, t := range tabs {
		if t.tab == m.activeTab {
			parts = append(parts, styleTabActive.Render(t.label))
		} else {
			parts = append(parts, styleTabInactive.Render(t.label))
		}
	}
	return styleHeader.Width(w).Render(strings.Join(parts, "  "))
}

func (m SettingsModel) renderListStatus(w int) string {
	return styleStatusBar.Width(w).Render(
		fmt.Sprintf("%s new  %s edit  %s delete  %s back  %s%s switch tabs",
			styleStatusKey.Render("n"),
			styleStatusKey.Render("e"),
			styleStatusKey.Render("d"),
			styleStatusKey.Render("esc"),
			styleStatusKey.Render("["),
			styleStatusKey.Render("]"),
		),
	)
}

func (m SettingsModel) IsDone() bool {
	return m.activeTab == tabGeneral && m.form.State == huh.StateCompleted
}

func (m SettingsModel) IsAborted() bool {
	return m.activeTab == tabGeneral && m.form.State == huh.StateAborted
}

func (m SettingsModel) IsOnListTab() bool {
	return m.activeTab != tabGeneral
}

func (m SettingsModel) IsOnCredTab() bool {
	return m.activeTab == tabCredentials
}

func (m SettingsModel) IsOnGroupsTab() bool {
	return m.activeTab == tabGroups
}

func (m SettingsModel) SwitchToGeneralTab() SettingsModel {
	m.activeTab = tabGeneral
	return m
}

func (m SettingsModel) SelectedCred() *model.Credential {
	return m.credList.SelectedCred()
}

func (m SettingsModel) SelectedGroup() *model.Group {
	return m.groupList.SelectedGroup()
}

func (m SettingsModel) Result() model.SyncConfig {
	return model.SyncConfig{
		Remote:    m.data.remote,
		Direction: m.data.direction,
	}
}

func (m SettingsModel) Clipboard() string {
	return m.data.clipboard
}

func (m *SettingsModel) buildForm() *huh.Form {
	d := m.data
	dirOpts := []huh.Option[string]{
		huh.NewOption("Push (local → remote)", "push"),
		huh.NewOption("Pull (remote → local)", "pull"),
		huh.NewOption("Sync (bisync)", "sync"),
	}
	clipOpts := []huh.Option[string]{
		huh.NewOption("Auto (native tool, else OSC52)", "auto"),
		huh.NewOption("Native tool (pbcopy/wl-copy/xclip)", "native"),
		huh.NewOption("OSC52 (terminal escape)", "osc52"),
		huh.NewOption("Off", "off"),
	}
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("rclone Remote").
				Description("e.g. gdrive:shellodex or s3:mybucket/shellodex").
				Value(&d.remote),
			huh.NewSelect[string]().
				Title("Default Sync Direction").
				Options(dirOpts...).
				Value(&d.direction),
			huh.NewSelect[string]().
				Title("Copy password on connect").
				Description("Put a host's password on the clipboard when connecting").
				Options(clipOpts...).
				Value(&d.clipboard),
		),
	).WithTheme(huh.ThemeCatppuccin()).
		WithWidth(60)
}
