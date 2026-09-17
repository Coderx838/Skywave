package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/skywave-chat/skywave/pkg/identity"
)

// ServerModal manages the Multi-Server Switcher drawer (Ctrl+S).
type ServerModal struct {
	serverList  *identity.ServerList
	cursor      int
	active      bool
	addingNew   bool
	nameInput   textinput.Model
	urlInput    textinput.Model
	activeField int // 0 = name, 1 = url
}

// NewServerModal initializes the server switcher modal.
func NewServerModal() ServerModal {
	sl, _ := identity.LoadServers()
	if sl == nil {
		d := identity.DefaultServers()
		sl = &d
	}

	nameIn := textinput.New()
	nameIn.Placeholder = "Node Name (e.g. My VPS Server)"
	nameIn.Prompt = "Name: "
	nameIn.CharLimit = 32

	urlIn := textinput.New()
	urlIn.Placeholder = "ws://192.168.1.100:8080/ws"
	urlIn.Prompt = "URL:  "
	urlIn.CharLimit = 128

	return ServerModal{
		serverList: sl,
		nameInput:  nameIn,
		urlInput:   urlIn,
	}
}

// Open activates the server switcher modal.
func (sm *ServerModal) Open(currentURL string) {
	sm.active = true
	sm.addingNew = false
	// Reload from disk to keep synchronized
	if fresh, err := identity.LoadServers(); err == nil && fresh != nil {
		sm.serverList = fresh
	}

	sm.cursor = 0
	for i, s := range sm.serverList.Servers {
		if s.URL == currentURL {
			sm.cursor = i
			break
		}
	}
}

// Close dismisses the modal.
func (sm *ServerModal) Close() {
	sm.active = false
	sm.addingNew = false
}

// IsActive returns whether modal is currently open.
func (sm *ServerModal) IsActive() bool {
	return sm.active
}

// MoveUp navigates server list up.
func (sm *ServerModal) MoveUp() {
	if sm.addingNew || len(sm.serverList.Servers) == 0 {
		return
	}
	sm.cursor--
	if sm.cursor < 0 {
		sm.cursor = len(sm.serverList.Servers) - 1
	}
}

// MoveDown navigates server list down.
func (sm *ServerModal) MoveDown() {
	if sm.addingNew || len(sm.serverList.Servers) == 0 {
		return
	}
	sm.cursor++
	if sm.cursor >= len(sm.serverList.Servers) {
		sm.cursor = 0
	}
}

// StartAdd switches modal into "Add Server" form.
func (sm *ServerModal) StartAdd() {
	sm.addingNew = true
	sm.activeField = 0
	sm.nameInput.Reset()
	sm.nameInput.Focus()
	sm.urlInput.Reset()
	sm.urlInput.Blur()
}

// CancelAdd returns to the server list view.
func (sm *ServerModal) CancelAdd() {
	sm.addingNew = false
}

// SubmitAdd saves the new server.
func (sm *ServerModal) SubmitAdd() *identity.ServerEntry {
	name := strings.TrimSpace(sm.nameInput.Value())
	srvURL := strings.TrimSpace(sm.urlInput.Value())
	if name == "" || srvURL == "" {
		return nil
	}
	if !strings.HasPrefix(srvURL, "ws://") && !strings.HasPrefix(srvURL, "wss://") {
		srvURL = "ws://" + srvURL
	}

	_ = sm.serverList.AddServer(name, srvURL, "")
	sm.addingNew = false
	entry := sm.serverList.Servers[len(sm.serverList.Servers)-1]
	return &entry
}

// Selected returns the currently highlighted server.
func (sm *ServerModal) Selected() *identity.ServerEntry {
	if len(sm.serverList.Servers) == 0 || sm.cursor < 0 || sm.cursor >= len(sm.serverList.Servers) {
		return nil
	}
	return &sm.serverList.Servers[sm.cursor]
}

// View renders the Server Switcher modal.
func (sm *ServerModal) View(screenWidth, screenHeight int, currentURL string) string {
	boxWidth := 68

	if sm.addingNew {
		header := sModalTitle("ADD CUSTOM SKYWAVE NODE")
		nameBox := sm.nameInput.View()
		urlBox := sm.urlInput.View()

		content := lipgloss.JoinVertical(lipgloss.Left,
			header,
			"\n"+nameBox,
			"\n"+urlBox,
			sStatus().Render("\n[Tab]: Switch field  [Enter]: Save & Connect  [Esc]: Back"),
		)
		modalBox := sModal().Width(boxWidth).Render(content)
		return lipgloss.Place(screenWidth, screenHeight, lipgloss.Center, lipgloss.Center, modalBox,
			lipgloss.WithWhitespaceChars(" "), lipgloss.WithWhitespaceForeground(current.Muted))
	}

	header := sModalTitle("SKYWAVE SERVER NODES (Ctrl+S)")

	var list strings.Builder
	list.WriteString("\n")

	for i, s := range sm.serverList.Servers {
		selected := (i == sm.cursor)
		isCurrent := (s.URL == currentURL)

		prefix := "   "
		if selected {
			prefix = " ▶ "
		}

		statusBadge := ""
		if isCurrent {
			statusBadge = lipgloss.NewStyle().Background(current.PanelBg).Foreground(current.Primary).Bold(true).Padding(0, 1).Render("[CONNECTED]")
		}

		cleanName := strings.ReplaceAll(s.Name, "🌊 ", "")
		cleanName = strings.ReplaceAll(cleanName, "🏠 ", "")
		line := fmt.Sprintf("%s%-24s %-26s %s", prefix, cleanName, sStatus().Render(s.URL), statusBadge)
		if selected {
			line = sSpotlightItem(true).Width(boxWidth - 4).Render(line)
		} else {
			line = sSpotlightItem(false).Width(boxWidth - 4).Render(line)
		}
		list.WriteString(line + "\n")
	}

	footer := sStatus().Render("\n  Navigate: [↑/↓]  ·  Connect: [Enter]  ·  Add: [A]  ·  Dismiss: [Esc]")

	content := lipgloss.JoinVertical(lipgloss.Left, header, list.String(), footer)
	modalBox := sModal().Width(boxWidth).Render(content)

	return lipgloss.Place(
		screenWidth,
		screenHeight,
		lipgloss.Center,
		lipgloss.Center,
		modalBox,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(current.Muted),
	)
}
