package ui

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/skywave-chat/skywave/pkg/client"
	"github.com/skywave-chat/skywave/pkg/identity"
	"github.com/skywave-chat/skywave/pkg/protocol"
)

// MsgPacket wraps incoming protocol.Packet for the event queue.
type MsgPacket protocol.Packet

// MsgError wraps an error event.
type MsgError error

// TickMsg drives clocks, typing expiry, and telemetry.
type TickMsg time.Time

// BootTickMsg drives the animated boot sequence.
type BootTickMsg time.Time

// Model is the Cyberdeck Transceiver TUI state.
type Model struct {
	client       *client.Client
	accountID    string
	isNewAcct    bool
	width        int
	height       int
	viewport     viewport.Model
	textInput    textinput.Model
	spinner      spinner.Model
	roomMessages map[string][]DisplayMessage // Messages segregated per room
	rooms        []protocol.RoomStatus
	topics       map[string]string
	activeRoom   string
	activeUsers  []string
	unread       map[string]int
	dmUnread     int
	mentions     int
	typing       map[string]time.Time
	cmdHistory   []string
	histIndex    int
	toast        string
	toastAt      time.Time
	connected    bool
	showHelp     bool
	ready        bool
	bootAt       time.Time
	lastTyping   time.Time
	partyOn      bool
	bellEnabled  bool

	// Staged Screen Flow
	stage        AppStage
	bootStep     int
	bootProgress float64

	// Required Startup Callsign Prompt
	nameInput textinput.Model
	nameErr   string

	// Radar Souls Drawer (collapsible)
	showRadar bool

	// Overlays & Modals
	palette     PaletteModel
	serverModal ServerModal
	roomModal   RoomModal
}

// NewModel creates the cyberdeck transceiver model.
func NewModel(c *client.Client, accountID string, isNewAccount bool, savedNick string) Model {
	ti := textinput.New()
	ti.Prompt = "" // No duplicate prompt! Handled by renderDeck
	ti.Placeholder = "Type transmission... [Tab: Radar, Ctrl+K: Spotlight, Ctrl+S: Nodes, F1: Deck]"
	ti.Focus()
	ti.CharLimit = protocol.MaxMessageLength
	ti.Width = 80

	// Name prompt input
	nameIn := textinput.New()
	nameIn.Prompt = "❯ "
	nameIn.PromptStyle = lipgloss.NewStyle().Foreground(current.Primary).Bold(true)
	nameIn.TextStyle = lipgloss.NewStyle().Foreground(current.Text).Bold(true)
	nameIn.Placeholder = "Enter callsign (e.g. Ghost, Cipher, Maverick)..."
	nameIn.PlaceholderStyle = lipgloss.NewStyle().Foreground(current.Muted)
	nameIn.CharLimit = 24
	nameIn.Width = 36
	nameIn.Focus()
	// Pre-populate only if a real custom nickname was previously chosen
	if savedNick != "" && !identity.IsGeneratedNick(savedNick) {
		nameIn.SetValue(savedNick)
		nameIn.CursorEnd()
	}

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(current.Primary)

	m := Model{
		client:       c,
		accountID:    accountID,
		isNewAcct:    isNewAccount,
		textInput:    ti,
		nameInput:    nameIn,
		spinner:      sp,
		activeRoom:   "#general",
		topics:       map[string]string{"#general": "Primary Global Frequency"},
		rooms:        []protocol.RoomStatus{{Name: "#general", MemberCount: 1, Topic: "Primary Global Frequency"}},
		roomMessages: make(map[string][]DisplayMessage),
		unread:       map[string]int{},
		typing:       map[string]time.Time{},
		cmdHistory:   []string{},
		histIndex:    -1,
		toast:        "Frequency locked",
		toastAt:      time.Now(),
		bellEnabled:  true,
		showRadar:    false,
		stage:        StageBoot,
		bootStep:     0,
		bootProgress: 0.05,
		palette:      NewPaletteModel(),
		serverModal:  NewServerModal(),
		roomModal:    NewRoomModal(),
	}

	m.roomMessages["#general"] = []DisplayMessage{
		{Kind: KindSystem, Content: "Frequency locked on #general — Anonymous, encrypted & terminal native.", Timestamp: time.Now()},
	}
	return m
}

// Init starts blink, packet listener, spinner, ticker, and boot animator.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		spinner.Tick,
		m.waitForPacket(),
		tickCmd(),
		bootTickCmd(),
	)
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return TickMsg(t) })
}

func bootTickCmd() tea.Cmd {
	return tea.Tick(80*time.Millisecond, func(t time.Time) tea.Msg { return BootTickMsg(t) })
}

// waitForPacket bridges the client channel into Bubble Tea messages.
func (m Model) waitForPacket() tea.Cmd {
	return func() tea.Msg {
		select {
		case pkt, ok := <-m.client.IncomingPackets:
			if !ok {
				return MsgError(fmt.Errorf("frequency interrupted"))
			}
			return MsgPacket(pkt)
		case err := <-m.client.Errors:
			return MsgError(err)
		}
	}
}

