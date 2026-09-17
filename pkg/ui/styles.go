package ui

import (
	"hash/fnv"

	"github.com/charmbracelet/lipgloss"
)

// Theme defines the refined industrial obsidian & cognac aesthetic palette.
type Theme struct {
	Name         string
	Primary      lipgloss.Color // Crimson Jewel Accent
	Secondary    lipgloss.Color // Warm Cognac / Leather Brown
	Accent       lipgloss.Color // Rust Amber
	Success      lipgloss.Color // Silver / Chalk White
	Warning      lipgloss.Color // Warm Cognac
	Danger       lipgloss.Color // Intense Crimson
	Muted        lipgloss.Color // Charcoal / Slate
	DarkBg       lipgloss.Color // Deep Matte Obsidian (#0C0C0C)
	PanelBg      lipgloss.Color // Dark Graphite (#141414)
	Border       lipgloss.Color // Hairline Border (#262626)
	BorderActive lipgloss.Color // Focused Border (#A75D33 / #E5383B)
	Text         lipgloss.Color // Stark Chalk White (#F0F0F0)
	TextDim      lipgloss.Color // Medium Silver Grey (#888888)
}

// Refined aesthetic themes
var Themes = map[string]Theme{
	"obsidian": {
		Name: "obsidian", Primary: "#E5383B", Secondary: "#9C6644",
		Accent: "#A75D33", Success: "#D4D4D4", Warning: "#C85A32",
		Danger: "#BA181B", Muted: "#484848", DarkBg: "#0C0C0C",
		PanelBg: "#141414", Border: "#262626", BorderActive: "#9C6644",
		Text: "#F0F0F0", TextDim: "#888888",
	},
	"cognac": {
		Name: "cognac", Primary: "#C85A32", Secondary: "#7F5539",
		Accent: "#B08968", Success: "#EDE0D4", Warning: "#8C482C",
		Danger: "#D90429", Muted: "#3D3835", DarkBg: "#0B0908",
		PanelBg: "#171412", Border: "#282320", BorderActive: "#C85A32",
		Text: "#F5EBE0", TextDim: "#8A8077",
	},
	"monochrome": {
		Name: "monochrome", Primary: "#FFFFFF", Secondary: "#A0A0A0",
		Accent: "#E5383B", Success: "#D0D0D0", Warning: "#666666",
		Danger: "#E5383B", Muted: "#383838", DarkBg: "#050505",
		PanelBg: "#101010", Border: "#222222", BorderActive: "#FFFFFF",
		Text: "#FFFFFF", TextDim: "#777777",
	},
	"crimson-night": {
		Name: "crimson-night", Primary: "#E5383B", Secondary: "#590D22",
		Accent: "#FF4D6D", Success: "#E0E1DD", Warning: "#800F2F",
		Danger: "#FF0033", Muted: "#3A3A3A", DarkBg: "#08080A",
		PanelBg: "#111114", Border: "#222228", BorderActive: "#E5383B",
		Text: "#F8F9FA", TextDim: "#8D99AE",
	},
}

// Active theme default: Obsidian
var current = Themes["obsidian"]

var themeCycleOrder = []string{"obsidian", "cognac", "monochrome", "crimson-night"}

// SetTheme switches the active theme.
func SetTheme(name string) bool {
	t, ok := Themes[name]
	if ok {
		current = t
	}
	return ok
}

// CycleTheme cycles next theme.
func CycleTheme() string {
	idx := 0
	for i, name := range themeCycleOrder {
		if name == current.Name {
			idx = (i + 1) % len(themeCycleOrder)
			break
		}
	}
	next := themeCycleOrder[idx]
	SetTheme(next)
	return next
}

// ThemeName returns active theme name.
func ThemeName() string { return current.Name }

// ThemeNames lists all theme identifiers.
func ThemeNames() []string { return themeCycleOrder }

// Tasteful callsign palette: Silver, Cognac, Sand, Steel, Muted Crimson
var authorPalette = []lipgloss.Color{
	"#D4D4D4", // Pure Silver
	"#9C6644", // Warm Cognac
	"#C85A32", // Rust Amber
	"#8A9EA7", // Muted Slate Blue
	"#B08968", // Sand Bronze
	"#A0A0A0", // Cool Steel
	"#E5383B", // Crimson Dot
	"#E0DCD3", // Chalk Bone
}

// AuthorColor deterministically picks a callsign color.
func AuthorColor(nick string) lipgloss.Color {
	h := fnv.New32a()
	_, _ = h.Write([]byte(nick))
	return authorPalette[int(h.Sum32())%len(authorPalette)]
}

// ---- Clean Transceiver Typography (No Garish Red Boxes!) ----

// Minimal logo: Stark white text with a crimson dot indicator
func sBrand() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(current.Text).
		Bold(true)
}

func sHeaderBar() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(current.DarkBg).
		Foreground(current.Text).
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(current.Border).
		Padding(0, 1)
}

// Horizontal Channel Tabs — Subtle and Clean
func sTabActive(name string, count int, isPrivate bool) string {
	lock := ""
	if isPrivate {
		lock = " " + lipgloss.NewStyle().Foreground(current.Accent).Bold(true).Render("◈")
	}
	dot := lipgloss.NewStyle().Foreground(current.Primary).Render("● ")
	label := lipgloss.NewStyle().Foreground(current.Text).Bold(true).Render(name)
	return " " + dot + label + lock + " "
}

func sTabInactive(idx int, name string, count int, isPrivate bool) string {
	lock := ""
	if isPrivate {
		lock = " " + lipgloss.NewStyle().Foreground(current.Muted).Render("◈")
	}
	num := lipgloss.NewStyle().Foreground(current.Muted).Render(lipgloss.NewStyle().SetString(string(rune('0'+idx))).String() + ":")
	label := lipgloss.NewStyle().Foreground(current.TextDim).Render(name)
	return " " + num + label + lock + " "
}

func sTabAdd() string {
	return lipgloss.NewStyle().
		Foreground(current.Secondary).
		Render(" [+] ")
}

// Chat canvas frame
func sCanvas() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(current.DarkBg).
		Padding(0, 1)
}

// Transmitter Input Deck — Subtle dark border, never screaming red
func sDeck() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(current.PanelBg).
		Border(lipgloss.NormalBorder(), true, false, false, false).
		BorderForeground(current.Border).
		Padding(0, 1)
}

func sDeckPrompt(room string) lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(current.Secondary).
		Bold(true)
}

// Statusline telemetry
func sStatusline() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(current.DarkBg).
		Foreground(current.Muted).
		Padding(0, 1)
}

func sStatus() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(current.Muted)
}

// Modals
func sModal() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(current.BorderActive).
		Background(current.PanelBg).
		Padding(1, 2)
}

func sModalTitle(title string) string {
	return lipgloss.NewStyle().
		Foreground(current.Text).
		Bold(true).
		MarginBottom(1).
		Render("◆ " + title)
}

func sSpotlightItem(selected bool) lipgloss.Style {
	if selected {
		return lipgloss.NewStyle().
			Background(current.PanelBg).
			Foreground(current.Primary).
			Bold(true)
	}
	return lipgloss.NewStyle().
		Foreground(current.TextDim)
}
