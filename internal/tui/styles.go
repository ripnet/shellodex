package tui

import "github.com/charmbracelet/lipgloss"

// Catppuccin Mocha palette
const (
	colorBase    = "#1e1e2e"
	colorSurface = "#313244"
	colorOverlay = "#45475a"
	colorText    = "#cdd6f4"
	colorSubtext = "#a6adc8"
	colorMuted   = "#585b70"
	colorBlue    = "#89b4fa"
	colorGreen   = "#a6e3a1"
	colorYellow  = "#f9e2af"
	colorRed     = "#f38ba8"
	colorMauve   = "#cba6f7"
	colorTeal    = "#94e2d5"
	colorPeach   = "#fab387"
)

var (
	styleBase = lipgloss.NewStyle().
			Background(lipgloss.Color(colorBase)).
			Foreground(lipgloss.Color(colorText))

	// Header: just styled text — no background band.
	styleHeader = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorMauve)).
			Bold(true).
			Padding(0, 4)

	// Search bar: plain text row, no background.
	styleSearchBar = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorText)).
			Padding(0, 1)

	styleSearchPrompt = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorBlue)).
				Bold(true)

	// Selected row is the ONLY element that uses a background highlight.
	styleSelected = lipgloss.NewStyle().
			Background(lipgloss.Color(colorOverlay)).
			Foreground(lipgloss.Color(colorText)).
			Bold(true)

	// Normal rows: text color only, terminal background shows through.
	styleNormal = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorText))

	styleGroupPath = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorSubtext))

	styleCursor = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorMauve)).
			Bold(true)

	styleMuted = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorMuted))

	// Status bar: dimmed text, no background.
	styleStatusBar = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorSubtext)).
			Padding(0, 4)

	styleStatusKey = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorBlue)).
			Bold(true)

	// Protocol badges: colored text only, no background pill.
	styleSSHBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorBlue)).
			Bold(true)

	styleTelnetBadge = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorYellow)).
				Bold(true)

	// Column headers: very dim, not competing with data.
	styleColHeader = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorMuted))

	styleGroupBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorTeal)).
			Bold(true)

	styleTagBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorPeach))

	styleUserCol = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorSubtext))

	stylePortCol = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorMuted))

	styleLastConn = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorMuted))

	styleBorder = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(colorOverlay))

	styleOverlayTitle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorMauve)).
				Bold(true).
				Padding(0, 1)

	styleDivider = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorOverlay))

	styleError = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorRed))

	styleSuccess = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorGreen))

	styleDetailLabel = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorSubtext)).
				Width(14)

	styleDetailValue = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorText))

	styleUpdateNotice = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorYellow)).
				Padding(0, 4)
)