// Update handles keys, resize, animation ticks, and network events.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
		cmds  []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.recalculateLayout()

	case BootTickMsg:
		if m.stage == StageBoot {
			m.bootProgress += 0.07
			if m.bootProgress > 0.25 && m.bootStep < 1 {
				m.bootStep = 1
			}
			if m.bootProgress > 0.50 && m.bootStep < 2 {
				m.bootStep = 2
			}
			if m.bootProgress > 0.75 && m.bootStep < 3 {
				m.bootStep = 3
			}
			if m.bootProgress >= 1.0 {
				m.bootProgress = 1.0
				m.bootStep = 4
				m.stage = StageNamePrompt
			} else {
				cmds = append(cmds, bootTickCmd())
			}
		}

	case TickMsg:
		now := time.Now()
		for u, t := range m.typing {
			if now.Sub(t) > 4*time.Second {
				delete(m.typing, u)
			}
		}
		cmds = append(cmds, tickCmd())

	case MsgPacket:
		pkt := protocol.Packet(msg)
		m.handleIncomingPacket(pkt)
		m.refreshChat()
		cmds = append(cmds, m.waitForPacket())

	case MsgError:
		m.connected = false
		m.roomMessages[m.activeRoom] = append(m.roomMessages[m.activeRoom], DisplayMessage{
			Kind: KindError, Content: fmt.Sprintf("Carrier lost: %v", msg), Timestamp: time.Now(),
		})
		m.refreshChat()
		cmds = append(cmds, m.waitForPacket())

	case spinner.TickMsg:
		var c tea.Cmd
		m.spinner, c = m.spinner.Update(msg)
		cmds = append(cmds, c)

	case tea.KeyMsg:
		// Stage 0: Boot Sequence Key Handling
		if m.stage == StageBoot {
			switch msg.Type {
			case tea.KeyEnter, tea.KeySpace, tea.KeyEsc:
				m.stage = StageNamePrompt
			case tea.KeyCtrlC:
				m.client.Disconnect()
				return m, tea.Quit
			}
			return m, nil
		}

		// Stage 1: Callsign Selection Screen
		if m.stage == StageNamePrompt {
			switch msg.Type {
			case tea.KeyEnter:
				val := strings.TrimSpace(m.nameInput.Value())
				if val == "" {
					m.nameErr = "Callsign is required to enter the frequency"
					return m, nil
				}
				clean, err := protocol.ValidateNickname(val)
				if err != nil {
					m.nameErr = "Must be 2-24 characters (letters, numbers, _, -)"
					return m, nil
				}
				m.nameErr = ""
				m.client.Nickname = clean
				if err := m.client.Authenticate(clean); err != nil {
					m.nameErr = "Carrier error: " + err.Error()
					return m, nil
				}
				id, _, _ := identity.LoadOrCreate(clean)
				if id != nil {
					_ = id.SetNickname(clean)
				}
				// Stage transition is triggered when server sends TypeAuthAck
				return m, nil
			case tea.KeyCtrlC, tea.KeyEsc:
				m.client.Disconnect()
				return m, tea.Quit
			default:
				var c tea.Cmd
				m.nameInput, c = m.nameInput.Update(msg)
				return m, c
			}
		}

		// Stage 2: Welcome Dashboard Key Handling
		if m.stage == StageWelcome {
			switch msg.Type {
			case tea.KeyEnter, tea.KeySpace:
				m.stage = StageChat
				m.recalculateLayout()
				return m, nil
			case tea.KeyEsc:
				m.client.Disconnect()
				return m, tea.Quit
			case tea.KeyCtrlT:
				next := CycleTheme()
				m.toast = "Theme: " + strings.ToUpper(next)
				m.toastAt = time.Now()
				return m, nil
			case tea.KeyCtrlS:
				m.serverModal.Open(m.client.ServerURL)
				return m, nil
			case tea.KeyF1:
				m.showHelp = true
				return m, nil
			case tea.KeyRunes:
				r := string(msg.Runes)
				if r == "s" || r == "S" {
					m.serverModal.Open(m.client.ServerURL)
					return m, nil
				} else if r == "n" || r == "N" {
					m.stage = StageNamePrompt
					return m, nil
				} else if r == "q" || r == "Q" {
					m.client.Disconnect()
					return m, tea.Quit
				} else if r == "?" {
					m.showHelp = true
					return m, nil
				}
			}
		}

		// Modals & Overlays Handling
		if m.showHelp {
			if msg.Type == tea.KeyEsc || msg.Type == tea.KeyEnter || msg.Type == tea.KeyCtrlC {
				m.showHelp = false
			}
			if msg.Type == tea.KeyCtrlC {
				m.client.Disconnect()
				return m, tea.Quit
			}
			return m, nil
		}

		if m.palette.IsActive() {
			switch msg.Type {
			case tea.KeyEsc:
				m.palette.Close()
			case tea.KeyUp:
				m.palette.MoveUp()
			case tea.KeyDown:
				m.palette.MoveDown()
			case tea.KeyEnter:
				item := m.palette.Selected()
				m.palette.Close()
				if item != nil {
					switch item.Type {
					case "channel":
						m.switchRoom(item.Value)
					case "user":
						m.textInput.SetValue("/dm @" + item.Value + " ")
						m.textInput.CursorEnd()
						m.textInput.Focus()
					case "action":
						quit, extra := m.handleSend(item.Value)
						if quit {
							m.client.Disconnect()
							return m, tea.Quit
						}
						if extra != nil {
							cmds = append(cmds, extra...)
						}
					}
				}
			default:
				var c tea.Cmd
				m.palette.input, c = m.palette.input.Update(msg)
				m.palette.filter()
				cmds = append(cmds, c)
			}
			return m, tea.Batch(cmds...)
		}

		if m.serverModal.IsActive() {
			switch msg.Type {
			case tea.KeyEsc:
				if m.serverModal.addingNew {
					m.serverModal.CancelAdd()
				} else {
					m.serverModal.Close()
				}
			case tea.KeyUp:
				m.serverModal.MoveUp()
			case tea.KeyDown:
				m.serverModal.MoveDown()
			case tea.KeyTab:
				if m.serverModal.addingNew {
					if m.serverModal.activeField == 0 {
						m.serverModal.activeField = 1
						m.serverModal.nameInput.Blur()
						m.serverModal.urlInput.Focus()
					} else {
						m.serverModal.activeField = 0
						m.serverModal.urlInput.Blur()
						m.serverModal.nameInput.Focus()
					}
				}
			case tea.KeyEnter:
				if m.serverModal.addingNew {
					entry := m.serverModal.SubmitAdd()
					if entry != nil {
						m.toast = "Registered: " + entry.Name
						m.roomMessages = make(map[string][]DisplayMessage)
						m.activeRoom = "#general"
						_ = m.client.SwitchServer(entry.URL, entry.Password)
						m.serverModal.Close()
						m.toast = "Dialing " + entry.URL
						m.refreshChat()
						cmds = append(cmds, m.waitForPacket())
					}
				} else {
					srv := m.serverModal.Selected()
					m.serverModal.Close()
					if srv != nil {
						m.toast = "Switching node: " + srv.Name
						m.roomMessages = make(map[string][]DisplayMessage)
						m.activeRoom = "#general"
						_ = m.client.SwitchServer(srv.URL, srv.Password)
						m.refreshChat()
						cmds = append(cmds, m.waitForPacket())
					}
				}
			case tea.KeyRunes:
				if !m.serverModal.addingNew {
					if string(msg.Runes) == "a" || string(msg.Runes) == "A" || string(msg.Runes) == "+" {
						m.serverModal.StartAdd()
						return m, nil
					}
				} else {
					if m.serverModal.activeField == 0 {
						var c tea.Cmd
						m.serverModal.nameInput, c = m.serverModal.nameInput.Update(msg)
						cmds = append(cmds, c)
					} else {
						var c tea.Cmd
						m.serverModal.urlInput, c = m.serverModal.urlInput.Update(msg)
						cmds = append(cmds, c)
					}
				}
			default:
				if m.serverModal.addingNew {
					if m.serverModal.activeField == 0 {
						var c tea.Cmd
						m.serverModal.nameInput, c = m.serverModal.nameInput.Update(msg)
						cmds = append(cmds, c)
					} else {
						var c tea.Cmd
						m.serverModal.urlInput, c = m.serverModal.urlInput.Update(msg)
						cmds = append(cmds, c)
					}
				}
			}
			return m, tea.Batch(cmds...)
		}

		if m.roomModal.IsActive() {
			switch msg.Type {
			case tea.KeyEsc:
				m.roomModal.Close()
			case tea.KeyTab:
				m.roomModal.SwitchField()
			case tea.KeyEnter:
				name, pass := m.roomModal.Values()
				m.roomModal.Close()
				if name != "" {
					_ = m.client.JoinRoom(name, pass)
					m.activeRoom = name
					if pass != "" {
						m.toast = "Encrypted frequency tuned: " + name
					} else {
						m.toast = "Tuned into " + name
					}
					m.refreshChat()
				}
			default:
				if m.roomModal.activeTab == 0 {
					var c tea.Cmd
					m.roomModal.nameInput, c = m.roomModal.nameInput.Update(msg)
					cmds = append(cmds, c)
				} else {
					var c tea.Cmd
					m.roomModal.passInput, c = m.roomModal.passInput.Update(msg)
					cmds = append(cmds, c)
				}
			}
			return m, tea.Batch(cmds...)
		}

		// Stage 3: Chat Key Handling
		switch msg.Type {
		case tea.KeyCtrlC:
			m.client.Disconnect()
			return m, tea.Quit

		case tea.KeyCtrlK, tea.KeyCtrlP:
			rooms := make([]string, 0, len(m.rooms))
			for _, r := range m.rooms {
				rooms = append(rooms, r.Name)
			}
			m.palette.Open(rooms, m.activeUsers)
			return m, nil

		case tea.KeyCtrlS:
			m.serverModal.Open(m.client.ServerURL)
			return m, nil

		case tea.KeyCtrlN:
			m.roomModal.Open()
			return m, nil

		case tea.KeyCtrlT:
			next := CycleTheme()
			m.toast = "Theme: " + strings.ToUpper(next)
			m.toastAt = time.Now()
			m.refreshChat()
			return m, nil

		case tea.KeyCtrlB:
			m.bellEnabled = !m.bellEnabled
			st := "ON 🔔"
			if !m.bellEnabled {
				st = "OFF 🔕"
			}
			m.toast = "Bell: " + st
			m.toastAt = time.Now()
			return m, nil

		case tea.KeyTab:
			m.showRadar = !m.showRadar
			m.recalculateLayout()
			return m, nil

		case tea.KeyF1:
			m.showHelp = true
			return m, nil

		case tea.KeyPgUp:
			m.viewport.HalfViewUp()
			return m, nil

		case tea.KeyPgDown:
			m.viewport.HalfViewDown()
			return m, nil

		case tea.KeyEsc:
			if m.textInput.Value() != "" {
				m.textInput.Reset()
			} else {
				m.stage = StageWelcome
			}
			return m, nil

		case tea.KeyEnter:
			text := strings.TrimSpace(m.textInput.Value())
			if text != "" {
				quit, follow := m.handleSend(text)
				m.cmdHistory = append(m.cmdHistory, text)
				m.histIndex = len(m.cmdHistory)
				m.textInput.Reset()
				if follow != nil {
					cmds = append(cmds, follow...)
				}
				if quit {
					m.client.Disconnect()
					return m, tea.Quit
				}
			}
			return m, nil

		case tea.KeyUp:
			if len(m.cmdHistory) > 0 && m.histIndex > 0 {
				m.histIndex--
				m.textInput.SetValue(m.cmdHistory[m.histIndex])
				m.textInput.CursorEnd()
			}
			return m, nil

		case tea.KeyDown:
			if len(m.cmdHistory) > 0 && m.histIndex < len(m.cmdHistory)-1 {
				m.histIndex++
				m.textInput.SetValue(m.cmdHistory[m.histIndex])
				m.textInput.CursorEnd()
			} else if m.histIndex >= len(m.cmdHistory)-1 {
				m.histIndex = len(m.cmdHistory)
				m.textInput.Reset()
			}
			return m, nil

		case tea.KeyRunes:
			if m.textInput.Value() == "" {
				r := string(msg.Runes)
				if r >= "1" && r <= "9" {
					idx := int(r[0]-'1')
					if idx < len(m.rooms) {
						m.switchRoom(m.rooms[idx].Name)
						return m, nil
					}
				} else if r == "[" {
					m.cycleRoom(-1)
					return m, nil
				} else if r == "]" {
					m.cycleRoom(1)
					return m, nil
				}
			}

			if time.Since(m.lastTyping) > 2500*time.Millisecond && m.connected {
				m.lastTyping = time.Now()
				_ = m.client.SendTyping()
			}
		}
	}

	m.textInput, tiCmd = m.textInput.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)
	cmds = append(cmds, tiCmd, vpCmd)

	return m, tea.Batch(cmds...)
}

