package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ripnet/shellodex/internal/model"
)

func testConfig() *model.Config {
	return &model.Config{
		Version: 1,
		Groups:  []model.Group{{ID: "g1", Name: "Lab Core"}},
		Hosts: []model.Host{
			{ID: "h1", Name: "Router-01", GroupID: "g1", Protocol: model.SSH, Hostname: "10.0.0.1", Port: 22},
			{ID: "h2", Name: "Switch-01", GroupID: "g1", Protocol: model.SSH, Hostname: "10.0.0.2", Port: 22},
			{ID: "h3", Name: "Bastion", Protocol: model.SSH, Hostname: "bastion.lab", Port: 22},
		},
	}
}

func rune1(r rune) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}} }

func newTestApp() AppModel {
	app := NewAppModel(testConfig(), "/tmp/none.json", func(*model.Config) error { return nil })
	app.width, app.height = 120, 30
	app.launcher.width, app.launcher.height = 120, 30
	return app
}

// step sends one message and returns the updated AppModel.
func step(m AppModel, msg tea.Msg) AppModel {
	next, _ := m.Update(msg)
	return next.(AppModel)
}

func TestBrowseNavigationWithJK(t *testing.T) {
	m := newTestApp()
	if m.launcher.cursor != 0 {
		t.Fatalf("initial cursor = %d, want 0", m.launcher.cursor)
	}
	m = step(m, rune1('j'))
	m = step(m, rune1('j'))
	if m.launcher.cursor != 2 {
		t.Fatalf("after jj cursor = %d, want 2", m.launcher.cursor)
	}
	m = step(m, rune1('k'))
	if m.launcher.cursor != 1 {
		t.Fatalf("after k cursor = %d, want 1", m.launcher.cursor)
	}
}

func TestSlashEntersSearchAndTypingDoesNotTriggerHotkeys(t *testing.T) {
	m := newTestApp()
	if m.launcher.IsSearching() {
		t.Fatal("should not start in search mode")
	}
	m = step(m, rune1('/'))
	if !m.launcher.IsSearching() {
		t.Fatal("'/' should enter search mode")
	}
	// Typing "bas" must filter to Bastion, NOT trigger the 'a' (add) hotkey.
	m = step(m, rune1('b'))
	m = step(m, rune1('a'))
	m = step(m, rune1('s'))
	if m.mode != modeLauncher {
		t.Fatalf("typing in search must not change mode; mode=%d", m.mode)
	}
	if got := len(m.launcher.filtered); got != 1 {
		t.Fatalf("filtered = %d, want 1 (Bastion)", got)
	}
	if h := m.launcher.SelectedHost(); h == nil || h.Name != "Bastion" {
		t.Fatalf("selected = %v, want Bastion", h)
	}
}

