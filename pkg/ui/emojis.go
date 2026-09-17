package ui

import "strings"

var emojiMap = map[string]string{
	":fire:":       "🔥",
	":rocket:":     "🚀",
	":wave:":       "👋",
	":skull:":      "💀",
	":heart:":      "❤️",
	":eyes:":       "👀",
	":100:":        "💯",
	":check:":      "✅",
	":x:":          "❌",
	":party:":      "🎉",
	":cat:":        "🐱",
	":dog:":        "🐶",
	":zap:":        "⚡",
	":lock:":       "🔒",
	":unlock:":     "🔓",
	":star:":       "⭐",
	":robot:":      "🤖",
	":alien:":      "👽",
	":ghost:":      "👻",
	":crown:":      "👑",
	":smile:":      "😊",
	":laugh:":      "😂",
	":cry:":        "😭",
	":sunglasses:": "😎",
	":coffee:":     "☕",
	":+1:":         "👍",
	":thumbsup:":   "👍",
	":-1:":         "👎",
	":thumbsdown:": "👎",
	":sparkles:":   "✨",
	":warning:":    "⚠️",
	":radio:":      "📻",
	":ocean:":      "🌊",
	":shield:":     "🛡️",
	":gem:":        "💎",
	":boom:":       "💥",
	":tada:":       "🎉",
}

// ExpandEmojis replaces emoji shortcodes like :fire: with their unicode counterparts.
func ExpandEmojis(text string) string {
	if !strings.Contains(text, ":") {
		return text
	}
	res := text
	for shortcode, emoji := range emojiMap {
		if strings.Contains(res, shortcode) {
			res = strings.ReplaceAll(res, shortcode, emoji)
		}
	}
	return res
}
