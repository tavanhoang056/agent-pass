package ui

import (
	"github.com/charmbracelet/lipgloss"
)

// Color palette — oh-my-pi inspired dark theme
var (
	ColorPrimary   = lipgloss.Color("#7C3AED") // Purple
	ColorSecondary = lipgloss.Color("#06B6D4") // Cyan
	ColorSuccess   = lipgloss.Color("#22C55E") // Green
	ColorWarning   = lipgloss.Color("#EAB308") // Yellow
	ColorDanger    = lipgloss.Color("#EF4444") // Red
	ColorMuted     = lipgloss.Color("#6B7280") // Gray
	ColorText      = lipgloss.Color("#E5E7EB") // Light gray
	ColorBright    = lipgloss.Color("#F9FAFB") // White
	ColorDim       = lipgloss.Color("#374151") // Dark gray
	ColorAccent    = lipgloss.Color("#A78BFA") // Light purple
	ColorOrange    = lipgloss.Color("#F97316") // Orange
)

// Box styles
var (
	BoxBorder = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(1, 2)

	BoxBorderSuccess = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorSuccess).
		Padding(1, 2)

	BoxBorderDanger = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorDanger).
		Padding(1, 2)
)

// Text styles
var (
	Title = lipgloss.NewStyle().
		Foreground(ColorBright).
		Bold(true).
		MarginBottom(1)

	Subtitle = lipgloss.NewStyle().
		Foreground(ColorAccent).
		Bold(true)

	Label = lipgloss.NewStyle().
		Foreground(ColorMuted)

	Value = lipgloss.NewStyle().
		Foreground(ColorText)

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
)

// Tree characters
const (
	TreeBranch = "├─"
	TreeLast   = "└─"
	TreePipe   = "│ "
	TreeSpace  = "  "
)

// Icons
const (
	IconCheck   = "✓"
	IconCross   = "✗"
	IconArrow   = "→"
	IconDot     = "●"
	IconDotOpen = "○"
	IconPass    = "🎟️"
	IconSwitch  = "🔄"
	IconQuota   = "📊"
	IconAgent   = "🤖"
	IconKey     = "🔑"
	IconWarn    = "⚠"
)

// ProgressBar renders a colored progress bar
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
	emptyStyle := lipgloss.NewStyle().Foreground(ColorDim)

	bar := ""
	for i := 0; i < filled; i++ {
		bar += filledStyle.Render("█")
	}
	for i := 0; i < empty; i++ {
		bar += emptyStyle.Render("░")
	}

	return bar
}
