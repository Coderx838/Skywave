package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// AppStage defines the active application screen.
type AppStage int

const (
	StageBoot AppStage = iota
	StageNamePrompt
	StageWelcome
	StageChat
)

// BootDiagnostics lists step messages during the animated boot.
var BootDiagnostics = []string{
	"Initializing ephemeral memory ring buffer...",
	"Deriving local Ed25519 anonymous ghost identity...",
	"Calibrating AES-256-GCM client encryption engine...",
	"Handshaking carrier node on port 8080...",
	"Frequency lock acquired. Transceiver ready.",
}

const asciiArt = ` ▄██████  █   █  █   █  █     █  ▄█████  █   █  ██████
░██       █ ▄█▀   ▀█▄█▀  █ ▄ █ █ ░██  ██ █   █ ░██    
░██████   ███▀     ███   ███████ ░██████ █   █ ░█████ 
     ██   █ ▀█▄    ███   ██ ▀ ██ ░██  ██ ░█ █░ ░██    
██████▀   █   █    ███   █     █ ░██  ██  ░█░  ░██████`

// RenderBootScreen renders the dramatic cinematic startup diagnostic.
func RenderBootScreen(width, height int, step int, progress float64) string {
	boxW := 66

	carrierTop := lipgloss.NewStyle().
		Foreground(current.Muted).
		Align(lipgloss.Center).
		Width(boxW).
		Render("◈ ─ [ C A R R I E R  I N I T I A L I Z A T I O N ] ─ ◈")

	logo := lipgloss.NewStyle().
		Foreground(current.Text).
		Bold(true).
		Align(lipgloss.Center).
		Width(boxW).
		Render(strings.TrimSpace(asciiArt))

	tagline := lipgloss.NewStyle().
		Foreground(current.Secondary).
		Bold(true).
		Align(lipgloss.Center).
		Width(boxW).
		Render("E N C R Y P T E D   T E R M I N A L   T R A N S C E I V E R")

	divider := lipgloss.NewStyle().
		Foreground(current.Border).
		Align(lipgloss.Center).
		Width(boxW).
		Render(strings.Repeat("─", boxW-4))

	barWidth := 38
	filled := int(progress * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}
	bar := lipgloss.NewStyle().Foreground(current.Primary).Render(strings.Repeat("█", filled)) +
		lipgloss.NewStyle().Foreground(current.Muted).Render(strings.Repeat("░", barWidth-filled))
	pct := fmt.Sprintf("%3d%%", int(progress*100))

	progLine := lipgloss.NewStyle().Align(lipgloss.Center).Width(boxW).Render(
		fmt.Sprintf("%s  %s", bar, lipgloss.NewStyle().Foreground(current.Text).Bold(true).Render(pct)),
	)

	var diagLines []string
	for i, msg := range BootDiagnostics {
		if i <= step {
			check := lipgloss.NewStyle().Foreground(current.Success).Render("●")
			diagLines = append(diagLines, fmt.Sprintf(" [%s] %s", check, lipgloss.NewStyle().Foreground(current.TextDim).Render(msg)))
		}
	}
	diagBox := lipgloss.NewStyle().
		Background(current.DarkBg).
		Border(lipgloss.NormalBorder()).
		BorderForeground(current.Border).
		Padding(0, 1).
		Width(boxW - 8).
		Render(strings.Join(diagLines, "\n"))

	diagContainer := lipgloss.NewStyle().Align(lipgloss.Center).Width(boxW).Render(diagBox)

	footer := lipgloss.NewStyle().
		Foreground(current.Muted).
		Align(lipgloss.Center).
		Width(boxW).
		Render("[Space / Enter]: Skip Diagnostic")

	content := lipgloss.JoinVertical(lipgloss.Center,
		carrierTop,
		"",
		logo,
		"",
		tagline,
		"",
		divider,
		"",
		diagContainer,
		"",
		progLine,
		"",
		footer,
	)

	frame := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(current.BorderActive).
		Background(current.PanelBg).
		Padding(1, 2).
		Render(content)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, frame)
}

