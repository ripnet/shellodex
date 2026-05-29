package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ripnet/shellodex/internal/connect"
	"github.com/ripnet/shellodex/internal/model"
	shellsync "github.com/ripnet/shellodex/internal/sync"
	"github.com/ripnet/shellodex/internal/version"
)

type viewMode int

const (
	modeLauncher viewMode = iota
	modeTree
	modeHostEdit
	modeCredList
	modeCredEdit
	modeGroupList
	modeGroupEdit
	modeSettings
	modeConfirmDelete
	modeSyncOverlay
	modePasswordPopup
)

// newGroupOption is the sentinel value used by the host form's Group select to
// mean "create a new group inline". It is never a real group ID.
const newGroupOption = "\x00new-group"

// deleteKind identifies what a pending delete confirmation targets.
type deleteKind int

const (
	deleteHostKind deleteKind = iota
	deleteCredKind
	deleteGroupKind
)

// ConnectRequest is set when the user selects a host to connect to.
// main.go reads this after p.Run() returns and calls connect.Connect.
type ConnectRequest struct {
	Host model.Host
}

type syncDoneMsg struct {
	output string
	err    error
	silent bool // if true, suppress the overlay on success
}

type clearStatusMsg struct{}

type updateCheckMsg struct {
	version string // empty if no update available
}

func checkUpdateCmd() tea.Cmd {
	return func() tea.Msg {
		v := version.CheckForUpdate(version.Version)
		return updateCheckMsg{version: v}
	}
}

type AppModel struct {
	cfg      *model.Config
	cfgPath  string
	mode     viewMode
	prevMode viewMode

	launcher  LauncherModel
	tree      TreeModel
	hostEdit  HostEditModel
	credList  CredListModel
	credEdit  CredEditModel
	groupList GroupListModel
	groupEdit GroupEditModel
	settings  SettingsModel

	// Inline group creation from the host form: when set, a finished group form
	// should attach the new group to pendingHost and save it.
	groupForHost bool
	pendingHost  *model.Host

	deleteTarget     string
	deleteTargetName string
	deleteKind       deleteKind

	syncOutput string
	syncErr    bool

	popupHostName string
	popupPassword string

	width  int
	height int

	ConnectRequest *ConnectRequest

	saveFn func(*model.Config) error
	loadFn func() (*model.Config, error)
}

func NewAppModel(cfg *model.Config, cfgPath string, saveFn func(*model.Config) error, loadFn func() (*model.Config, error)) AppModel {
	return AppModel{
		cfg:       cfg,
		cfgPath:   cfgPath,
		saveFn:    saveFn,
		loadFn:    loadFn,
		mode:      modeLauncher,
		launcher:  NewLauncherModel(cfg),
		tree:      NewTreeModel(cfg),
		credList:  NewCredListModel(cfg),
		groupList: NewGroupListModel(cfg),
	}
}

