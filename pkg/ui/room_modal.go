package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// RoomModal manages the "New Channel / Encrypted Room" dialog (Ctrl+N).
type RoomModal struct {
	active     bool
	nameInput  textinput.Model
	passInput  textinput.Model
	activeTab  int // 0 = name, 1 = pass
}

// NewRoomModal initializes the room creation modal.
func NewRoomModal() RoomModal {
	nameIn := textinput.New()
	nameIn.Placeholder = "#room-name (e.g. #underground)"
	nameIn.Prompt = "Channel: "
	nameIn.CharLimit = 32

	passIn := textinput.New()
	passIn.Placeholder = "Optional passcode (enables E2EE AES-256)"
	passIn.Prompt = "Secret:  "
	passIn.EchoMode = textinput.EchoPassword
	passIn.CharLimit = 64

	return RoomModal{
		nameInput: nameIn,
		passInput: passIn,
	}
}

// Open activates the modal.
func (rm *RoomModal) Open() {
	rm.active = true
	rm.activeTab = 0
	rm.nameInput.Reset()
	rm.nameInput.Focus()
	rm.passInput.Reset()
	rm.passInput.Blur()
}

// Close dismisses the modal.
func (rm *RoomModal) Close() {
	rm.active = false
}

// IsActive returns whether modal is active.
func (rm *RoomModal) IsActive() bool {
	return rm.active
}

// SwitchField toggles between name and password input.
func (rm *RoomModal) SwitchField() {
	if rm.activeTab == 0 {
		rm.activeTab = 1
		rm.nameInput.Blur()
		rm.passInput.Focus()
	} else {
		rm.activeTab = 0
		rm.passInput.Blur()
		rm.nameInput.Focus()
	}
}

// Values returns the entered channel name and passphrase.
func (rm *RoomModal) Values() (string, string) {
	name := strings.TrimSpace(rm.nameInput.Value())
	pass := strings.TrimSpace(rm.passInput.Value())
	return name, pass
}

// View renders the modal.
func (rm *RoomModal) View(screenWidth, screenHeight int) string {
	boxWidth := 54

	header := sModalTitle("TUNE NEW FREQUENCY / E2EE (Ctrl+N)")
	shieldNotice := sStatus().Render("Passphrase triggers client-side AES-256-GCM E2EE.")

	content := lipgloss.JoinVertical(lipgloss.Left,
		header,
		"\n"+rm.nameInput.View(),
		"\n"+rm.passInput.View(),
		"\n"+shieldNotice,
		sStatus().Render("\n[Tab]: Switch Field  [Enter]: Join/Create  [Esc]: Cancel"),
	)

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