// switchRoom cleanly switches the active frequency, updates client room, clears unread, and refreshes.
func (m *Model) switchRoom(target string) {
	if target == "" || target == m.activeRoom {
		return
	}
	m.activeRoom = target
	m.client.CurrentRoom = target
	_ = m.client.JoinRoom(target, "")
	m.unread[target] = 0
	m.toast = "Tuned into " + target
	m.toastAt = time.Now()
	m.refreshChat()
}

// hasMessage checks if a message ID has already been recorded in a room.
func (m *Model) hasMessage(room, id string) bool {
	if id == "" {
		return false
	}
	for _, msg := range m.roomMessages[room] {
		if msg.ID == id {
			return true
		}
	}
	return false
}

func (m *Model) cycleRoom(delta int) {
	if len(m.rooms) == 0 {
		return
	}
	curIdx := 0
	for i, r := range m.rooms {
		if r.Name == m.activeRoom {
			curIdx = i
			break
		}
	}
	nextIdx := (curIdx + delta + len(m.rooms)) % len(m.rooms)
	m.switchRoom(m.rooms[nextIdx].Name)
}

// recalculateLayout handles clean full-width sizing.
func (m *Model) recalculateLayout() {
	headerH := 2
	deckH := 3
	statusH := 1
	canvasH := m.height - headerH - deckH - statusH
	if canvasH < 5 {
		canvasH = 5
	}

	chatW := m.width - 2
	if m.showRadar && m.width >= 90 {
		chatW = m.width - 26
	}
	if chatW < 24 {
		chatW = 24
	}

	if !m.ready {
		m.viewport = viewport.New(chatW, canvasH)
		m.ready = true
	} else {
		m.viewport.Width = chatW
		m.viewport.Height = canvasH
	}
	m.textInput.Width = chatW - 16
	if m.textInput.Width < 10 {
		m.textInput.Width = 10
	}
	m.refreshChat()
}

