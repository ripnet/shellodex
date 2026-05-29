package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ripnet/shellodex/internal/model"
)

// treeLineCount returns how many rendered lines a tree node occupies.
func treeLineCount(n *treeNode) int {
	return 1
}

type treeNode struct {
	group    *model.Group
	host     *model.Host
	children []*treeNode
	expanded bool
	depth    int
}

type TreeModel struct {
	cfg    *model.Config
	roots  []*treeNode
	flat   []*treeNode // flattened visible nodes for cursor navigation
	cursor int
	width  int
	height int
}

func NewTreeModel(cfg *model.Config) TreeModel {
	m := TreeModel{cfg: cfg}
	m.rebuild()
	return m
}

func (m *TreeModel) SetConfig(cfg *model.Config) {
	m.cfg = cfg
	m.rebuild()
}

func (m TreeModel) Update(msg tea.Msg) (TreeModel, tea.Cmd) {
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
			if m.cursor < len(m.flat)-1 {
				m.cursor++
			}
		case " ", "right":
			m.toggleExpand()
		case "left":
			m.collapseOrParent()
		}
	}
	return m, nil
}

func (m TreeModel) SelectedHost() *model.Host {
	if m.cursor >= len(m.flat) {
		return nil
	}
	return m.flat[m.cursor].host
}

func (m TreeModel) View() string {
	if m.width == 0 {
		return ""
	}
	treeW := m.width / 3
	if treeW < 20 {
		treeW = 20
	}
	detailW := m.width - treeW - 1

	treePane := m.renderTree(treeW)
	detailPane := m.renderDetail(detailW)

	// No trailing newline — avoids adding an extra blank row that would push the header off screen.
	divLines := m.height - 2
	if divLines < 1 {
		divLines = 1
	}
	divider := styleDivider.Render(strings.Repeat("│\n", divLines-1) + "│")

	row := lipgloss.JoinHorizontal(lipgloss.Top, treePane, divider, detailPane)

	title := styleHeader.Width(m.width).Render("  shellodex  —  tree view")
	status := styleStatusBar.Width(m.width).Render(
		fmt.Sprintf("%s back  %s connect  %s edit  %s groups  %s expand/collapse",
			styleStatusKey.Render("tab"),
			styleStatusKey.Render("enter"),
			styleStatusKey.Render("e"),
			styleStatusKey.Render("g"),
			styleStatusKey.Render("space"),
		),
	)
	return lipgloss.JoinVertical(lipgloss.Left, title, row, status)
}

func (m TreeModel) renderTree(w int) string {
	listH := m.height - 2
	if listH < 1 {
		listH = 1
	}

	// Build a slice of rendered line-blocks, one per visible node.
	// Count lines used so far to find which nodes fit.
	type block struct {
		lines []string
	}
	var blocks []block
	linesUsed := 0

	// Scroll: find starting node so cursor is always visible.
	// Walk forward from node 0; stop when we've consumed enough lines that
	// the cursor node's last line would be > listH.
	start := 0
	{
		// Find the minimal start such that cursor fits in listH lines.
		linesBefore := 0
		for i := 0; i < m.cursor; i++ {
			linesBefore += treeLineCount(m.flat[i])
		}
		cursorLines := treeLineCount(m.flat[m.cursor])
		if linesBefore+cursorLines > listH {
			// Shrink start until cursor fits.
			acc := 0
			for i := m.cursor; i >= 0; i-- {
				acc += treeLineCount(m.flat[i])
				if acc >= listH {
					start = i + 1
					break
				}
			}
		}
	}

	groupStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(colorMauve))

	for i := start; i < len(m.flat) && linesUsed < listH; i++ {
		node := m.flat[i]
		selected := i == m.cursor
		indent := strings.Repeat("  ", node.depth)

		var b block
		if node.group != nil {
			marker := "▶ "
			if node.expanded {
				marker = "▼ "
			}
			name := indent + marker + node.group.Name
			name = plainPad(name, w)
			if selected {
				b.lines = []string{styleSelected.Width(w).Render(name)}
			} else {
				b.lines = []string{groupStyle.Width(w).Render(name)}
			}
		} else {
			h := node.host
			namePart := indent + "  " + h.Name
			hostnameW := len([]rune(h.Hostname))
			nameW := w - hostnameW - 1
			if nameW < 1 {
				nameW = 1
			}
			if selected {
				line := plainPad(namePart, nameW) + " " + h.Hostname
				b.lines = []string{styleSelected.Width(w).Render(plainPad(line, w))}
			} else {
				styledLine := plainPad(indent+"  "+styleNormal.Render(h.Name), nameW) + " " + styleMuted.Render(h.Hostname)
				b.lines = []string{styledLine}
			}
		}
		blocks = append(blocks, b)
		for _, l := range b.lines {
			_ = l
			linesUsed++
		}
	}

	var sb strings.Builder
	rendered := 0
	for bi, blk := range blocks {
		for li, line := range blk.lines {
			if rendered > 0 {
				sb.WriteRune('\n')
			}
			sb.WriteString(line)
			rendered++
			_ = li
		}
		_ = bi
	}
	// Fill remaining lines
	for rendered < listH {
		sb.WriteRune('\n')
		rendered++
	}
	return sb.String()
}


func (m TreeModel) renderDetail(w int) string {
	if m.cursor >= len(m.flat) {
		return lipgloss.NewStyle().Width(w).Height(m.height - 2).Render("")
	}
	node := m.flat[m.cursor]

	if node.group != nil {
		return m.renderGroupDetail(node, w)
	}
	return m.renderHostDetail(node.host, w)
}

