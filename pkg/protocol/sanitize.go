package protocol

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"unicode"
)

// ANSI escape sequence matcher to sanitize untrusted terminal text
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b\].*?\x07|\x1b[PX^_].*?\x1b\\`)

// Regex for allowed nickname characters: alphanumeric, underscore, hyphen
var validNickRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{2,24}$`)

// Regex for allowed room names: alphanumeric, underscore, hyphen
var validRoomRegex = regexp.MustCompile(`^#?[a-zA-Z0-9_-]{2,32}$`)

const (
	MaxMessageLength = 2000
	MaxNickLength    = 24
	MinNickLength    = 2
	MaxRoomLength    = 32
)

// SanitizeText removes ANSI escape sequences, control characters, and normalizes space.
// This is critical for terminal safety to prevent ANSI terminal injection exploits.
func SanitizeText(input string) string {
	// 1. Strip ANSI escape sequences
	clean := ansiRegex.ReplaceAllString(input, "")

	// 2. Remove dangerous control characters (preserve newline and tab)
	var b strings.Builder
	for _, r := range clean {
		if r == '\n' || r == '\t' || !unicode.IsControl(r) {
			b.WriteRune(r)
		}
	}
	res := strings.TrimSpace(b.String())

	// 3. Enforce maximum rune length
	runes := []rune(res)
	if len(runes) > MaxMessageLength {
		return string(runes[:MaxMessageLength])
	}
	return res
}

// ValidateNickname checks if a nickname meets the safety and format constraints.
func ValidateNickname(nick string) (string, error) {
	clean := strings.TrimSpace(ansiRegex.ReplaceAllString(nick, ""))
	if len(clean) < MinNickLength {
		return "", fmt.Errorf("nickname must be at least %d characters", MinNickLength)
	}
	if len(clean) > MaxNickLength {
		return "", fmt.Errorf("nickname must be at most %d characters", MaxNickLength)
	}
	if !validNickRegex.MatchString(clean) {
		return "", fmt.Errorf("nickname can only contain letters, numbers, hyphens, and underscores")
	}
	return clean, nil
}

// NormalizeRoomName ensures a room name starts with '#' and matches valid pattern.
func NormalizeRoomName(room string) (string, error) {
	clean := strings.TrimSpace(ansiRegex.ReplaceAllString(room, ""))
	if !strings.HasPrefix(clean, "#") {
		clean = "#" + clean
	}
	if len(clean) < 3 || len(clean) > MaxRoomLength {
		return "", fmt.Errorf("room name must be between 2 and %d characters", MaxRoomLength)
	}
	if !validRoomRegex.MatchString(clean) {
		return "", fmt.Errorf("room name contains invalid characters")
	}
	return strings.ToLower(clean), nil
}

var randomAdjectives = []string{
	"ghost", "silent", "neon", "crypto", "shadow", "cyber", "cosmic",
	"quantum", "astral", "nova", "solar", "lunar", "vortex", "echo",
	"hyper", "zero", "turbo", "pulse", "static", "flux", "swift",
}

var randomNouns = []string{
	"rider", "surfer", "node", "pixel", "spark", "runner", "wave",
	"operator", "phantom", "hacker", "drifter", "seeker", "glider",
	"sentinel", "beacon", "voyager", "nomad", "nexus", "orbit",
}

// GenerateAnonymousNick generates a safe, creative, anonymous nickname.
func GenerateAnonymousNick() string {
	adjIdx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(randomAdjectives))))
	nounIdx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(randomNouns))))
	num, _ := rand.Int(rand.Reader, big.NewInt(900))
	return fmt.Sprintf("%s-%s-%d", randomAdjectives[adjIdx.Int64()], randomNouns[nounIdx.Int64()], num.Int64()+100)
}
