package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/ripnet/shellodex/internal/model"
)

// groupFormData is heap-allocated and referenced via pointer so huh's value
// bindings survive Bubble Tea's value copies. See the note on hostFormData.
type groupFormData struct {
	name     string
	parentID string
}

type GroupEditModel struct {
	form   *huh.Form
	cfg    *model.Config
	data   *groupFormData
	isNew  bool
	origID string
}

func NewGroupEditModel(cfg *model.Config, existing *model.Group) GroupEditModel {
	m := GroupEditModel{cfg: cfg, data: &groupFormData{}}
	if existing != nil {
		m.origID = existing.ID
		m.data.name = existing.Name
		m.data.parentID = existing.ParentID
	} else {
		m.isNew = true
	}
	m.form = m.buildForm()
	return m
}

func (m GroupEditModel) Init() tea.Cmd {
	return m.form.Init()
}

func (m GroupEditModel) Update(msg tea.Msg) (GroupEditModel, tea.Cmd) {
	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
	}
	return m, cmd
}

func (m GroupEditModel) View() string {
	return m.form.View()
}

func (m GroupEditModel) IsDone() bool {
	return m.form.State == huh.StateCompleted
}

func (m GroupEditModel) IsAborted() bool {
	return m.form.State == huh.StateAborted
}

func (m GroupEditModel) Result() model.Group {
	d := m.data
	id := m.origID
	if id == "" {
		id = model.NewID()
	}
	return model.Group{
		ID:       id,
		Name:     d.name,
		ParentID: d.parentID,
	}
}

func (m *GroupEditModel) buildForm() *huh.Form {
	d := m.data

	// Parent options exclude the group itself and any of its descendants, so a
	// group can never become its own ancestor (which would make GroupPath loop).
	parentOpts := []huh.Option[string]{huh.NewOption("(top level)", "")}
	for _, g := range m.cfg.Groups {
		if m.isSelfOrDescendant(g.ID) {
			continue
		}
		path := m.cfg.GroupPath(g.ID)
		if path == "" {
			path = g.Name
		}
		parentOpts = append(parentOpts, huh.NewOption(path, g.ID))
	}

	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Name").
				Description("e.g. Lab Core").
				Value(&d.name),
			huh.NewSelect[string]().
				Title("Parent group").
				Description("Nest this group under another, or keep it top level").
				Options(parentOpts...).
				Value(&d.parentID),
		),
	).WithTheme(huh.ThemeCatppuccin()).
		WithWidth(60)
}

// isSelfOrDescendant reports whether candidateID is the group being edited or
// nested somewhere beneath it.
func (m *GroupEditModel) isSelfOrDescendant(candidateID string) bool {
	if m.origID == "" {
		return false // new group has no descendants yet
	}
	id := candidateID
	visited := map[string]bool{}
	for id != "" && !visited[id] {
		if id == m.origID {
			return true
		}
		visited[id] = true
		id = m.parentOf(id)
	}
	return false
}

func (m *GroupEditModel) parentOf(id string) string {
	for _, g := range m.cfg.Groups {
		if g.ID == id {
			return g.ParentID
		}
	}
	return ""
}
