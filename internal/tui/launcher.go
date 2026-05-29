package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ripnet/shellodex/internal/model"
)

type hostEntry struct {
	host      model.Host
	groupPath string
	username  string // effective username resolved from credential or inline fields
}

// LauncherModel is the primary view. It is modal:
//   - Browse mode (default): single-letter hotkeys + j/k navigation.
//   - Search mode (entered with "/"): typing filters the list; Esc returns
//     to browse mode. This avoids the collision between typing a query and
//     triggering action hotkeys.
type LauncherModel struct {
	cfg       *model.Config
	all       []hostEntry
	filtered  []hostEntry
	cursor    int
	input     textinput.Model
	searching bool
	width     int
	height    int
	// statusMsg is a transient message shown in the status bar (e.g. "Saved")
	statusMsg string
	// updateNotice is set when a newer version is available
	updateNotice string
}

func NewLauncherModel(cfg *model.Config) LauncherModel {
	ti := textinput.New()
	ti.Placeholder = "filter hosts…"
	ti.PromptStyle = styleSearchPrompt
	ti.TextStyle = styleNormal
	ti.PlaceholderStyle = styleMuted
	ti.Prompt = "/ "

	m := LauncherModel{
		cfg:   cfg,
		input: ti,
	}
	m.rebuildAll()
	m.refilter()
	return m
}

func (m *LauncherModel) SetConfig(cfg *model.Config) {
	m.cfg = cfg
	m.rebuildAll()
	m.refilter()
}

func (m *LauncherModel) SetStatus(msg string) {
	m.statusMsg = msg
}

func (m *LauncherModel) SetUpdateNotice(version string) {
	m.updateNotice = version
}

// IsSearching reports whether the launcher is in search mode. The app uses
// this to decide whether single-letter keys are hotkeys or query text.
func (m LauncherModel) IsSearching() bool {
	return m.searching
}

// StartSearch enters search mode and focuses the input.
func (m *LauncherModel) StartSearch() tea.Cmd {
	m.searching = true
	m.statusMsg = ""
	return m.input.Focus()
}

// stopSearch exits search mode, clears the query, and shows the full list —
// but keeps the cursor on whatever host was highlighted, so browse-mode hotkeys
// (edit/delete/…) act on the host the user just searched for.
func (m *LauncherModel) stopSearch() {
	var keepID string
	if h := m.SelectedHost(); h != nil {
		keepID = h.ID
	}
	m.searching = false
	m.input.Blur()
	m.input.Reset()
	m.refilter()
	if keepID != "" {
		for i, e := range m.filtered {
			if e.host.ID == keepID {
				m.cursor = i
				break
			}
		}
	}
}

func (m LauncherModel) Update(msg tea.Msg) (LauncherModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		key := msg.String()
		if m.searching {
			switch key {
			case "esc":
				m.stopSearch()
				return m, nil
			case "up", "ctrl+p":
				m.moveCursor(-1)
				return m, nil
			case "down", "ctrl+n":
				m.moveCursor(1)
				return m, nil
			default:
				// Everything else is query text.
				prev := m.input.Value()
				var cmd tea.Cmd
				m.input, cmd = m.input.Update(msg)
				if m.input.Value() != prev {
					m.refilter()
				}
				return m, cmd
			}
		}

		// Browse mode: navigation only (action hotkeys are handled by the app).
		// Note: g/G are intentionally NOT bound here — "g" is the Groups hotkey.
		switch key {
		case "up", "k", "ctrl+p":
			m.moveCursor(-1)
		case "down", "j", "ctrl+n":
			m.moveCursor(1)
		case "home":
			m.cursor = 0
		case "end":
			m.cursor = max(0, len(m.filtered)-1)
		}
		return m, nil
	}
	return m, nil
}

func (m *LauncherModel) moveCursor(delta int) {
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor > len(m.filtered)-1 {
		m.cursor = max(0, len(m.filtered)-1)
	}
}

func (m LauncherModel) SelectedHost() *model.Host {
	if len(m.filtered) == 0 {
		return nil
	}
	h := m.filtered[m.cursor].host
	return &h
}