// handleSend parses commands or sends chat.
func (m *Model) handleSend(input string) (bool, []tea.Cmd) {
	lower := strings.ToLower(strings.TrimSpace(input))

	switch lower {
	case "skywave rocks", "skywave forever":
		m.addMsg(KindSystem, "✨ The frequency resonates with your devotion. +100 aura.")
		return false, nil
	case "iddqd":
		m.addMsg(KindSystem, "🔫 God mode: active. Cyberdeck frequency calibrated.")
		return false, nil
	}

	if !strings.HasPrefix(input, "/") {
		if err := m.client.SendChat(input); err != nil {
			m.addMsg(KindError, "Carrier dropped: "+err.Error())
		}
		return false, nil
	}

	parts := strings.Fields(input)
	cmd := strings.ToLower(parts[0])
	args := parts[1:]

	switch cmd {
	case "/help", "/?":
		m.showHelp = true
	case "/join", "/j":
		if len(args) == 0 {
			m.addMsg(KindError, "usage: /join <#room> [passcode]")
			return false, nil
		}
		room := args[0]
		pass := ""
		if len(args) > 1 {
			pass = args[1]
		}
		_ = m.client.JoinRoom(room, pass)
		m.activeRoom = room
		if pass != "" {
			m.toast = "Locked into E2EE room " + room
		} else {
			m.toast = "Tuned into " + room
		}
		m.refreshChat()
	case "/leave", "/part":
		_ = m.client.JoinRoom("#general", "")
		m.activeRoom = "#general"
		m.toast = "Returned to #general"
		m.refreshChat()
	case "/dm", "/msg", "/tell":
		if len(args) < 2 {
			m.addMsg(KindError, "usage: /dm <@user> <message>")
			return false, nil
		}
		target := strings.TrimPrefix(args[0], "@")
		msgBody := strings.Join(args[1:], " ")
		if err := m.client.SendDirectMsg(target, msgBody); err != nil {
			m.addMsg(KindError, "dm failed: "+err.Error())
		}
	case "/nick":
		if len(args) < 1 {
			m.addMsg(KindError, "usage: /nick <new_name>")
			return false, nil
		}
		if err := m.client.ChangeNick(args[0]); err != nil {
			m.addMsg(KindError, err.Error())
		} else {
			m.client.Nickname = args[0]
			id, _, _ := identity.LoadOrCreate(args[0])
			if id != nil {
				_ = id.SetNickname(args[0])
			}
		}
	case "/server":
		m.serverModal.Open(m.client.ServerURL)
	case "/bell":
		m.bellEnabled = !m.bellEnabled
		st := "ENABLED"
		if !m.bellEnabled {
			st = "MUTED"
		}
		m.addMsg(KindSystem, "🔔 Chime bell is now "+st)
	case "/theme":
		if len(args) < 1 {
			m.addMsg(KindSystem, "available themes: "+strings.Join(ThemeNames(), ", "))
			return false, nil
		}
		if SetTheme(args[0]) {
			m.toast = "Theme: " + args[0]
			m.refreshChat()
		} else {
			m.addMsg(KindError, "unknown theme — choose: "+strings.Join(ThemeNames(), ", "))
		}
	case "/topic":
		if len(args) < 1 {
			m.addMsg(KindSystem, "Topic: "+m.topics[m.activeRoom])
			return false, nil
		}
		topic := strings.Join(args, " ")
		_ = m.client.SetTopic(topic)
	case "/whoami", "/id":
		id, _, _ := identity.LoadOrCreate(m.client.Nickname)
		if id != nil {
			m.addMsg(KindSystem, fmt.Sprintf("Identity: @%s | Ghost Account: %s", id.Nickname, id.AccountID))
		}
	case "/party":
		m.partyOn = !m.partyOn
		st := "OFF"
		if m.partyOn {
			st = "ON 🎉"
		}
		m.addMsg(KindSystem, "Party mode: "+st)
	case "/clear":
		m.roomMessages[m.activeRoom] = []DisplayMessage{}
		m.refreshChat()
	case "/hack":
		for _, l := range []string{
			"Decentralized relay handshaking...",
			"Injecting AES-256 ephemeral keys...",
			"Zero-knowledge cipher verified. Root access: Granted. 😎",
		} {
			m.addMsg(KindSystem, l)
		}
	case "/matrix":
		for _, l := range []string{
			"Wake up, @" + m.client.Nickname + "...",
			"The wave has you...",
			"Follow the white rabbit. 🐇",
		} {
			m.addMsg(KindSystem, l)
		}
	case "/quit", "/exit", "/q":
		return true, nil
	default:
		m.addMsg(KindError, fmt.Sprintf("Unknown ritual '%s' — press F1 for manual", cmd))
	}
	return false, nil
}

