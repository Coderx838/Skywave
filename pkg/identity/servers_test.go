package identity

import "testing"

func TestDefaultServers(t *testing.T) {
	d := DefaultServers()
	if len(d.Servers) == 0 {
		t.Fatalf("Expected default servers to be non-empty")
	}

	foundOfficial := false
	for _, s := range d.Servers {
		if s.URL == "wss://hub.skywave.chat/ws" {
			foundOfficial = true
			break
		}
	}
	if !foundOfficial {
		t.Fatalf("Official hub should be in default servers")
	}
}