func (m AppModel) Init() tea.Cmd {
	cmds := []tea.Cmd{checkUpdateCmd()}
	if m.cfg.Sync.SyncOnStartup && m.cfg.Sync.Remote != "" {
		cmds = append(cmds, m.doSync(true))
	}
	return tea.Batch(cmds...)
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case updateCheckMsg:
		if msg.version != "" {
			m.launcher.SetUpdateNotice(msg.version)
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.launcher.width = msg.Width
		m.launcher.height = msg.Height
		m.tree.width = msg.Width
		m.tree.height = msg.Height
		m.credList.width = msg.Width
		m.credList.height = msg.Height
		m.groupList.width = msg.Width
		m.groupList.height = msg.Height
		m.settings.width = msg.Width
		m.settings.height = msg.Height
		m.settings.credList.width = msg.Width
		m.settings.credList.height = msg.Height
		m.settings.groupList.width = msg.Width
		m.settings.groupList.height = msg.Height
		return m, nil

	case syncDoneMsg:
		if msg.err != nil {
			if msg.output != "" {
				m.syncOutput = msg.output
			} else {
				m.syncOutput = msg.err.Error()
			}
			m.syncErr = true
			m.mode = modeSyncOverlay
		} else {
			if m.loadFn != nil {
				if newCfg, err := m.loadFn(); err == nil {
					m.cfg = newCfg
					m.refreshSubmodels()
				}
			}
			if !msg.silent {
				m.syncOutput = msg.output
				if m.syncOutput == "" {
					m.syncOutput = "Sync complete."
				}
				m.syncErr = false
				m.mode = modeSyncOverlay
			}
		}
		return m, nil

	case clearStatusMsg:
		m.launcher.SetStatus("")
		m.tree.SetStatus("")
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m.delegateUpdate(msg)
}

func (m AppModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if key == "ctrl+c" {
		return m, tea.Quit
	}

	switch m.mode {
	case modeLauncher:
		return m.launcherKey(key, msg)
	case modeTree:
		return m.treeKey(key, msg)
	case modeHostEdit:
		return m.formKey(msg)
	case modeCredList:
		return m.credListKey(key, msg)
	case modeCredEdit:
		return m.credFormKey(msg)
	case modeGroupList:
		return m.groupListKey(key, msg)
	case modeGroupEdit:
		return m.groupFormKey(msg)
	case modeSettings:
		return m.settingsFormKey(msg)
	case modeConfirmDelete:
		return m.confirmDeleteKey(key)
	case modeSyncOverlay:
		m.mode = modeLauncher
		m.syncOutput = ""
		return m, nil
	case modePasswordPopup:
		return m.passwordPopupKey(key)
	}
	return m, nil
}

func (m AppModel) launcherKey(key string, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// In search mode, only Enter (connect) is an app-level action; everything
	// else — typing, Esc (exit search), arrows — belongs to the launcher.
	if m.launcher.IsSearching() {
		if key == "enter" {
			if host := m.launcher.SelectedHost(); host != nil {
				m.ConnectRequest = &ConnectRequest{Host: *host}
				return m, tea.Quit
			}
			return m, nil
		}
		var cmd tea.Cmd
		m.launcher, cmd = m.launcher.Update(msg)
		return m, cmd
	}

	// Browse mode: single-letter hotkeys.
	switch key {
	case "enter":
		if host := m.launcher.SelectedHost(); host != nil {
			m.ConnectRequest = &ConnectRequest{Host: *host}
			return m, tea.Quit
		}
	case "/":
		return m, m.launcher.StartSearch()
	case "tab":
		m.mode = modeTree
		return m, nil
	case "e":
		if host := m.launcher.SelectedHost(); host != nil {
			m.hostEdit = NewHostEditModel(m.cfg, host)
			m.prevMode = modeLauncher
			m.mode = modeHostEdit
			return m, m.hostEdit.Init()
		}
	case "a", "n":
		m.hostEdit = NewHostEditModel(m.cfg, nil)
		m.prevMode = modeLauncher
		m.mode = modeHostEdit
		return m, m.hostEdit.Init()
	case "d":
		if host := m.launcher.SelectedHost(); host != nil {
			m.deleteTarget = host.ID
			m.deleteTargetName = host.Name
			m.deleteKind = deleteHostKind
			m.prevMode = modeLauncher
			m.mode = modeConfirmDelete
			return m, nil
		}
	case "o":
		newSort := m.launcher.CycleSort()
		m.cfg.DefaultSort = newSort
		m.saveConfig()
		return m, nil
	case "s":
		m.settings = NewSettingsModel(m.cfg)
		m.settings.width = m.width
		m.settings.height = m.height
		m.settings.credList.width = m.width
		m.settings.credList.height = m.height
		m.settings.groupList.width = m.width
		m.settings.groupList.height = m.height
		m.prevMode = modeLauncher
		m.mode = modeSettings
		return m, m.settings.Init()
	case "p":
		if host := m.launcher.SelectedHost(); host != nil {
			status, cmd := m.copyHostPassword(host)
			m.launcher.SetStatus(status)
			m.tree.SetStatus(status)
			return m, cmd
		}
	case "P":
		if host := m.launcher.SelectedHost(); host != nil {
			m.openPasswordPopup(host, modeLauncher)
			return m, nil
		}
	case "ctrl+r":
		return m, m.doSync(false)
	case "q", "esc":
		return m, tea.Quit
	default:
		// Navigation keys (j/k/arrows/g/G) and anything else → launcher.
		var cmd tea.Cmd
		m.launcher, cmd = m.launcher.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m AppModel) treeKey(key string, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key {
	case "tab", "esc":
		m.mode = modeLauncher
		return m, nil
	case "enter":
		if host := m.tree.SelectedHost(); host != nil {
			m.ConnectRequest = &ConnectRequest{Host: *host}
			return m, tea.Quit
		}
	case "e":
		if host := m.tree.SelectedHost(); host != nil {
			m.hostEdit = NewHostEditModel(m.cfg, host)
			m.prevMode = modeTree
			m.mode = modeHostEdit
			return m, m.hostEdit.Init()
		}
	case "n":
		m.hostEdit = NewHostEditModel(m.cfg, nil)
		m.prevMode = modeTree
		m.mode = modeHostEdit
		return m, m.hostEdit.Init()
	case "d":
		if host := m.tree.SelectedHost(); host != nil {
			m.deleteTarget = host.ID
			m.deleteTargetName = host.Name
			m.deleteKind = deleteHostKind
			m.prevMode = modeTree
			m.mode = modeConfirmDelete
			return m, nil
		}
	case "p":
		if host := m.tree.SelectedHost(); host != nil {
			status, cmd := m.copyHostPassword(host)
			m.launcher.SetStatus(status)
			m.tree.SetStatus(status)
			return m, cmd
		}
	case "P":
		if host := m.tree.SelectedHost(); host != nil {
			m.openPasswordPopup(host, modeTree)
			return m, nil
		}
	case "g":
		m.groupList = NewGroupListModel(m.cfg)
		m.groupList.width = m.width
		m.groupList.height = m.height
		m.mode = modeGroupList
		return m, nil
	case "q":
		return m, tea.Quit
	default:
		var cmd tea.Cmd
		m.tree, cmd = m.tree.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m AppModel) formKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Esc cancels the form and returns to the previous view.
	// (huh's own cancel key is ctrl+c which we reserve for app quit.)
	if msg.String() == "esc" {
		m.mode = m.prevMode
		return m, nil
	}
	var cmd tea.Cmd
	m.hostEdit, cmd = m.hostEdit.Update(msg)
	if m.hostEdit.IsDone() {
		return m.finishHostEdit()
	} else if m.hostEdit.IsAborted() {
		m.mode = m.prevMode
	}
	return m, cmd
}

// finishHostEdit saves the completed host, unless its group is the inline
// "new group" sentinel — in which case it opens the group form first and
// remembers to attach the new group to the host afterward.
func (m AppModel) finishHostEdit() (AppModel, tea.Cmd) {
	host := m.hostEdit.Result()
	if host.GroupID == newGroupOption {
		host.GroupID = ""
		m.pendingHost = &host
		m.groupForHost = true
		m.groupEdit = NewGroupEditModel(m.cfg, nil)
		m.mode = modeGroupEdit
		return m, m.groupEdit.Init()
	}
	m.upsertHost(host)
	m.saveConfig()
	m.refreshSubmodels()
	m.launcher.SetStatus("Saved")
	m.mode = m.prevMode
	return m, nil
}

func (m AppModel) credListKey(key string, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key {
	case "esc", "q":
		m.mode = modeLauncher
		return m, nil
	case "n":
		m.credEdit = NewCredEditModel(nil)
		m.prevMode = modeCredList
		m.mode = modeCredEdit
		return m, m.credEdit.Init()
	case "e":
		if cred := m.credList.SelectedCred(); cred != nil {
			m.credEdit = NewCredEditModel(cred)
			m.prevMode = modeCredList
			m.mode = modeCredEdit
			return m, m.credEdit.Init()
		}
	case "d":
		if cred := m.credList.SelectedCred(); cred != nil {
			m.deleteTarget = cred.ID
			m.deleteTargetName = cred.Name
			m.deleteKind = deleteCredKind
			m.prevMode = modeCredList
			m.mode = modeConfirmDelete
			return m, nil
		}
	default:
		var cmd tea.Cmd
		m.credList, cmd = m.credList.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m AppModel) credFormKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" {
		m.mode = m.prevMode
		return m, nil
	}
	var cmd tea.Cmd
	m.credEdit, cmd = m.credEdit.Update(msg)
	if m.credEdit.IsDone() {
		cred := m.credEdit.Result()
		m.upsertCred(cred)
		m.saveConfig()
		m.refreshSubmodels()
		m.mode = m.prevMode
	} else if m.credEdit.IsAborted() {
		m.mode = m.prevMode
	}
	return m, cmd
}

func (m AppModel) groupListKey(key string, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key {
	case "esc", "q":
		m.mode = modeLauncher
		return m, nil
	case "n":
		m.groupEdit = NewGroupEditModel(m.cfg, nil)
		m.prevMode = modeGroupList
		m.mode = modeGroupEdit
		return m, m.groupEdit.Init()
	case "e":
		if g := m.groupList.SelectedGroup(); g != nil {
			m.groupEdit = NewGroupEditModel(m.cfg, g)
			m.prevMode = modeGroupList
			m.mode = modeGroupEdit
			return m, m.groupEdit.Init()
		}
	case "d":
		if g := m.groupList.SelectedGroup(); g != nil {
			m.deleteTarget = g.ID
			m.deleteTargetName = g.Name
			m.deleteKind = deleteGroupKind
			m.prevMode = modeGroupList
			m.mode = modeConfirmDelete
			return m, nil
		}
	default:
		var cmd tea.Cmd
		m.groupList, cmd = m.groupList.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m AppModel) groupFormKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" {
		return m.cancelGroupForm()
	}
	var cmd tea.Cmd
	m.groupEdit, cmd = m.groupEdit.Update(msg)
	if m.groupEdit.IsDone() {
		return m.finishGroupForm()
	} else if m.groupEdit.IsAborted() {
		return m.cancelGroupForm()
	}
	return m, cmd
}

