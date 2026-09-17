package identity

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// ServerEntry holds bookmark data for a Skywave server.
type ServerEntry struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Password string `json:"password,omitempty"`
	IsCustom bool   `json:"is_custom"`
}

// ServerList manages user's bookmarked servers.
type ServerList struct {
	ActiveURL string        `json:"active_url"`
	Servers   []ServerEntry `json:"servers"`
}

// ServersPath returns ~/.skywave/servers.json
func ServersPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "servers.json"), nil
}

// DefaultServers provides starter default hubs.
func DefaultServers() ServerList {
	return ServerList{
		ActiveURL: "ws://localhost:8080/ws",
		Servers: []ServerEntry{
			{
				Name:     "🌊 Skywave Official Hub",
				URL:      "wss://hub.skywave.chat/ws",
				IsCustom: false,
			},
			{
				Name:     "🏠 Localhost Dev Node",
				URL:      "ws://localhost:8080/ws",
				IsCustom: false,
			},
		},
	}
}

// LoadServers loads saved servers from ~/.skywave/servers.json or initializes defaults.
func LoadServers() (*ServerList, error) {
	p, err := ServersPath()
	if err != nil {
		sl := DefaultServers()
		return &sl, nil
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			sl := DefaultServers()
			_ = sl.Save()
			return &sl, nil
		}
		return nil, err
	}

	var sl ServerList
	if err := json.Unmarshal(data, &sl); err != nil || len(sl.Servers) == 0 {
		d := DefaultServers()
		return &d, nil
	}
	return &sl, nil
}

// Save persists the server list.
func (sl *ServerList) Save() error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	p, err := ServersPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(sl, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0600)
}

// AddServer appends a new server to the list.
func (sl *ServerList) AddServer(name, url, password string) error {
	sl.Servers = append(sl.Servers, ServerEntry{
		Name:     name,
		URL:      url,
		Password: password,
		IsCustom: true,
	})
	return sl.Save()
}
