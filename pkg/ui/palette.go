package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// PaletteItem represents an actionable entry in the Spotlight Quick Navigation.
type PaletteItem struct {
	Type        string // "channel", "user", "action"
	Title       string
	Description string
	Value       string
}

// PaletteModel manages the Command Palette overlay (Ctrl+K).
type PaletteModel struct {
	input      textinput.Model
	items      []PaletteItem
	filtered   []PaletteItem
	cursor     int
	active     bool
	maxVisible int
}

// NewPaletteModel initializes the clean spotlight palette.
func NewPaletteModel() PaletteModel {
	ti := textinput.New()
	ti.Placeholder = "Type channel, callsign, or command..."
	ti.Prompt = "❯ "
	ti.PromptStyle = lipgloss.NewStyle().Foreground(current.Primary).Bold(true)
	ti.TextStyle = lipgloss.NewStyle().Foreground(current.Text)
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(current.Muted)
	ti.CharLimit = 64
	ti.Width = 60

	return PaletteModel{
		input:      ti,
		maxVisible: 8,
	}
}

// Open activates the command palette and populates candidates.
func (p *PaletteModel) Open(rooms []string, users []string) {
	p.active = true
	p.input.Reset()
	p.input.Focus()
	p.cursor = 0

	var items []PaletteItem

	// Channels
	for _, r := range rooms {
		name := r
		if !strings.HasPrefix(name, "#") {
			name = "#" + name
		}
		items = append(items, PaletteItem{
			Type:        "channel",
			Title:       name,
			Description: "Jump to frequency",
			Value:       name,
		})
	}

	// Users / Souls
	for _, u := range users {
		nick := strings.TrimPrefix(u, "@")
		items = append(items, PaletteItem{
			Type:        "user",
			Title:       "@" + nick,
			Description: "Direct transmission",
			Value:       nick,
		})
	}

	// Actions & Commands (Clean, zero emojis)
	actions := []struct{ title, desc, val string }{
		{"/theme obsidian", "Obsidian black & cognac theme", "/theme obsidian"},
		{"/theme cognac", "Warm cognac & rust theme", "/theme cognac"},
		{"/theme monochrome", "Stark black & white theme", "/theme monochrome"},
		{"/theme crimson-night", "Deep night & crimson theme", "/theme crimson-night"},
		{"/server", "Open Server Switcher drawer", "/server"},
		{"/leave", "Leave channel and return to #general", "/leave"},
		{"/clear", "Clear canvas transmission history", "/clear"},
		{"/bell", "Toggle audio mention chime", "/bell"},
		{"/whoami", "Inspect anonymous identity telemetry", "/whoami"},
		{"/quit", "Power down transceiver", "/quit"},
	}
	for _, a := range actions {
		items = append(items, PaletteItem{
			Type:        "action",
			Title:       a.title,
			Description: a.desc,
			Value:       a.val,
		})
	}

	p.items = items
	p.filter()
}

// Close deactivates the palette.
func (p *PaletteModel) Close() {
	p.active = false
	p.input.Blur()
}

// IsActive returns whether palette is visible.
func (p *PaletteModel) IsActive() bool {
	return p.active
}

// Filter matches search text against items.
func (p *PaletteModel) filter() {
	query := strings.ToLower(strings.TrimSpace(p.input.Value()))
	if query == "" {
		p.filtered = p.items
		p.cursor = 0
		return
	}

	var matched []PaletteItem
	for _, it := range p.items {
		target := strings.ToLower(it.Title + " " + it.Description)
		if strings.Contains(target, query) {
			matched = append(matched, it)
		}
	}
	p.filtered = matched
	if p.cursor >= len(p.filtered) {
		p.cursor = 0
	}
}

// MoveUp moves selection cursor up.
func (p *PaletteModel) MoveUp() {
	if len(p.filtered) == 0 {
		return
	}
	p.cursor--
	if p.cursor < 0 {
		p.cursor = len(p.filtered) - 1
	}
}