// finishGroupForm saves the completed group. If the form was opened inline from
// the host editor, it attaches the new group to the pending host and saves that
// too, returning straight to the launcher.
func (m AppModel) finishGroupForm() (AppModel, tea.Cmd) {
	g := m.groupEdit.Result()
	m.upsertGroup(g)
	if m.groupForHost && m.pendingHost != nil {
		host := *m.pendingHost
		host.GroupID = g.ID
		m.upsertHost(host)
		m.groupForHost = false
		m.pendingHost = nil
		m.saveConfig()
		m.refreshSubmodels()
		m.launcher.SetStatus("Saved")
		m.mode = modeLauncher
		return m, nil
	}
	m.saveConfig()
	m.refreshSubmodels()
	m.mode = m.prevMode
	return m, nil
}

// cancelGroupForm backs out of the group form. When it was opened inline from
// the host editor, it reopens the host form (dropping the unsaved group choice)
// so the user doesn't lose their host edits.
func (m AppModel) cancelGroupForm() (AppModel, tea.Cmd) {
	if m.groupForHost && m.pendingHost != nil {
		host := *m.pendingHost
		host.GroupID = ""
		m.groupForHost = false
		m.pendingHost = nil
		m.hostEdit = NewHostEditModel(m.cfg, &host)
		m.mode = modeHostEdit
		return m, m.hostEdit.Init()
	}
	m.mode = m.prevMode
	return m, nil
}

