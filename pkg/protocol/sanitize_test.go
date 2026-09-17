package protocol

import (
	"testing"
)

func TestSanitizeText(t *testing.T) {
	// Test ANSI escape sequence injection stripping
	maliciousInput := "\x1b[31;1mHacked text\x1b[0m with \x1b]0;Evil Title\x07 payload"
	expected := "Hacked text with  payload"
	got := SanitizeText(maliciousInput)
	if got != expected {
		t.Fatalf("SanitizeText failed. Expected %q, got %q", expected, got)
	}

	// Test control character stripping
	ctrlInput := "Safe text\x00\x07\x1b"
	expectedCtrl := "Safe text"
	gotCtrl := SanitizeText(ctrlInput)
	if gotCtrl != expectedCtrl {
		t.Fatalf("Control char stripping failed. Expected %q, got %q", expectedCtrl, gotCtrl)
	}
}

func TestValidateNickname(t *testing.T) {
	// Valid nicknames
	validNicks := []string{"neo", "alice_99", "cyber-wave", "ghost_runner"}
	for _, n := range validNicks {
		if _, err := ValidateNickname(n); err != nil {
			t.Errorf("Valid nickname %q was rejected: %v", n, err)
		}
	}

	// Invalid nicknames
	invalidNicks := []string{"a", "this_nickname_is_way_too_long_and_should_fail_validation", "bad nick", "user!@#"}
	for _, n := range invalidNicks {
		if _, err := ValidateNickname(n); err == nil {
			t.Errorf("Invalid nickname %q should have been rejected", n)
		}
	}
}

func TestNormalizeRoomName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		wantErr  bool
	}{
		{"general", "#general", false},
		{"#tech", "#tech", false},
		{"#Cool-Room_9", "#cool-room_9", false},
		{"a", "", true},
		{"#invalid space", "", true},
	}

	for _, tc := range tests {
		got, err := NormalizeRoomName(tc.input)
		if tc.wantErr && err == nil {
			t.Errorf("Expected error for %q, got nil", tc.input)
		}
		if !tc.wantErr && (err != nil || got != tc.expected) {
			t.Errorf("For %q expected %q, got %q (err: %v)", tc.input, tc.expected, got, err)
		}
	}
}