// handleIncomingPacket translates server packets into per-room feeds.
func (m *Model) handleIncomingPacket(p protocol.Packet) {
	switch p.Type {
	case protocol.TypeAuthAck:
		raw, _ := json.Marshal(p.Payload)
		var ack protocol.AuthAckPayload
		if err := json.Unmarshal(raw, &ack); err == nil {
			m.rooms = ack.Rooms
			for _, r := range ack.Rooms {
				if r.Topic != "" {
					m.topics[r.Name] = r.Topic
				}
			}
			if ack.Nickname != "" {
				m.client.Nickname = ack.Nickname
			}
			m.connected = true
			if ack.MOTD != "" {
				m.toast = "MOTD: " + ack.MOTD
			}
			if m.stage == StageNamePrompt || m.stage == StageBoot {
				m.stage = StageChat
				m.recalculateLayout()
			}
		}

	case protocol.TypeChatMsg:
		raw, _ := json.Marshal(p.Payload)
		var chat protocol.ChatMsgPayload
		if err := json.Unmarshal(raw, &chat); err == nil {
			dm := FromChatPayload(chat, m.client.Nickname)
			if dm.Mentioned {
				m.mentions++
				m.toast = "Mentioned by @" + chat.Sender
				m.toastAt = time.Now()
				if m.bellEnabled {
					fmt.Print("\a")
				}
			}
			targetRoom := chat.Room
			if targetRoom == "" {
				targetRoom = "#general"
			}
			if targetRoom != m.activeRoom {
				m.unread[targetRoom]++
			}
			if m.partyOn {
				dm.Content = ">> " + dm.Content + " <<"
			}
			// Isolate to specific room & deduplicate
			if !m.hasMessage(targetRoom, chat.ID) {
				m.roomMessages[targetRoom] = append(m.roomMessages[targetRoom], dm)
			}
		}

	case protocol.TypeDirectMsg:
		raw, _ := json.Marshal(p.Payload)
		var msgData protocol.DirectMsgPayload
		if err := json.Unmarshal(raw, &msgData); err == nil {
			m.dmUnread++
			dm := FromDMPayload(msgData)
			if !m.hasMessage(m.activeRoom, msgData.ID) {
				m.roomMessages[m.activeRoom] = append(m.roomMessages[m.activeRoom], dm)
			}
			m.toast = "Direct transmission from @" + msgData.Sender
			m.toastAt = time.Now()
			if m.bellEnabled {
				fmt.Print("\a")
			}
		}

	case protocol.TypeTyping:
		raw, _ := json.Marshal(p.Payload)
		var t protocol.TypingPayload
		if err := json.Unmarshal(raw, &t); err == nil && t.User != m.client.Nickname {
			m.typing[t.User] = time.Now()
		}

	case protocol.TypeSystemMsg:
		raw, _ := json.Marshal(p.Payload)
		var sys protocol.SystemMsgPayload
		if err := json.Unmarshal(raw, &sys); err == nil {
			kind := KindSystem
			l := strings.ToLower(sys.Content)
			if strings.Contains(l, "tuned into") || strings.Contains(l, "joined") || strings.Contains(l, "[+]") {
				kind = KindJoin
			} else if strings.Contains(l, "left") || strings.Contains(l, "switched") || strings.Contains(l, "[-]") {
				kind = KindLeave
			}

			targetRoom := sys.Room
			if targetRoom == "" {
				targetRoom = m.activeRoom
			}

			m.roomMessages[targetRoom] = append(m.roomMessages[targetRoom], DisplayMessage{Kind: kind, Content: sys.Content, Timestamp: time.Now()})

			if targetRoom != m.activeRoom && kind == KindSystem {
				m.unread[targetRoom]++
			}
		}

	case protocol.TypeErrorMsg:
		raw, _ := json.Marshal(p.Payload)
		var e protocol.ErrorMsgPayload
		if err := json.Unmarshal(raw, &e); err == nil {
			if e.Code == "NICK_IN_USE" || e.Code == "INVALID_NICK" {
				if m.stage == StageNamePrompt || m.stage == StageBoot {
					m.stage = StageNamePrompt
					m.nameErr = e.Message
					m.nameInput.Focus()
					return
				}
				m.addMsg(KindError, "▲ "+e.Message)
				m.toast = "▲ " + e.Message
				m.toastAt = time.Now()
				return
			}
			m.roomMessages[m.activeRoom] = append(m.roomMessages[m.activeRoom], DisplayMessage{Kind: KindError, Content: e.Message, Timestamp: time.Now()})
			m.toast = "▲ " + e.Message
			m.toastAt = time.Now()
		}

	case protocol.TypeRoomList:
		raw, _ := json.Marshal(p.Payload)
		var rList []protocol.RoomStatus
		if err := json.Unmarshal(raw, &rList); err == nil {
			m.rooms = rList
			for _, r := range rList {
				if r.Topic != "" {
					m.topics[r.Name] = r.Topic
				}
			}
		}

	case protocol.TypeUserList:
		raw, _ := json.Marshal(p.Payload)
		var uList protocol.UserListPayload
		if err := json.Unmarshal(raw, &uList); err == nil {
			if uList.Room == m.activeRoom || uList.Room == "" {
				m.activeUsers = uList.Users
			}
		}
	}
}