func (m AppModel) settingsFormKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if key == "esc" {
		if m.settings.IsOnListTab() {
			m.settings = m.settings.SwitchToGeneralTab()
			return m, nil
		}
		m.mode = m.prevMode
		return m, nil
	}

	// CRUD on the Credentials tab
	if m.settings.IsOnCredTab() {
		switch key {
		case "n":
			m.credEdit = NewCredEditModel(nil)
			m.prevMode = modeSettings
			m.mode = modeCredEdit
			return m, m.credEdit.Init()
		case "e":
			if cred := m.settings.SelectedCred(); cred != nil {
				m.credEdit = NewCredEditModel(cred)
				m.prevMode = modeSettings
				m.mode = modeCredEdit
				return m, m.credEdit.Init()
			}
		case "d":
			if cred := m.settings.SelectedCred(); cred != nil {
				m.deleteTarget = cred.ID
				m.deleteTargetName = cred.Name
				m.deleteKind = deleteCredKind
				m.prevMode = modeSettings
				m.mode = modeConfirmDelete
				return m, nil
			}
		}
	}

	// CRUD on the Groups tab
	if m.settings.IsOnGroupsTab() {
		switch key {
		case "n":
			m.groupEdit = NewGroupEditModel(m.cfg, nil)
			m.prevMode = modeSettings
			m.mode = modeGroupEdit
			return m, m.groupEdit.Init()
		case "e":
			if g := m.settings.SelectedGroup(); g != nil {
				m.groupEdit = NewGroupEditModel(m.cfg, g)
				m.prevMode = modeSettings
				m.mode = modeGroupEdit
				return m, m.groupEdit.Init()
			}
		case "d":
			if g := m.settings.SelectedGroup(); g != nil {
				m.deleteTarget = g.ID
				m.deleteTargetName = g.Name
				m.deleteKind = deleteGroupKind
				m.prevMode = modeSettings
				m.mode = modeConfirmDelete
				return m, nil
			}
		}
	}

	var cmd tea.Cmd
	m.settings, cmd = m.settings.Update(msg)
	if m.settings.IsDone() {
		m.cfg.Sync = m.settings.Result()
		m.cfg.ClipboardMode = m.settings.Clipboard()
		m.saveConfig()
		m.mode = m.prevMode
	} else if m.settings.IsAborted() {
		m.mode = m.prevMode
	}
	return m, cmd
}

