package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/ripnet/shellodex/internal/model"
)

// credFormData is heap-allocated and referenced via pointer so huh's value
// bindings survive Bubble Tea's value copies of CredEditModel. See the note
// on hostFormData for details.
type credFormData struct {
	name     string
	username string
	password string
	keyPath  string
}

type CredEditModel struct {
	form   *huh.Form
	data   *credFormData
	isNew  bool
	origID string
}

func NewCredEditModel(existing *model.Credential) CredEditModel {
	m := CredEditModel{data: &credFormData{}}
	if existing != nil {
		m.origID = existing.ID
		m.data.name = existing.Name
		m.data.username = existing.Username
		m.data.password = existing.Password
		m.data.keyPath = existing.KeyPath
	} else {
		m.isNew = true
	}
	m.form = m.buildForm()
	return m
}

func (m CredEditModel) Init() tea.Cmd {
	return m.form.Init()
}

func (m CredEditModel) Update(msg tea.Msg) (CredEditModel, tea.Cmd) {
	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
	}
	return m, cmd
}

func (m CredEditModel) View() string {
	return m.form.View()
}

func (m CredEditModel) IsDone() bool {
	return m.form.State == huh.StateCompleted
}

func (m CredEditModel) IsAborted() bool {
	return m.form.State == huh.StateAborted
}

func (m CredEditModel) Result() model.Credential {
	d := m.data
	id := m.origID
	if id == "" {
		id = model.NewID()
	}
	return model.Credential{
		ID:       id,
		Name:     d.name,
		Username: d.username,
		Password: d.password,
		KeyPath:  d.keyPath,
	}
}

func (m *CredEditModel) buildForm() *huh.Form {
	d := m.data
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Name").
				Description("e.g. lab-admin").
				Value(&d.name),
			huh.NewInput().
				Title("Username").
				Value(&d.username),
			huh.NewInput().
				Title("Password").
				Description("Stored in plaintext — see _warning in config file").
				EchoMode(huh.EchoModePassword).
				Value(&d.password),
			huh.NewInput().
				Title("SSH Key Path").
				Description("Optional: path to private key (e.g. ~/.ssh/id_ed25519)").
				Value(&d.keyPath),
		),
	).WithTheme(huh.ThemeCatppuccin()).
		WithWidth(60)
}
