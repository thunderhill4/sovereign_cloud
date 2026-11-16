package handlers

import (
	"testing"
	"time"
)

func TestAIChatTimeout(t *testing.T) {
	cases := map[string]time.Duration{
		"":    defaultAIChatTimeout,
		"10m": 10 * time.Minute,
		"90s": 90 * time.Second,
		"abc": defaultAIChatTimeout,
		"0":   defaultAIChatTimeout,
		"-1m": defaultAIChatTimeout,
		"5s":  defaultAIChatTimeout, // below the floor: too short for any local model
		"2h":  defaultAIChatTimeout, // above the cap
		"30m": 30 * time.Minute,
	}
	for in, want := range cases {
		t.Setenv("AI_CHAT_TIMEOUT", in)
		if got := aiChatTimeout(); got != want {
			t.Errorf("AI_CHAT_TIMEOUT=%q -> %v, want %v", in, got, want)
		}
	}
}
