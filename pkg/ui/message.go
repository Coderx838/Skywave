package ui

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/skywave-chat/skywave/pkg/protocol"
)

// MessageKind specifies message transmission type.
type MessageKind int

const (
	KindChat MessageKind = iota
	KindSystem
	KindError
	KindDM
	KindJoin
	KindLeave
)

// DisplayMessage is one entry in the transceiver chat stream.
type DisplayMessage struct {
	ID          string
	Kind        MessageKind
	Sender      string
	Recipient   string
	Content     string
	Timestamp   time.Time
	IsEncrypted bool
	Mentioned   bool
	Grouped     bool
}

var urlRegex = regexp.MustCompile(`https?://[^\s/$.?#].[^\s]*`)

// Render formats a transmission with clean, aesthetic cyberdeck typography.
func (m DisplayMessage) Render(currentNick string, maxCol int) string {
	ts := m.Timestamp.Format("15:04")
	timeStr := lipgloss.NewStyle().Foreground(current.Muted).Render(ts)
	content := ExpandEmojis(m.Content)

	switch m.Kind {
	case KindSystem:
		icon := lipgloss.NewStyle().Foreground(current.Warning).Render("◆")
		text := lipgloss.NewStyle().Foreground(current.Warning).Render(content)
		return fmt.Sprintf(" %s %s %s", timeStr, icon, text)

	case KindJoin:
		icon := lipgloss.NewStyle().Foreground(current.Success).Render("▸")
		text := lipgloss.NewStyle().Foreground(current.Success).Render(content)
		return fmt.Sprintf(" %s %s %s", timeStr, icon, text)

	case KindLeave:
		icon := lipgloss.NewStyle().Foreground(current.Muted).Render("—")
		text := lipgloss.NewStyle().Foreground(current.Muted).Italic(true).Render(content)
		return fmt.Sprintf(" %s %s %s", timeStr, icon, text)

	case KindError:
		icon := lipgloss.NewStyle().Foreground(current.Danger).Bold(true).Render("▲")
		text := lipgloss.NewStyle().Foreground(current.Danger).Bold(true).Render(content)
		return fmt.Sprintf(" %s %s %s", timeStr, icon, text)

	case KindDM:
		var target string
		if m.Sender == currentNick {
			target = "DM ➔ @" + m.Recipient
		} else {
			target = "DM ➔ @" + m.Sender
		}
		head := lipgloss.NewStyle().Background(current.Accent).Foreground(current.DarkBg).Bold(true).Padding(0, 1).Render(target)
		lock := ""
		if m.IsEncrypted {
			lock = " 🔒"
		}
		body := formatBody(content, currentNick)
		return fmt.Sprintf(" %s %s%s\n         │ %s", timeStr, head, lock, body)

	case KindChat:
		fallthrough
	default:
		// Grouped continuation from same author
		if m.Grouped {
			body := formatBody(content, currentNick)
			return fmt.Sprintf("       │ %s", body)
		}

		var authStr string
		var badge string
		if m.Sender == currentNick {
			authStr = lipgloss.NewStyle().Foreground(current.Success).Bold(true).Render("@" + m.Sender)
			badge = lipgloss.NewStyle().Foreground(current.Muted).Render(" [YOU]")
		} else {
			color := AuthorColor(m.Sender)
			authStr = lipgloss.NewStyle().Foreground(color).Bold(true).Render("@" + m.Sender)
		}

		lock := ""
		if m.IsEncrypted {
			lock = lipgloss.NewStyle().Foreground(current.Accent).Render(" 🔒")
		}

		arrow := lipgloss.NewStyle().Foreground(current.Muted).Render("❯")
		body := formatBody(content, currentNick)
		return fmt.Sprintf(" %s │ %s%s%s %s %s", timeStr, authStr, badge, lock, arrow, body)
	}
}

// formatBody handles code blocks, link colors, and mention highlights.
func formatBody(content, currentNick string) string {
	// Fenced markdown code blocks
	if strings.HasPrefix(content, "```") && strings.HasSuffix(content, "```") {
		trimmed := strings.Trim(content, "`")
		lines := strings.Split(trimmed, "\n")
		top := lipgloss.NewStyle().Foreground(current.Secondary).Render("╭── 💻 CODE ──────────────────────────────")
		bot := lipgloss.NewStyle().Foreground(current.Secondary).Render("╰─────────────────────────────────────────")
		box := lipgloss.NewStyle().Foreground(current.Text).Background(current.PanelBg).Padding(0, 1)

		var b strings.Builder
		b.WriteString("\n" + top + "\n")
		for _, l := range lines {
			b.WriteString(box.Render("│ "+l) + "\n")
		}
		b.WriteString(bot)
		return b.String()
	}

	// URLs
	styled := urlRegex.ReplaceAllStringFunc(content, func(u string) string {
		return lipgloss.NewStyle().Foreground(current.Primary).Underline(true).Render(u)
	})

	// Mentions
	if currentNick != "" && strings.Contains(styled, "@"+currentNick) {
		token := "@" + currentNick
		tag := lipgloss.NewStyle().Background(current.Accent).Foreground(current.DarkBg).Bold(true).Padding(0, 1).Render(token)
		styled = strings.ReplaceAll(styled, token, tag)
	}

	return lipgloss.NewStyle().Foreground(current.Text).Render(styled)
}

// DayDivider renders a clean minimal timeline separator.
func DayDivider(t time.Time) string {
	label := t.Format("Jan 02, 2006")
	if isToday(t) {
		label = "Today"
	}
	bar := lipgloss.NewStyle().Foreground(current.Border).Render("───")
	tag := lipgloss.NewStyle().Foreground(current.Muted).Render(" " + label + " ")
	return " " + bar + tag + bar
}

func isToday(t time.Time) bool {
	n := time.Now()
	return n.Year() == t.Year() && n.YearDay() == t.YearDay()
}

// FromChatPayload converts protocol payload.
func FromChatPayload(p protocol.ChatMsgPayload, currentNick string) DisplayMessage {
	mentioned := currentNick != "" && strings.Contains(p.Content, "@"+currentNick)
	return DisplayMessage{
		ID:          p.ID,
		Kind:        KindChat,
		Sender:      p.Sender,
		Content:     p.Content,
		Timestamp:   time.UnixMilli(p.Timestamp),
		IsEncrypted: p.IsEncrypted,
		Mentioned:   mentioned,
	}
}

// FromDMPayload converts protocol DM payload.
func FromDMPayload(p protocol.DirectMsgPayload) DisplayMessage {
	return DisplayMessage{
		ID:          p.ID,
		Kind:        KindDM,
		Sender:      p.Sender,
		Recipient:   p.Recipient,
		Content:     p.Content,
		Timestamp:   time.UnixMilli(p.Timestamp),
		IsEncrypted: p.IsEncrypted,
	}
}