// MoveDown moves selection cursor down.
func (p *PaletteModel) MoveDown() {
	if len(p.filtered) == 0 {
		return
	}
	p.cursor++
	if p.cursor >= len(p.filtered) {
		p.cursor = 0
	}
}

// Selected returns the currently selected palette item, if any.
func (p *PaletteModel) Selected() *PaletteItem {
	if len(p.filtered) == 0 || p.cursor < 0 || p.cursor >= len(p.filtered) {
		return nil
	}
	return &p.filtered[p.cursor]
}

// View renders the clean, minimal, non-emoji Spotlight palette.
func (p *PaletteModel) View(screenWidth, screenHeight int) string {
	boxWidth := 72

	title := lipgloss.NewStyle().
		Foreground(current.Text).
		Bold(true).
		Render("◆ QUICK NAVIGATION // SPOTLIGHT (Ctrl+K)")

	searchPrompt := lipgloss.NewStyle().
		Background(current.DarkBg).
		Padding(0, 1).
		Width(boxWidth - 6).
		Render(p.input.View())

	divider := lipgloss.NewStyle().
		Foreground(current.Border).
		Render(strings.Repeat("─", boxWidth-6))

	var list strings.Builder

	if len(p.filtered) == 0 {
		list.WriteString("\n" + lipgloss.NewStyle().Foreground(current.Muted).Render("   No matching channels, callsigns, or commands") + "\n\n")
	} else {
		start := 0
		if p.cursor >= p.maxVisible {
			start = p.cursor - p.maxVisible + 1
		}
		end := start + p.maxVisible
		if end > len(p.filtered) {
			end = len(p.filtered)
		}

		for i := start; i < end; i++ {
			it := p.filtered[i]
			selected := (i == p.cursor)

			prefix := "   "
			if selected {
				prefix = lipgloss.NewStyle().Foreground(current.Primary).Bold(true).Render(" ▶ ")
			}

			// Clean category tag
			tag := ""
			switch it.Type {
			case "channel":
				tag = lipgloss.NewStyle().Foreground(current.Secondary).Render("[chan]")
			case "user":
				tag = lipgloss.NewStyle().Foreground(current.TextDim).Render("[soul]")
			case "action":
				tag = lipgloss.NewStyle().Foreground(current.Muted).Render("[cmd] ")
			}

			var titleStyle lipgloss.Style
			var descStyle lipgloss.Style

			if selected {
				titleStyle = lipgloss.NewStyle().Foreground(current.Text).Bold(true)
				descStyle = lipgloss.NewStyle().Foreground(current.TextDim)
			} else {
				titleStyle = lipgloss.NewStyle().Foreground(current.TextDim)
				descStyle = lipgloss.NewStyle().Foreground(current.Muted)
			}

			formattedTitle := titleStyle.Render(fmt.Sprintf("%-22s", it.Title))
			formattedDesc := descStyle.Render(it.Description)

			line := fmt.Sprintf("%s%s %s %s", prefix, tag, formattedTitle, formattedDesc)
			if selected {
				line = lipgloss.NewStyle().
					Background(current.DarkBg).
					Width(boxWidth - 6).
					Render(line)
			}
			list.WriteString(line + "\n")
		}
	}

	footer := lipgloss.NewStyle().
		Foreground(current.Muted).
		Render("\n  Navigate: [↑/↓]  ·  Execute: [Enter]  ·  Dismiss: [Esc]")

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"\n",
		searchPrompt,
		divider,
		list.String(),
		footer,
	)

	modalBox := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(current.BorderActive).
		Background(current.PanelBg).
		Padding(1, 2).
		Width(boxWidth).
		Render(content)

	return lipgloss.Place(
		screenWidth,
		screenHeight,
		lipgloss.Center,
		lipgloss.Center,
		modalBox,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(current.DarkBg),
	)
}