func TestEscExitsSearchToBrowse(t *testing.T) {
	m := newTestApp()
	m = step(m, rune1('/'))
	m = step(m, rune1('b'))
	m = step(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.launcher.IsSearching() {
		t.Fatal("Esc should exit search mode")
	}
	if got := len(m.launcher.filtered); got != 3 {
		t.Fatalf("after Esc filtered = %d, want 3 (full list)", got)
	}
}

// Searching for a host then pressing Esc must leave the cursor ON that host so
// that browse-mode hotkeys act on it.
func TestEscKeepsSelectionThenEdits(t *testing.T) {
	m := newTestApp()
	m = step(m, rune1('/'))
	// "switch" matches only Switch-01.
	for _, r := range "switch" {
		m = step(m, rune1(r))
	}
	if h := m.launcher.SelectedHost(); h == nil || h.Name != "Switch-01" {
		t.Fatalf("in-search selected = %v, want Switch-01", h)
	}
	m = step(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.launcher.IsSearching() {
		t.Fatal("Esc should exit search mode")
	}
	if h := m.launcher.SelectedHost(); h == nil || h.Name != "Switch-01" {
		t.Fatalf("after Esc selected = %v, want Switch-01 (selection preserved)", h)
	}
	// Now the edit hotkey must open the form for the host we searched for.
	m = step(m, rune1('e'))
	if m.mode != modeHostEdit {
		t.Fatalf("'e' after Esc should open host edit; mode=%d", m.mode)
	}
	if got := m.hostEdit.Result().Name; got != "Switch-01" {
		t.Fatalf("editing host = %q, want Switch-01", got)
	}
}

func TestBrowseHotkeyAddOpensForm(t *testing.T) {
	m := newTestApp()
	m = step(m, rune1('a'))
	if m.mode != modeHostEdit {
		t.Fatalf("'a' in browse should open host edit form; mode=%d", m.mode)
	}
}

func TestBrowseHotkeyDisabledInSearch(t *testing.T) {
	m := newTestApp()
	m = step(m, rune1('/'))
	m = step(m, rune1('e')) // would be "edit" in browse, but must be query text here
	if m.mode != modeLauncher {
		t.Fatalf("'e' in search must stay in launcher; mode=%d", m.mode)
	}
	if m.launcher.input.Value() != "e" {
		t.Fatalf("query = %q, want \"e\"", m.launcher.input.Value())
	}
}

func TestEnterConnectsSelectedHost(t *testing.T) {
	m := newTestApp()
	m = step(m, rune1('j')) // select Switch-01
	m = step(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.ConnectRequest == nil {
		t.Fatal("Enter should set a ConnectRequest")
	}
	if m.ConnectRequest.Host.Name != "Switch-01" {
		t.Fatalf("connect host = %q, want Switch-01", m.ConnectRequest.Host.Name)
	}
}

func TestGroupHotkeyOpensManager(t *testing.T) {
	m := newTestApp()
	m = step(m, rune1('g'))
	if m.mode != modeGroupList {
		t.Fatalf("'g' should open the group manager; mode=%d", m.mode)
	}
}

func TestDeleteGroupReparentsChildrenAndHosts(t *testing.T) {
	cfg := &model.Config{
		Groups: []model.Group{
			{ID: "root", Name: "Root"},
			{ID: "mid", Name: "Mid", ParentID: "root"},
			{ID: "leaf", Name: "Leaf", ParentID: "mid"},
		},
		Hosts: []model.Host{
			{ID: "h1", Name: "H1", GroupID: "mid"},
		},
	}
	app := NewAppModel(cfg, "/tmp/none.json", func(*model.Config) error { return nil })
	app.deleteGroup("mid")

	for _, g := range cfg.Groups {
		if g.ID == "mid" {
			t.Fatal("mid should be deleted")
		}
		if g.ID == "leaf" && g.ParentID != "root" {
			t.Fatalf("leaf reparented to %q, want root", g.ParentID)
		}
	}
	if cfg.Hosts[0].GroupID != "root" {
		t.Fatalf("host reparented to %q, want root", cfg.Hosts[0].GroupID)
	}
}

// The host form's "➕ New group…" sentinel must divert to the group editor and,
// once a group is created, attach it to the pending host and save.
func TestInlineNewGroupFlow(t *testing.T) {
	m := newTestApp()
	h := &model.Host{ID: "h9", Name: "OneOff", Protocol: model.SSH, Hostname: "1.2.3.4", Port: 22, GroupID: newGroupOption}
	m.hostEdit = NewHostEditModel(m.cfg, h)
	m.prevMode = modeLauncher
	m.mode = modeHostEdit

	next, _ := m.finishHostEdit()
	m = next
	if m.mode != modeGroupEdit {
		t.Fatalf("sentinel group should open the group editor; mode=%d", m.mode)
	}
	if !m.groupForHost || m.pendingHost == nil || m.pendingHost.GroupID != "" {
		t.Fatalf("inline group state wrong: forHost=%v pending=%v", m.groupForHost, m.pendingHost)
	}

	m.groupEdit = NewGroupEditModel(m.cfg, &model.Group{ID: "ng", Name: "New"})
	next, _ = m.finishGroupForm()
	m = next
	if m.mode != modeLauncher {
		t.Fatalf("after inline group create should return to launcher; mode=%d", m.mode)
	}
	if saved := m.cfg.HostByID("h9"); saved == nil || saved.GroupID != "ng" {
		t.Fatalf("host not attached to new group: %+v", saved)
	}
}

func TestArrowKeysNavigateInBrowse(t *testing.T) {
	m := newTestApp()
	m = step(m, tea.KeyMsg{Type: tea.KeyDown})
	if m.launcher.cursor != 1 {
		t.Fatalf("Down cursor = %d, want 1", m.launcher.cursor)
	}
	m = step(m, tea.KeyMsg{Type: tea.KeyUp})
	if m.launcher.cursor != 0 {
		t.Fatalf("Up cursor = %d, want 0", m.launcher.cursor)
	}
}
