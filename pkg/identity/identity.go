package identity

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/skywave-chat/skywave/pkg/protocol"
)

// Identity is a fully anonymous local account.
// No email, no phone, no password. Just a stable random AccountID
// plus a human-readable nickname the user can change anytime.
type Identity struct {
	AccountID string `json:"account_id"`
	Nickname  string `json:"nickname"`
	CreatedAt int64  `json:"created_at"`
}

// Dir returns the local config dir (~/.skywave).
func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".skywave"), nil
}

// Path returns the identity file path.
func Path() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "identity.json"), nil
}

// Load reads the saved identity, or returns (nil, nil) if none exists.
func Load() (*Identity, error) {
	p, err := Path()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var id Identity
	if err := json.Unmarshal(data, &id); err != nil {
		return nil, fmt.Errorf("corrupt identity file: %w", err)
	}
	if id.AccountID == "" {
		return nil, nil
	}
	return &id, nil
}

// Save persists the identity to disk with 0600 permissions.
func (id *Identity) Save() error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	p, err := Path()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(id, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0600)
}

// Create generates a brand-new anonymous account and saves it.
func Create(nickname string) (*Identity, error) {
	if nickname == "" {
		nickname = protocol.GenerateAnonymousNick()
	}
	clean, err := protocol.ValidateNickname(nickname)
	if err != nil {
		clean = protocol.GenerateAnonymousNick()
	}
	id := &Identity{
		AccountID: "anon-" + uuid.New().String()[:8],
		Nickname:  clean,
		CreatedAt: time.Now().Unix(),
	}
	if err := id.Save(); err != nil {
		return nil, err
	}
	return id, nil
}

// IsGeneratedNick checks if a nickname matches the auto-generated format (e.g. silent-glider-104).
func IsGeneratedNick(nick string) bool {
	parts := strings.Split(nick, "-")
	if len(parts) == 3 {
		if _, err := strconv.Atoi(parts[2]); err == nil {
			return true
		}
	}
	return false
}

// LoadOrCreate returns the saved account, or creates one on first run.
// If preferredNick is non-empty and valid, it overrides the saved nickname.
func LoadOrCreate(preferredNick string) (*Identity, bool, error) {
	existing, err := Load()
	if err != nil {
		return nil, false, err
	}
	if existing == nil {
		id, err := Create(preferredNick)
		if err != nil {
			return nil, false, err
		}
		return id, true, nil
	}
	// If the existing nickname is a generated placeholder, clear it so user sets their own name
	if IsGeneratedNick(existing.Nickname) {
		existing.Nickname = ""
	}
	// Explicit -nick flag wins and is persisted.
	if preferredNick != "" && preferredNick != existing.Nickname {
		if clean, err := protocol.ValidateNickname(preferredNick); err == nil {
			existing.Nickname = clean
			_ = existing.Save()
		}
	}
	return existing, false, nil
}

// SetNickname validates, applies and persists a nickname change.
func (id *Identity) SetNickname(nick string) error {
	clean, err := protocol.ValidateNickname(nick)
	if err != nil {
		return err
	}
	id.Nickname = clean
	return id.Save()
}

// Reset deletes the local account file.
func Reset() error {
	p, err := Path()
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
