package server

import (
	"testing"
	"time"
)

func TestRateLimiter(t *testing.T) {
	// Rate: 2 tokens/sec, capacity: 3 tokens
	rl := NewRateLimiter(2.0, 3.0)

	// First 3 calls should immediately pass
	if !rl.Allow() || !rl.Allow() || !rl.Allow() {
		t.Fatalf("Expected first 3 burst tokens to be allowed")
	}

	// 4th immediate call should be blocked
	if rl.Allow() {
		t.Fatalf("Expected 4th call to be rate limited")
	}

	// Wait 600ms (should regenerate > 1 token)
	time.Sleep(600 * time.Millisecond)
	if !rl.Allow() {
		t.Fatalf("Expected refilled token after waiting to be allowed")
	}
}

func TestRoomAuthentication(t *testing.T) {
	pubRoom := NewRoom("#general", "General Chat", "", 50)
	if !pubRoom.Authenticate("") || !pubRoom.Authenticate("anypassword") {
		t.Fatalf("Public room should always authenticate successfully")
	}

	privRoom := NewRoom("#secret", "Secret Room", "mypass123", 50)
	if !privRoom.IsPrivate {
		t.Fatalf("Room with password should be marked private")
	}

	if privRoom.Authenticate("wrongpass") {
		t.Fatalf("Private room should reject incorrect password")
	}

	if !privRoom.Authenticate("mypass123") {
		t.Fatalf("Private room should accept correct password")
	}
}
