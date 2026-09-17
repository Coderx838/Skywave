package ui

import "testing"

func TestExpandEmojis(t *testing.T) {
	input := "Hello :wave: world! This chat is on :fire: and :rocket:"
	expected := "Hello 👋 world! This chat is on 🔥 and 🚀"
	got := ExpandEmojis(input)
	if got != expected {
		t.Fatalf("ExpandEmojis failed: expected %q, got %q", expected, got)
	}

	noEmoji := "Just a normal message without codes"
	if ExpandEmojis(noEmoji) != noEmoji {
		t.Fatalf("Plain text should remain unchanged")
	}
}