func (m *Model) addMsg(kind MessageKind, content string) {
	m.roomMessages[m.activeRoom] = append(m.roomMessages[m.activeRoom], DisplayMessage{Kind: kind, Content: content, Timestamp: time.Now()})
	m.refreshChat()
}

// refreshChat renders transmissions exclusively for activeRoom.
func (m *Model) refreshChat() {
	var lines []string
	var lastDay string
	var lastSender string
	var lastTime time.Time

	// Clean, centered header card for active channel
	channelIntro := lipgloss.NewStyle().Foreground(current.Secondary).Render(fmt.Sprintf("── ◈ %s ──", m.activeRoom))
	topicStr := m.topics[m.activeRoom]
	if topicStr == "" {
		topicStr = "Active frequency on the decentralized wave."
	}
	topicCard := lipgloss.NewStyle().Foreground(current.TextDim).Italic(true).Render(topicStr)
	lines = append(lines, "", " "+channelIntro, " "+topicCard, "")

	// E2EE Banner if active
	if key := m.client.GetRoomKey(m.activeRoom); key != "" {
		bannerTop := lipgloss.NewStyle().Foreground(current.Primary).Render(" ╭── 🔒 E2EE SHIELD ACTIVE // AES-256-GCM ─────────────────────────────")
		bannerMid := lipgloss.NewStyle().Foreground(current.TextDim).Render(" │  Passphrase derived key active. Server relay only sees ciphertext.")
		bannerBot := lipgloss.NewStyle().Foreground(current.Primary).Render(" ╰────────────────────────────────────────────────────────────────────")
		lines = append(lines, bannerTop, bannerMid, bannerBot, "")
	}

	activeMsgs := m.roomMessages[m.activeRoom]
	if len(activeMsgs) == 0 {
		emptyNotice := lipgloss.NewStyle().Foreground(current.Muted).Italic(true).Render(fmt.Sprintf("  Frequency %s is quiet. Be the first to transmit.", m.activeRoom))
		lines = append(lines, "", emptyNotice)
	}
	for i, msg := range activeMsgs {
		day := msg.Timestamp.Format("2006-01-02")
		if day != lastDay {
			lines = append(lines, DayDivider(msg.Timestamp))
			lastDay = day
			lastSender = ""
		}

		// Message grouping
		isGrouped := false
		if msg.Kind == KindChat && lastSender == msg.Sender && msg.Timestamp.Sub(lastTime) < 90*time.Second {
			isGrouped = true
		}
		activeMsgs[i].Grouped = isGrouped

		lines = append(lines, activeMsgs[i].Render(m.client.Nickname, m.viewport.Width))
		lastSender = msg.Sender
		lastTime = msg.Timestamp
	}

	m.viewport.SetContent(strings.Join(lines, "\n"))
	m.viewport.GotoBottom()
}