func (m AppModel) confirmDeleteKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "y", "Y":
		switch m.deleteKind {
		case deleteHostKind:
			m.deleteHost(m.deleteTarget)
		case deleteCredKind:
			m.deleteCred(m.deleteTarget)
		case deleteGroupKind:
			m.deleteGroup(m.deleteTarget)
		}
		m.saveConfig()
		m.refreshSubmodels()
		m.launcher.SetStatus(fmt.Sprintf("Deleted %q", m.deleteTargetName))
	}
	m.mode = m.prevMode
	return m, nil
}

// delegateUpdate routes non-key messages (timers, blink ticks, and the huh
// forms' internal nextFieldMsg/nextGroupMsg commands) to whichever view is
// currently active. Without this, those messages are dropped and forms can't
// advance between fields.
func (m AppModel) delegateUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.mode {
	case modeLauncher:
		m.launcher, cmd = m.launcher.Update(msg)
	case modeTree:
		m.tree, cmd = m.tree.Update(msg)
	case modeHostEdit:
		m.hostEdit, cmd = m.hostEdit.Update(msg)
		if m.hostEdit.IsDone() {
			return m.finishHostEdit()
		} else if m.hostEdit.IsAborted() {
			m.mode = m.prevMode
		}
	case modeGroupEdit:
		m.groupEdit, cmd = m.groupEdit.Update(msg)
		if m.groupEdit.IsDone() {
			return m.finishGroupForm()
		} else if m.groupEdit.IsAborted() {
			return m.cancelGroupForm()
		}
	case modeGroupList:
		m.groupList, cmd = m.groupList.Update(msg)
	case modeCredEdit:
		m.credEdit, cmd = m.credEdit.Update(msg)
		if m.credEdit.IsDone() {
			m.upsertCred(m.credEdit.Result())
			m.saveConfig()
			m.refreshSubmodels()
			m.mode = m.prevMode
		} else if m.credEdit.IsAborted() {
			m.mode = m.prevMode
		}
	case modeSettings:
		m.settings, cmd = m.settings.Update(msg)
		if m.settings.IsDone() {
			m.cfg.Sync = m.settings.Result()
			m.cfg.ClipboardMode = m.settings.Clipboard()
			m.saveConfig()
			m.mode = m.prevMode
		} else if m.settings.IsAborted() {
			m.mode = m.prevMode
		}
		return m, cmd
	case modeCredList:
		m.credList, cmd = m.credList.Update(msg)
	}
	return m, cmd
}

