package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Catppuccin Mocha / Modern Dark Palette
var (
	ColorPrimary   = lipgloss.Color("#89b4fa") // Blue
	ColorSecondary = lipgloss.Color("#cba6f7") // Mauve
	ColorSuccess   = lipgloss.Color("#a6e3a1") // Green
	ColorWarning   = lipgloss.Color("#f9e2af") // Yellow
	ColorDanger    = lipgloss.Color("#f38ba8") // Red
	ColorMuted     = lipgloss.Color("#6c7086") // Overlay0
	ColorEmptyBar  = lipgloss.Color("#313244") // Surface0
	ColorBg        = lipgloss.Color("#1e1e2e") // Base
	ColorBright    = lipgloss.Color("#cdd6f4") // Text
	ColorAccent    = lipgloss.Color("#fab387") // Peach
	ColorDim       = lipgloss.Color("#585b70") // Surface2
)

var (
	BoxBorder = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorMuted).
		Padding(1, 2)

	Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorPrimary)

	Subtitle = lipgloss.NewStyle().
		Foreground(ColorSecondary)

	Success = lipgloss.NewStyle().
		Foreground(ColorSuccess).
		Bold(true)

	Warning = lipgloss.NewStyle().
		Foreground(ColorWarning).
		Bold(true)

	Danger = lipgloss.NewStyle().
		Foreground(ColorDanger).
		Bold(true)

	Muted = lipgloss.NewStyle().
		Foreground(ColorMuted)

	Accent = lipgloss.NewStyle().
		Foreground(ColorAccent)

	Bright = lipgloss.NewStyle().
		Foreground(ColorBright).
		Bold(true)

	AgentName = lipgloss.NewStyle().
		Foreground(ColorSecondary).
		Bold(true)

	AccountActive = lipgloss.NewStyle().
		Foreground(ColorSuccess).
		Bold(true)

	AccountInactive = lipgloss.NewStyle().
		Foreground(ColorMuted)

	Label = lipgloss.NewStyle().
		Foreground(ColorPrimary)
)

// Tree characters
const (
	TreeBranch = "├─"
	TreeLast   = "└─"
	TreePipe   = "│ "
	TreeSpace  = "   "
)

// Icons
const (
	IconCheck   = "✓"
	IconCross   = "✕"
	IconArrow   = "→"
	IconDot     = "●"
	IconDotOpen = "○"
	IconQuota   = "📊"
	IconSwitch  = "🔄"
	IconAgent   = "🤖"
	IconKey     = "🔑"
	IconWarn    = "⚠"
)

// ProgressBar renders a clean progress bar
func ProgressBar(percent float64, width int) string {
	filled := int(percent / 100 * float64(width))
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}
	empty := width - filled

	var color lipgloss.Color
	switch {
	case percent >= 70:
		color = ColorSuccess
	case percent >= 30:
		color = ColorWarning
	default:
		color = ColorDanger
	}

	filledStyle := lipgloss.NewStyle().Foreground(color)
	emptyStyle := lipgloss.NewStyle().Foreground(ColorEmptyBar)

	return filledStyle.Render(strings.Repeat("█", filled)) + emptyStyle.Render(strings.Repeat("█", empty))
}