// View renders based on active stage.
func (m Model) View() string {
	if !m.ready {
		return "\n  " + m.spinner.View() + "  Initializing Cyberdeck Transceiver...\n"
	}

	// Staged Screen 1: Animated Boot Sequence
	if m.stage == StageBoot {
		return RenderBootScreen(m.width, m.height, m.bootStep, m.bootProgress)
	}

	// Staged Screen 2: Required Startup Callsign Selection
	if m.stage == StageNamePrompt {
		return RenderNamePrompt(m.width, m.height, m.nameInput.View(), m.nameErr)
	}

	// Staged Screen 3: Welcome Dashboard Gateway
	if m.stage == StageWelcome {
		return RenderWelcomeDashboard(m.width, m.height, m.client.Nickname, m.accountID, m.client.ServerName, m.client.ServerURL, 12)
	}

	// Overlays (active on top of StageChat)
	if m.palette.IsActive() {
		return m.palette.View(m.width, m.height)
	}
	if m.serverModal.IsActive() {
		return m.serverModal.View(m.width, m.height, m.client.ServerURL)
	}
	if m.roomModal.IsActive() {
		return m.roomModal.View(m.width, m.height)
	}

	// Staged Screen 4: Dedicated Cyberdeck Chat
	header := m.renderHeader()
	canvas := sCanvas().Width(m.viewport.Width).Height(m.viewport.Height).Render(m.viewport.View())

	var centerRow string
	if m.showRadar && m.width >= 90 {
		radarDrawer := m.renderRadar(24, m.viewport.Height)
		centerRow = lipgloss.JoinHorizontal(lipgloss.Top, canvas, radarDrawer)
	} else {
		centerRow = canvas
	}

	deck := m.renderDeck()
	status := m.renderStatusline()

	base := lipgloss.JoinVertical(lipgloss.Left, header, centerRow, deck, status)
	if m.showHelp {
		return m.overlayHelp(base)
	}
	return base
}

func (m Model) renderHeader() string {
	logo := sBrand().Render("SKYWAVE")

	var tabBuilder strings.Builder
	for i, r := range m.rooms {
		if i >= 6 {
			break
		}
		if r.Name == m.activeRoom {
			tabBuilder.WriteString(sTabActive(r.Name, r.MemberCount, r.IsPrivate))
		} else {
			tabBuilder.WriteString(sTabInactive(i+1, r.Name, r.MemberCount, r.IsPrivate))
		}
	}
	tabBuilder.WriteString(sTabAdd())

	left := lipgloss.JoinHorizontal(lipgloss.Center, logo, "  ", tabBuilder.String())

	radarStatus := lipgloss.NewStyle().Foreground(current.TextDim).Render(fmt.Sprintf("● %d SOULS", len(m.activeUsers)))
	if m.showRadar {
		radarStatus = lipgloss.NewStyle().Background(current.PanelBg).Foreground(current.Primary).Bold(true).Padding(0, 1).Render(fmt.Sprintf("◈ %d SOULS [RADAR ON]", len(m.activeUsers)))
	}

	clock := lipgloss.NewStyle().Foreground(current.Muted).Render(time.Now().Format("15:04"))
	ping := lipgloss.NewStyle().Foreground(current.Primary).Render("● 12ms")
	themePill := lipgloss.NewStyle().Foreground(current.Secondary).Render(strings.ToUpper(ThemeName()))

	right := lipgloss.JoinHorizontal(lipgloss.Center, radarStatus, "   ", ping, "   ", clock, "   ", themePill)

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if gap < 1 {
		gap = 1
	}

	return sHeaderBar().Width(m.width).Render(
		lipgloss.JoinHorizontal(lipgloss.Center, left, strings.Repeat(" ", gap), right),
	)
}