// RenderNamePrompt renders the required callsign selection screen.
func RenderNamePrompt(width, height int, inputView string, errMsg string) string {
	boxW := 66

	carrierTop := lipgloss.NewStyle().
		Foreground(current.Muted).
		Align(lipgloss.Center).
		Width(boxW).
		Render("◈ ─ [ S I G N A L  C A R R I E R ] ─ ◈")

	logo := lipgloss.NewStyle().
		Foreground(current.Text).
		Bold(true).
		Align(lipgloss.Center).
		Width(boxW).
		Render(strings.TrimSpace(asciiArt))

	tagline := lipgloss.NewStyle().
		Foreground(current.Secondary).
		Bold(true).
		Align(lipgloss.Center).
		Width(boxW).
		Render("E N C R Y P T E D   T E R M I N A L   T R A N S C E I V E R")

	divider := lipgloss.NewStyle().
		Foreground(current.Border).
		Align(lipgloss.Center).
		Width(boxW).
		Render(strings.Repeat("─", boxW-4))

	title := lipgloss.NewStyle().
		Foreground(current.Text).
		Bold(true).
		Align(lipgloss.Center).
		Width(boxW).
		Render("IDENTIFY ON FREQUENCY")

	sub := lipgloss.NewStyle().
		Foreground(current.TextDim).
		Align(lipgloss.Center).
		Width(boxW).
		Render("Enter your callsign to unlock the carrier wave")

	inputField := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(current.BorderActive).
		Background(current.DarkBg).
		Foreground(current.Text).
		Padding(0, 1).
		Width(boxW - 8).
		Align(lipgloss.Left).
		Render(inputView)

	inputContainer := lipgloss.NewStyle().
		Align(lipgloss.Center).
		Width(boxW).
		Render(inputField)

	footer := lipgloss.NewStyle().
		Foreground(current.Muted).
		Align(lipgloss.Center).
		Width(boxW).
		Render("[Enter]: Lock Callsign & Enter  ·  [Ctrl+C]: Power Down")

	var items []string
	items = append(items, carrierTop, "", logo, "", tagline, "", divider, "", title, sub, "", inputContainer)

	if errMsg != "" {
		errLine := lipgloss.NewStyle().
			Foreground(current.Danger).
			Bold(true).
			Align(lipgloss.Center).
			Width(boxW).
			Render("▲ " + errMsg)
		items = append(items, "", errLine)
	}

	items = append(items, "", footer)

	content := lipgloss.JoinVertical(lipgloss.Center, items...)

	frame := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(current.BorderActive).
		Background(current.PanelBg).
		Padding(1, 2).
		Render(content)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, frame)
}

// RenderWelcomeDashboard renders the centered Gateway Screen before entering chat.
func RenderWelcomeDashboard(width, height int, nick, accountID, serverName, serverURL string, pingMs int) string {
	boxW := 66

	carrierTop := lipgloss.NewStyle().
		Foreground(current.Muted).
		Align(lipgloss.Center).
		Width(boxW).
		Render("◈ ─ [ C Y B E R D E C K   G A T E W A Y ] ─ ◈")

	logo := lipgloss.NewStyle().
		Foreground(current.Text).
		Bold(true).
		Align(lipgloss.Center).
		Width(boxW).
		Render(strings.TrimSpace(asciiArt))

	tagline := lipgloss.NewStyle().
		Foreground(current.Secondary).
		Bold(true).
		Align(lipgloss.Center).
		Width(boxW).
		Render("E N C R Y P T E D   T E R M I N A L   T R A N S C E I V E R")

	// Telemetry Box (width 58)
	tTitle := lipgloss.NewStyle().Foreground(current.Secondary).Bold(true).Render("┌── SYSTEM TELEMETRY ──────────────────────────────────────┐")
	tBot := lipgloss.NewStyle().Foreground(current.Border).Render("└──────────────────────────────────────────────────────────┘")
	tCall := lipgloss.NewStyle().Foreground(current.Text).Bold(true).Render("@" + nick)
	tPing := lipgloss.NewStyle().Foreground(current.Success).Bold(true).Render(fmt.Sprintf("● %dms ONLINE", pingMs))
	tID := lipgloss.NewStyle().Foreground(current.TextDim).Render(accountID)
	tCipher := lipgloss.NewStyle().Foreground(current.Text).Render("AES-256-GCM [LOCAL]")
	tNode := lipgloss.NewStyle().Foreground(current.Text).Render(serverName)
	tURL := lipgloss.NewStyle().Foreground(current.TextDim).Render(serverURL)

	row1 := fmt.Sprintf("│  CALLSIGN : %-21s STATUS : %-19s│", tCall, tPing)
	row2 := fmt.Sprintf("│  GHOST ID : %-21s CIPHER : %-19s│", tID, tCipher)
	row3 := fmt.Sprintf("│  NODE     : %-21s RELAY  : %-19s│", tNode, tURL)

	telemetryBox := lipgloss.NewStyle().Align(lipgloss.Center).Width(boxW).Render(
		lipgloss.JoinVertical(lipgloss.Left, tTitle, row1, row2, row3, tBot),
	)

	// Action Box (width 58)
	cTitle := lipgloss.NewStyle().Foreground(current.Muted).Bold(true).Render("┌── FREQUENCY NAVIGATION ──────────────────────────────────┐")
	cBot := lipgloss.NewStyle().Foreground(current.Border).Render("└──────────────────────────────────────────────────────────┘")
	act1 := "│  [Enter / Space]  Tune into Primary Frequency (#general) │"
	act2 := "│  [s]              Node Switcher / Server Relay Bookmarks │"
	act3 := "│  [n]              Calibrate Callsign Handle              │"
	act4 := fmt.Sprintf("│  [Ctrl+T]         Cycle Theme Palette (%-17s)│", strings.ToUpper(ThemeName()))
	act5 := "│  [q / Esc]        Power Down Cyberdeck                   │"

	actionsBox := lipgloss.NewStyle().Align(lipgloss.Center).Width(boxW).Render(
		lipgloss.JoinVertical(lipgloss.Left, cTitle, act1, act2, act3, act4, act5, cBot),
	)

	content := lipgloss.JoinVertical(lipgloss.Center,
		carrierTop,
		"",
		logo,
		"",
		tagline,
		"",
		telemetryBox,
		"",
		actionsBox,
	)

	frame := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(current.BorderActive).
		Background(current.PanelBg).
		Padding(1, 2).
		Render(content)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, frame)
}