func (m AppModel) View() string {
	if m.width == 0 {
		return ""
	}
	switch m.mode {
	case modeTree:
		return m.tree.View()
	case modeHostEdit:
		return m.centeredForm(m.hostEdit.View())
	case modeCredList:
		return m.credList.View()
	case modeCredEdit:
		return m.centeredForm(m.credEdit.View())
	case modeGroupList:
		return m.groupList.View()
	case modeGroupEdit:
		return m.centeredForm(m.groupEdit.View())
	case modeSettings:
		return m.settings.View()
	case modeConfirmDelete:
		return m.confirmDeleteView()
	case modeSyncOverlay:
		return m.syncOverlayView()
	case modePasswordPopup:
		return m.passwordPopupView()
	default:
		return m.launcher.View()
	}
}

func (m AppModel) centeredForm(content string) string {
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(colorMauve)).
		Padding(1, 2).
		Render(content)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func (m AppModel) confirmDeleteView() string {
	prompt := fmt.Sprintf("Delete %q?  [y] yes  [any other key] cancel", m.deleteTargetName)
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(colorRed)).
		Padding(1, 3).
		Render(styleError.Render(prompt))
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func (m AppModel) syncOverlayView() string {
	style := styleSuccess
	prefix := "✓  "
	if m.syncErr {
		style = styleError
		prefix = "✗  "
	}
	lines := style.Render(prefix+m.syncOutput) + "\n\n" + styleMuted.Render("Press any key to continue")
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(colorOverlay)).
		Padding(1, 3).
		Render(lines)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func (m *AppModel) openPasswordPopup(host *model.Host, prev viewMode) {
	cred := m.cfg.EffectiveCredential(host)
	pw := ""
	if cred != nil {
		pw = cred.Password
	}
	m.popupHostName = host.Name
	m.popupPassword = pw
	m.prevMode = prev
	m.mode = modePasswordPopup
}

func (m AppModel) passwordPopupKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "p":
		if m.popupPassword != "" && m.cfg.ClipboardMode != "off" {
			_ = connect.CopyPassword(m.popupPassword, m.cfg.ClipboardMode)
			m.launcher.SetStatus("Password copied to clipboard")
			m.tree.SetStatus("Password copied to clipboard")
			m.mode = m.prevMode
			return m, clearStatusAfter(3 * time.Second)
		}
	}
	m.mode = m.prevMode
	return m, nil
}

func (m AppModel) passwordPopupView() string {
	pw := m.popupPassword
	if pw == "" {
		pw = "(no password configured)"
	}
	content := styleOverlayTitle.Render(m.popupHostName) + "\n\n" +
		styleDetailLabel.Render("Password:  ") + pw + "\n\n" +
		styleMuted.Render("p copy   any key close")
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(colorMauve)).
		Padding(1, 3).
		Render(content)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func (m *AppModel) copyHostPassword(host *model.Host) (string, tea.Cmd) {
	if m.cfg.ClipboardMode == "off" {
		return "Clipboard is disabled", nil
	}
	cred := m.cfg.EffectiveCredential(host)
	if cred == nil || cred.Password == "" {
		return "No password configured", nil
	}
	if err := connect.CopyPassword(cred.Password, m.cfg.ClipboardMode); err != nil {
		return "Clipboard error: " + err.Error(), nil
	}
	return "Password copied to clipboard", clearStatusAfter(3 * time.Second)
}