func (m LauncherModel) View() string {
	if m.width == 0 {
		return ""
	}
	w := m.width

	var title, secondRow string
	if m.searching {
		title = styleHeader.Width(w).Render("shellodex  —  search")
		secondRow = styleSearchBar.Width(w).Render(m.input.View())
	} else {
		title = styleHeader.Width(w).Render("shellodex")
		secondRow = styleSearchBar.Width(w).Render(
			styleMuted.Render("press / to search"))
	}

	div := styleDivider.Render(strings.Repeat("─", w))

	cols := m.colLayout(w)
	colHeaders := m.renderColHeaders(w, cols)

	// header(1) + secondRow(1) + divider(1) + colHeaders(1) + statusbar(1) = 5 fixed rows
	// +1 when an update notice is shown
	fixedRows := 5
	if m.updateNotice != "" {
		fixedRows = 6
	}
	listHeight := m.height - fixedRows
	if listHeight < 1 {
		listHeight = 1
	}

	rows := m.renderRows(listHeight, w, cols)
	status := m.renderStatus(w)

	parts := []string{title, secondRow, div, colHeaders, rows}
	if m.updateNotice != "" {
		notice := styleUpdateNotice.Width(w).Render(
			fmt.Sprintf("Update available: %s — run shellodex --update", m.updateNotice))
		parts = append(parts, notice)
	}
	parts = append(parts, status)
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// colWidths holds the calculated widths for the adaptive column layout.
type colWidths struct {
	name    int
	host    int // hostname only
	port    int // ":NNNNN" or "      "
	user    int // 0 = hidden
	last    int // 0 = hidden
	labels  int // 0 = hidden
}

const (
	colCursorW = 2  // "▶ " or "  "
	colInnerPad = 4  // left inset to align with header text padding
	colBadgeW  = 3  // "SSH" or "TEL"
	colPortW   = 7  // ":22222 " (port + space)
	colHostW   = 18
	colUserW   = 12
	colLastW   = 8
	colGapW    = 2
)

func (m LauncherModel) colLayout(w int) colWidths {
	// Available width after fixed-cost columns (always visible).
	// colInnerPad is part of the row prefix, not a separate column.
	fixed := colInnerPad + colCursorW + colHostW + colPortW + colGapW + colBadgeW
	avail := w - fixed

	var cw colWidths
	cw.host = colHostW
	cw.port = colPortW

	// Progressively add optional columns as width allows.
	if avail >= colUserW+colGapW+10 {
		cw.user = colUserW
		avail -= colUserW + colGapW
	}
	if cw.user > 0 && avail >= colLastW+colGapW+10 {
		cw.last = colLastW
		avail -= colLastW + colGapW
	}
	if cw.last > 0 && avail >= 14+colGapW+10 {
		// Labels get remaining space (min 14 for a short group name + one tag).
		cw.labels = avail - colGapW - 10 // leave 10 for name
		avail = 10
	}

	// Name gets whatever is left.
	cw.name = avail
	if cw.name < 8 {
		cw.name = 8
	}
	return cw
}

func (m LauncherModel) renderColHeaders(w int, cw colWidths) string {
	// Build the header text aligned to the same column positions as the rows.
	var b strings.Builder
	// Match row prefix: innerPad + cursor placeholder
	b.WriteString(strings.Repeat(" ", colInnerPad+colCursorW))
	b.WriteString(plainPad("NAME", cw.name))
	b.WriteString(strings.Repeat(" ", colGapW))
	b.WriteString(plainPad("HOSTNAME", cw.host+cw.port))
	if cw.user > 0 {
		b.WriteString(strings.Repeat(" ", colGapW))
		b.WriteString(plainPad("USER", cw.user))
	}
	if cw.last > 0 {
		b.WriteString(strings.Repeat(" ", colGapW))
		b.WriteString(plainPad("LAST CONN", cw.last))
	}
	if cw.labels > 0 {
		b.WriteString(strings.Repeat(" ", colGapW))
		b.WriteString(plainPad("LABELS", cw.labels))
	}
	b.WriteString(strings.Repeat(" ", colGapW))
	b.WriteString("   ") // badge placeholder
	return styleColHeader.Width(w).Render(b.String())
}

func (m LauncherModel) renderRows(maxRows, w int, cw colWidths) string {
	if len(m.filtered) == 0 {
		return styleMuted.Padding(1, 4).Render("No hosts match.")
	}

	// Scroll window: keep cursor visible
	start := 0
	if m.cursor >= maxRows {
		start = m.cursor - maxRows + 1
	}
	end := start + maxRows
	if end > len(m.filtered) {
		end = len(m.filtered)
	}

	var sb strings.Builder
	for i := start; i < end; i++ {
		e := m.filtered[i]
		selected := i == m.cursor

		prefix := strings.Repeat(" ", colInnerPad)
		cursorStr := "  "
		if selected {
			cursorStr = "▶ "
		}

		name := plainPad(e.host.Name, cw.name)

		// Hostname + port combined into one block.
		port := e.host.Port
		if port == 0 {
			port = int(model.DefaultPort(e.host.Protocol))
		}
		hn := plainPad(e.host.Hostname, cw.host)
		portStr := plainPad(fmt.Sprintf(":%d", port), cw.port)

		var badgeStr string
		if e.host.Protocol == model.Telnet {
			badgeStr = styleTelnetBadge.Render("TEL")
		} else {
			badgeStr = styleSSHBadge.Render("SSH")
		}

		if selected {
			// Selected row: fill the full width with the highlight background, then
			// append the protocol badge outside the highlight so it keeps its color.
			plain := prefix + cursorStr + name + strings.Repeat(" ", colGapW) + hn + portStr
			if cw.user > 0 {
				plain += strings.Repeat(" ", colGapW) + plainPad(e.username, cw.user)
			}
			if cw.last > 0 {
				plain += strings.Repeat(" ", colGapW) + plainPad(relativeTime(e.host.LastConnected), cw.last)
			}
			if cw.labels > 0 {
				plain += strings.Repeat(" ", colGapW) + plainPad(plainLabels(e.groupPath, e.host.Tags), cw.labels)
			}
			// Pad to (w - badge width) so the highlight bar fills the terminal row.
			plain = plainPad(plain, w-colBadgeW)
			sb.WriteString(styleSelected.Render(plain) + badgeStr)
		} else {
			// Unselected: colored text only, no background — terminal bg shows through.
			row := prefix +
				"  " + // cursor placeholder, no arrow on unselected rows
				styleNormal.Render(name) +
				"  " + // colGapW
				styleNormal.Render(hn) +
				stylePortCol.Render(portStr)
			if cw.user > 0 {
				row += "  " + styleUserCol.Render(plainPad(e.username, cw.user))
			}
			if cw.last > 0 {
				row += "  " + styleLastConn.Render(plainPad(relativeTime(e.host.LastConnected), cw.last))
			}
			if cw.labels > 0 {
				row += "  " + renderLabels(e.groupPath, e.host.Tags, cw.labels)
			}
			row += "  " + badgeStr
			sb.WriteString(row)
		}

		if i < end-1 {
			sb.WriteRune('\n')
		}
	}

	// Stable height: pad unused rows with blank lines.
	rendered := end - start
	for i := rendered; i < maxRows; i++ {
		sb.WriteRune('\n')
	}

	return sb.String()
}

// relativeTime formats a last-connected timestamp as a compact relative string.
func relativeTime(t *time.Time) string {
	if t == nil {
		return "never"
	}
	d := time.Since(*t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	default:
		return fmt.Sprintf("%dmo", int(d.Hours()/(24*30)))
	}
}

// renderLabels builds a styled label string (group badge + tag pills) truncated to maxW.
func renderLabels(groupPath string, tags []string, maxW int) string {
	var parts []string
	if groupPath != "" {
		// Show only the leaf group name to keep it compact.
		leaf := groupPath
		if idx := strings.LastIndex(groupPath, " / "); idx >= 0 {
			leaf = groupPath[idx+3:]
		}
		parts = append(parts, styleGroupBadge.Render(leaf))
	}
	for _, tag := range tags {
		if tag != "" {
			parts = append(parts, styleTagBadge.Render(tag))
		}
	}
	if len(parts) == 0 {
		return strings.Repeat(" ", maxW)
	}
	joined := strings.Join(parts, " ")
	plain := plainLabels(groupPath, tags)
	if len([]rune(plain)) > maxW {
		return joined // let the terminal clip; styled badges look better untruncated
	}
	// Pad to maxW using plain-text width.
	pad := maxW - len([]rune(plain))
	if pad > 0 {
		return joined + strings.Repeat(" ", pad)
	}
	return joined
}

// plainLabels returns the unstyled text of labels (for width calculations and search).
func plainLabels(groupPath string, tags []string) string {
	var parts []string
	if groupPath != "" {
		leaf := groupPath
		if idx := strings.LastIndex(groupPath, " / "); idx >= 0 {
			leaf = groupPath[idx+3:]
		}
		parts = append(parts, leaf)
	}
	for _, tag := range tags {
		if tag != "" {
			parts = append(parts, tag)
		}
	}
	return strings.Join(parts, " ")
}

func (m LauncherModel) renderStatus(w int) string {
	var hint string
	if m.searching {
		hint = fmt.Sprintf("%s connect   %s back to browse   type to filter",
			styleStatusKey.Render("enter"),
			styleStatusKey.Render("esc"),
		)
	} else {
		hint = fmt.Sprintf("%s connect  %s search  %s new  %s edit  %s delete  %s creds  %s groups  %s settings  %s tree  %s quit",
			styleStatusKey.Render("enter"),
			styleStatusKey.Render("/"),
			styleStatusKey.Render("a"),
			styleStatusKey.Render("e"),
			styleStatusKey.Render("d"),
			styleStatusKey.Render("c"),
			styleStatusKey.Render("g"),
			styleStatusKey.Render("s"),
			styleStatusKey.Render("tab"),
			styleStatusKey.Render("q"),
		)
	}
	if m.statusMsg != "" {
		hint = styleSuccess.Render(m.statusMsg) + "   " + hint
	}
	return styleStatusBar.Width(w).Render(hint)
}

func (m *LauncherModel) rebuildAll() {
	m.all = make([]hostEntry, 0, len(m.cfg.Hosts))
	for _, h := range m.cfg.Hosts {
		var uname string
		if cred := m.cfg.EffectiveCredential(&h); cred != nil {
			uname = cred.Username
		}
		m.all = append(m.all, hostEntry{
			host:      h,
			groupPath: m.cfg.GroupPath(h.GroupID),
			username:  uname,
		})
	}
}

func (m *LauncherModel) refilter() {
	query := strings.ToLower(m.input.Value())
	if query == "" {
		m.filtered = make([]hostEntry, len(m.all))
		copy(m.filtered, m.all)
	} else {
		m.filtered = m.filtered[:0]
		for _, e := range m.all {
			if matchesQuery(query, e) {
				m.filtered = append(m.filtered, e)
			}
		}
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
}

func matchesQuery(query string, e hostEntry) bool {
	if strings.Contains(strings.ToLower(e.host.Name), query) ||
		strings.Contains(strings.ToLower(e.host.Hostname), query) ||
		strings.Contains(strings.ToLower(e.host.Notes), query) ||
		strings.Contains(strings.ToLower(e.groupPath), query) {
		return true
	}
	for _, tag := range e.host.Tags {
		if strings.Contains(strings.ToLower(tag), query) {
			return true
		}
	}
	return false
}

// plainPad truncates or right-pads s to exactly w terminal cells using plain ASCII.
func plainPad(s string, w int) string {
	runes := []rune(s)
	if len(runes) > w {
		if w > 1 {
			return string(runes[:w-1]) + "…"
		}
		return string(runes[:w])
	}
	return s + strings.Repeat(" ", w-len(runes))
}