func (m Model) renderRadar(width, height int) string {
	var b strings.Builder
	title := lipgloss.NewStyle().Background(current.PanelBg).Foreground(current.Text).Bold(true).Render(" ◈ SOULS ON RADAR ")
	b.WriteString(title + "\n\n")

	users := append([]string(nil), m.activeUsers...)
	sort.Strings(users)

	for _, u := range users {
		prefix := lipgloss.NewStyle().Foreground(current.Primary).Render("● ")
		style := lipgloss.NewStyle().Foreground(current.TextDim)
		suffix := ""
		if u == m.client.Nickname {
			style = lipgloss.NewStyle().Foreground(current.Text).Bold(true)
			suffix = " (YOU)"
		}
		if _, ok := m.typing[u]; ok {
			suffix += " ~"
		}
		b.WriteString(fmt.Sprintf(" %s%s%s\n", prefix, style.Render(u), suffix))
	}

	b.WriteString("\n" + lipgloss.NewStyle().Foreground(current.Muted).Render(" [Tab]: Close Radar\n /dm @nick <msg>"))

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Background(current.PanelBg).
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(current.Border).
		Padding(0, 1).
		Render(b.String())
}

func (m Model) renderDeck() string {
	// Clean, single prompt prefix (ti.Prompt is empty to prevent duplicate "> >")
	prompt := sDeckPrompt(m.activeRoom).Render(m.activeRoom + " ❯ ")

	typingNotice := ""
	if len(m.typing) > 0 {
		var nicks []string
		for u := range m.typing {
			nicks = append(nicks, "@"+u)
		}
		typingNotice = lipgloss.NewStyle().Foreground(current.Warning).Italic(true).Render(" ~ " + strings.Join(nicks, ", ") + " transmitting…")
	}

	inputLine := lipgloss.JoinHorizontal(lipgloss.Center, prompt, m.textInput.View(), typingNotice)
	return sDeck().Width(m.width).Render(inputLine)
}

func (m Model) renderStatusline() string {
	toast := m.toast
	if time.Since(m.toastAt) > 6*time.Second {
		toast = fmt.Sprintf("@%s on %s", m.client.Nickname, m.client.ServerName)
	}

	left := lipgloss.NewStyle().Foreground(current.TextDim).Render(" " + toast)
	right := lipgloss.NewStyle().Foreground(current.Muted).Render("[1-9]: Dial · [Tab]: Radar · [Ctrl+K]: Spotlight · [Ctrl+S]: Nodes · [Esc]: Gateway · [F1]: Help ")

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if gap < 1 {
		gap = 1
	}

	return sStatusline().Width(m.width).Render(
		lipgloss.JoinHorizontal(lipgloss.Center, left, strings.Repeat(" ", gap), right),
	)
}

// overlayHelp renders the Cyberdeck manual.
func (m Model) overlayHelp(_ string) string {
	rows := []string{
		"  🌊 SKYWAVE TRANSCEIVER // COMMAND DECK",
		"  ──────────────────────────────────────────────────────────────────",
		"  FREQUENCY & RADAR NAVIGATION",
		"  1 .. 9            Quick-dial frequency tabs (when prompt empty)",
		"  [  and  ]         Cycle to previous / next frequency channel",
		"  Tab               Toggle Souls Radar Drawer (live member radar)",
		"  Ctrl+K / Ctrl+P   Spotlight Quick Jumper (Jump to room/soul/action)",
		"  Ctrl+S            Server Switcher Drawer (Hot-swap server nodes)",
		"  Ctrl+N            Tune New Frequency / Encrypted E2EE Room",
		"  Ctrl+T            Cycle Cyber Themes (Obsidian, Cognac, Monochrome)",
		"  Ctrl+B            Toggle Audio Chime Bell (Mentions & DMs)",
		"  PgUp / PgDn       Scroll transmission history",
		"  Esc               Cancel / return to Gateway Screen",
		"",
		"  CORE TRANSMISSIONS",
		"  ──────────────────────────────────────────────────────────────────",
		"  /join <#room> [key]    Tune room · [key] triggers AES-256 E2EE shield",
		"  /leave                 Return to #general",
		"  /dm <@nick> <msg>      Direct encrypted 1-on-1 transmission",
		"  /nick <new_name>       Rename persistent anonymous callsign",
		"  /topic <text>          Calibrate channel frequency topic",
		"  /theme <name>          obsidian, cognac, monochrome, crimson-night",
		"  /server                Open Server Node Manager",
		"  /party, /hack, /matrix Classified cyber easter eggs",
		"  /clear                 Wipe canvas history",
		"  /quit                  Disconnect and power down transceiver",
	}

	body := sModalTitle("CYBERDECK REFERENCE MANUAL") + "\n" +
		strings.Join(rows, "\n") + "\n\n" +
		lipgloss.NewStyle().Foreground(current.Muted).Render("Press ESC or ENTER to return to transceiver")

	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		sModal().Render(body),
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(current.Muted),
	)
}
