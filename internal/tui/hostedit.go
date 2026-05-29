package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/ripnet/shellodex/internal/model"
)

// hostFormData holds the form's bound values. It is heap-allocated and
// referenced via pointer so that huh's value bindings survive the value
// copies that Bubble Tea makes of HostEditModel on every Update.
type hostFormData struct {
	name          string
	hostname      string
	portStr       string
	protocol      string
	credID        string
	username      string
	password      string
	keyPath       string
	jumpID        string
	groupID       string
	notes         string
	tagsStr       string     // comma- or space-separated tags
	lastConnected *time.Time // pass-through; not editable in the form
}

// HostEditModel wraps a huh form for creating or editing a Host.
type HostEditModel struct {
	form   *huh.Form
	cfg    *model.Config
	data   *hostFormData
	isNew  bool
	origID string
}

func NewHostEditModel(cfg *model.Config, existing *model.Host) HostEditModel {
	m := HostEditModel{cfg: cfg, data: &hostFormData{}}
	if existing != nil {
		m.origID = existing.ID
		m.data.name = existing.Name
		m.data.hostname = existing.Hostname
		m.data.portStr = fmt.Sprintf("%d", existing.Port)
		if m.data.portStr == "0" {
			m.data.portStr = ""
		}
		m.data.protocol = string(existing.Protocol)
		m.data.credID = existing.CredentialID
		m.data.username = existing.Username
		m.data.password = existing.Password
		m.data.keyPath = existing.KeyPath
		m.data.jumpID = existing.JumpHostID
		m.data.groupID = existing.GroupID
		m.data.notes = existing.Notes
		m.data.tagsStr = strings.Join(existing.Tags, ", ")
		m.data.lastConnected = existing.LastConnected
	} else {
		m.isNew = true
	}
	if m.data.protocol == "" {
		m.data.protocol = string(model.SSH)
	}
	m.form = m.buildForm()
	return m
}

func (m HostEditModel) Init() tea.Cmd {
	return m.form.Init()
}

func (m HostEditModel) Update(msg tea.Msg) (HostEditModel, tea.Cmd) {
	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
	}
	return m, cmd
}

func (m HostEditModel) View() string {
	return m.form.View()
}

func (m HostEditModel) IsDone() bool {
	return m.form.State == huh.StateCompleted
}

func (m HostEditModel) IsAborted() bool {
	return m.form.State == huh.StateAborted
}

// Result builds the Host from the form values. Call only when IsDone() is true.
func (m HostEditModel) Result() model.Host {
	d := m.data
	port, err := strconv.Atoi(d.portStr)
	if err != nil || port <= 0 {
		port = int(model.DefaultPort(model.Protocol(d.protocol)))
	}
	id := m.origID
	if id == "" {
		id = model.NewID()
	}
	// Inline credentials only apply when no shared credential is linked.
	username, password, keyPath := d.username, d.password, d.keyPath
	if d.credID != "" {
		username, password, keyPath = "", "", ""
	}
	// Parse comma/space-separated tags, filtering empty tokens.
	var tags []string
	for _, t := range strings.FieldsFunc(d.tagsStr, func(r rune) bool { return r == ',' || r == ' ' }) {
		if t != "" {
			tags = append(tags, t)
		}
	}
	return model.Host{
		ID:            id,
		Name:          d.name,
		GroupID:       d.groupID,
		Protocol:      model.Protocol(d.protocol),
		Hostname:      d.hostname,
		Port:          port,
		CredentialID:  d.credID,
		Username:      username,
		Password:      password,
		KeyPath:       keyPath,
		JumpHostID:    d.jumpID,
		Notes:         d.notes,
		Tags:          tags,
		LastConnected: d.lastConnected,
	}
}

func (m *HostEditModel) buildForm() *huh.Form {
	d := m.data

	protoOpts := []huh.Option[string]{
		huh.NewOption("SSH", string(model.SSH)),
		huh.NewOption("Telnet", string(model.Telnet)),
	}

	credOpts := []huh.Option[string]{huh.NewOption("None", "")}
	for _, c := range m.cfg.Credentials {
		credOpts = append(credOpts, huh.NewOption(c.Name, c.ID))
	}

	// Jump host options (only SSH hosts are valid jump hosts)
	jumpOpts := []huh.Option[string]{huh.NewOption("None", "")}
	for _, h := range m.cfg.Hosts {
		if h.ID != m.origID && h.Protocol == model.SSH {
			jumpOpts = append(jumpOpts, huh.NewOption(h.Name+" ("+h.Hostname+")", h.ID))
		}
	}

	groupOpts := []huh.Option[string]{huh.NewOption("(none)", "")}
	for _, g := range m.cfg.Groups {
		groupOpts = append(groupOpts, huh.NewOption(m.cfg.GroupPath(g.ID), g.ID))
	}
	// Sentinel option: pick this to create a group on the fly. The app catches
	// newGroupOption when the form completes and opens the group editor.
	groupOpts = append(groupOpts, huh.NewOption("➕ New group…", newGroupOption))

	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Name").
				Description("Display label for this host").
				Value(&d.name),
			huh.NewInput().
				Title("Hostname / IP").
				Value(&d.hostname),
			huh.NewSelect[string]().
				Title("Protocol").
				Options(protoOpts...).
				Value(&d.protocol),
			huh.NewInput().
				Title("Port").
				Description("Leave blank for protocol default").
				Value(&d.portStr),
		),
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Credential").
				Description("Pick a shared credential, or leave as None to enter one inline").
				Options(credOpts...).
				Value(&d.credID),
		),
		// Inline (per-host) credential — shown only when no shared credential is
		// selected, so the two sources never compete.
		huh.NewGroup(
			huh.NewInput().
				Title("Username").
				Value(&d.username),
			huh.NewInput().
				Title("Password").
				Description("Optional — leave blank to type at the prompt or use keys").
				EchoMode(huh.EchoModePassword).
				Value(&d.password),
			huh.NewInput().
				Title("Key path").
				Description("Optional — e.g. ~/.ssh/id_ed25519").
				Value(&d.keyPath),
		).WithHideFunc(func() bool { return d.credID != "" }),
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Jump Host").
				Description("Connect via this bastion first").
				Options(jumpOpts...).
				Value(&d.jumpID),
			huh.NewSelect[string]().
				Title("Group").
				Options(groupOpts...).
				Value(&d.groupID),
			huh.NewInput().
				Title("Tags").
				Description("Space- or comma-separated labels, e.g. prod lab core").
				Value(&d.tagsStr),
			huh.NewText().
				Title("Notes").
				Value(&d.notes),
		),
	).WithTheme(huh.ThemeCatppuccin()).
		WithWidth(60)
}