func (m TreeModel) renderHostDetail(host *model.Host, w int) string {
	cred := m.cfg.CredentialByID(host.CredentialID)
	jump := m.cfg.HostByID(host.JumpHostID)

	port := host.Port
	if port == 0 {
		port = int(model.DefaultPort(host.Protocol))
	}

	// Effective username
	username := host.Username
	if username == "" && cred != nil {
		username = cred.Username
	}

	lines := []string{
		styleOverlayTitle.Render(host.Name),
		"",
		row("Protocol", string(host.Protocol)),
		row("Hostname", host.Hostname),
		row("Port", fmt.Sprintf("%d", port)),
	}
	if username != "" {
		lines = append(lines, row("Username", username))
	}
	if cred != nil {
		lines = append(lines, row("Credential", cred.Name))
	}
	if jump != nil {
		lines = append(lines, row("Jump via", jump.Name+" ("+jump.Hostname+")"))
	}
	lines = append(lines, row("Last connected", relativeTime(host.LastConnected)))
	if len(host.Tags) > 0 {
		var tagLine strings.Builder
		for i, tag := range host.Tags {
			if i > 0 {
				tagLine.WriteString("  ")
			}
			tagLine.WriteString(styleTagBadge.Render(tag))
		}
		lines = append(lines, "", styleDetailLabel.Render("Tags:"), "  "+tagLine.String())
	}
	if host.Notes != "" {
		lines = append(lines, "", styleDetailLabel.Render("Notes:"), styleDetailValue.Width(w-2).Render(host.Notes))
	}

	return lipgloss.NewStyle().Width(w).Padding(0, 1).Render(strings.Join(lines, "\n"))
}

func (m TreeModel) renderGroupDetail(node *treeNode, w int) string {
	g := node.group
	// Count direct and total hosts
	direct, total := 0, 0
	var memberNames []string
	for _, h := range m.cfg.Hosts {
		if h.GroupID == g.ID {
			direct++
			memberNames = append(memberNames, h.Name)
		}
	}
	// Count all descendants via config
	for _, h := range m.cfg.Hosts {
		gid := h.GroupID
		for gid != "" {
			if gid == g.ID {
				total++
				break
			}
			// Walk up
			parent := m.cfg.GroupByID(gid)
			if parent == nil {
				break
			}
			gid = parent.ParentID
		}
	}

	lines := []string{
		styleOverlayTitle.Render(g.Name),
		"",
		row("Path", m.cfg.GroupPath(g.ID)),
		row("Direct hosts", fmt.Sprintf("%d", direct)),
	}
	if total != direct {
		lines = append(lines, row("Total hosts", fmt.Sprintf("%d", total)))
	}
	if len(memberNames) > 0 {
		lines = append(lines, "", styleDetailLabel.Render("Members:"))
		for _, name := range memberNames {
			lines = append(lines, "  "+styleDetailValue.Render(name))
		}
	}
	return lipgloss.NewStyle().Width(w).Padding(0, 1).Render(strings.Join(lines, "\n"))
}

func row(label, value string) string {
	return styleDetailLabel.Render(label+":") + styleDetailValue.Render(value)
}

func (m *TreeModel) rebuild() {
	// Build group map
	groupChildren := make(map[string][]*treeNode)
	for i := range m.cfg.Groups {
		g := &m.cfg.Groups[i]
		node := &treeNode{group: g, expanded: true}
		groupChildren[g.ParentID] = append(groupChildren[g.ParentID], node)
	}
	// Build host map by group
	hostsByGroup := make(map[string][]*treeNode)
	for i := range m.cfg.Hosts {
		h := &m.cfg.Hosts[i]
		node := &treeNode{host: h}
		hostsByGroup[h.GroupID] = append(hostsByGroup[h.GroupID], node)
	}
	// Recursively attach children
	var build func(nodes []*treeNode, depth int)
	build = func(nodes []*treeNode, depth int) {
		for _, n := range nodes {
			n.depth = depth
			if n.group != nil {
				kids := groupChildren[n.group.ID]
				build(kids, depth+1)
				n.children = append(kids, hostsByGroup[n.group.ID]...)
				for _, h := range hostsByGroup[n.group.ID] {
					h.depth = depth + 1
				}
			}
		}
	}
	roots := groupChildren[""]
	build(roots, 0)
	// Add ungrouped hosts at root
	ungrouped := hostsByGroup[""]
	for _, n := range ungrouped {
		n.depth = 0
	}
	m.roots = append(roots, ungrouped...)
	m.flatten()
}

func (m *TreeModel) flatten() {
	m.flat = m.flat[:0]
	var walk func(nodes []*treeNode)
	walk = func(nodes []*treeNode) {
		for _, n := range nodes {
			m.flat = append(m.flat, n)
			if n.group != nil && n.expanded {
				walk(n.children)
			}
		}
	}
	walk(m.roots)
	if m.cursor >= len(m.flat) {
		m.cursor = max(0, len(m.flat)-1)
	}
}

func (m *TreeModel) toggleExpand() {
	if m.cursor >= len(m.flat) {
		return
	}
	node := m.flat[m.cursor]
	if node.group != nil {
		node.expanded = !node.expanded
		m.flatten()
	}
}

func (m *TreeModel) collapseOrParent() {
	if m.cursor >= len(m.flat) {
		return
	}
	node := m.flat[m.cursor]
	if node.group != nil && node.expanded {
		node.expanded = false
		m.flatten()
		return
	}
	// Move cursor to parent
	depth := node.depth
	for i := m.cursor - 1; i >= 0; i-- {
		if m.flat[i].depth < depth {
			m.cursor = i
			break
		}
	}
}
