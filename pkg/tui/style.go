package tui

import (
	_ "embed"
	"github.com/charmbracelet/lipgloss"
)

const LogoColor = "#816FA6"

var (
	//go:embed logo.txt
	logo      string
	logoStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(LogoColor))
)

func RenderLogo() string {
	return logoStyle.Render(logo)
}
