package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func Banner() string {
	logo := "" +
		"    _    ____ ____   _    ____ ____  \n" +
		"   / \\  / ___|  _ \\ / \\  / ___/ ___| \n" +
		"  / _ \\| |  _| |_) / _ \\ \\___ \\___ \\ \n" +
		" / ___ \\ |_| |  __/ ___ \\ ___) |__) |\n" +
		"/_/   \\_\\____|_| /_/   \\_\\____/____/ \n"

	logoStyle := lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true)

	tagline := lipgloss.NewStyle().
		Foreground(ColorMuted).
		Italic(true).
		Render("  AI Agent Account Switcher")

	version := lipgloss.NewStyle().
		Foreground(ColorDim).
		Render("  v0.1.0")

	separator := lipgloss.NewStyle().
		Foreground(ColorDim).
		Render(strings.Repeat("-", 40))

	return fmt.Sprintf("\n%s\n%s %s\n%s\n", logoStyle.Render(logo), tagline, version, separator)
}

func SectionHeader(icon, title string) string {
	return fmt.Sprintf("%s  %s", icon, Title.Render(title))
}

func SuccessMessage(msg string) string {
	check := Success.Render(IconCheck)
	return fmt.Sprintf("\n  %s %s\n", check, Success.Render(msg))
}

func ErrorMessage(msg string) string {
	cross := Danger.Render(IconCross)
	return fmt.Sprintf("\n  %s %s\n", cross, Danger.Render(msg))
}

func WarningMessage(msg string) string {
	warn := Warning.Render(IconWarn)
	return fmt.Sprintf("\n  %s %s\n", warn, Warning.Render(msg))
}
