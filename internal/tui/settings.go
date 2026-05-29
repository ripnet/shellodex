package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/ripnet/shellodex/internal/model"
)

// settingsFormData is heap-allocated and referenced via pointer so huh's
// value bindings survive Bubble Tea's value copies of SettingsModel.
type settingsFormData struct {
	remote    string
	direction string
	clipboard string
}

type SettingsModel struct {
	form *huh.Form
	data *settingsFormData
}

func NewSettingsModel(cfg *model.Config) SettingsModel {
	m := SettingsModel{data: &settingsFormData{
		remote:    cfg.Sync.Remote,
		direction: cfg.Sync.Direction,
		clipboard: cfg.ClipboardMode,
	}}
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
	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
	}
	return m, cmd
}

func (m SettingsModel) View() string {
	return m.form.View()
}

func (m SettingsModel) IsDone() bool {
	return m.form.State == huh.StateCompleted
}

func (m SettingsModel) IsAborted() bool {
	return m.form.State == huh.StateAborted
}

func (m SettingsModel) Result() model.SyncConfig {
	return model.SyncConfig{
		Remote:    m.data.remote,
		Direction: m.data.direction,
	}
}

// Clipboard returns the selected clipboard-on-connect mode.
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