func clearStatusAfter(d time.Duration) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(d)
		return clearStatusMsg{}
	}
}

// doSync runs rclone in a background goroutine and delivers a syncDoneMsg.
// When silent is true the overlay is suppressed on success.
func (m *AppModel) doSync(silent bool) tea.Cmd {
	cfgPath := m.cfgPath
	remote := m.cfg.Sync.Remote
	direction := m.cfg.Sync.Direction

	if remote == "" {
		return func() tea.Msg {
			return syncDoneMsg{err: fmt.Errorf("no rclone remote configured — open Settings (s) to set one")}
		}
	}

	return func() tea.Msg {
		var result shellsync.Result
		switch direction {
		case "pull":
			result = shellsync.Pull(remote, cfgPath)
		case "sync":
			result = shellsync.Sync(cfgPath, remote)
		default:
			result = shellsync.Push(cfgPath, remote)
		}
		return syncDoneMsg{output: result.Output, err: result.Err, silent: silent}
	}
}

// --- Config mutation helpers ---

func (m *AppModel) upsertHost(h model.Host) {
	for i, existing := range m.cfg.Hosts {
		if existing.ID == h.ID {
			m.cfg.Hosts[i] = h
			return
		}
	}
	m.cfg.Hosts = append(m.cfg.Hosts, h)
}

func (m *AppModel) upsertCred(c model.Credential) {
	for i, existing := range m.cfg.Credentials {
		if existing.ID == c.ID {
			m.cfg.Credentials[i] = c
			return
		}
	}
	m.cfg.Credentials = append(m.cfg.Credentials, c)
}

func (m *AppModel) deleteHost(id string) {
	out := m.cfg.Hosts[:0]
	for _, h := range m.cfg.Hosts {
		if h.ID != id {
			out = append(out, h)
		}
	}
	m.cfg.Hosts = out
}

func (m *AppModel) deleteCred(id string) {
	out := m.cfg.Credentials[:0]
	for _, c := range m.cfg.Credentials {
		if c.ID != id {
			out = append(out, c)
		}
	}
	m.cfg.Credentials = out
}

func (m *AppModel) upsertGroup(g model.Group) {
	for i, existing := range m.cfg.Groups {
		if existing.ID == g.ID {
			m.cfg.Groups[i] = g
			return
		}
	}
	m.cfg.Groups = append(m.cfg.Groups, g)
}

// deleteGroup removes a group and re-parents its children and member hosts up to
// the deleted group's own parent, so nothing is orphaned or left dangling.
func (m *AppModel) deleteGroup(id string) {
	var parent string
	for _, g := range m.cfg.Groups {
		if g.ID == id {
			parent = g.ParentID
			break
		}
	}
	out := m.cfg.Groups[:0]
	for _, g := range m.cfg.Groups {
		if g.ID == id {
			continue
		}
		if g.ParentID == id {
			g.ParentID = parent
		}
		out = append(out, g)
	}
	m.cfg.Groups = out
	for i := range m.cfg.Hosts {
		if m.cfg.Hosts[i].GroupID == id {
			m.cfg.Hosts[i].GroupID = parent
		}
	}
}

func (m *AppModel) refreshSubmodels() {
	m.launcher.SetConfig(m.cfg)
	m.tree.SetConfig(m.cfg)
	m.groupList.SetConfig(m.cfg)
	m.settings.credList.SetConfig(m.cfg)
	m.settings.groupList.SetConfig(m.cfg)
}

func (m *AppModel) saveConfig() {
	if m.saveFn != nil {
		_ = m.saveFn(m.cfg)
	}
}
