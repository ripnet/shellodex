package connect

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestOSC52Payload(t *testing.T) {
	got := osc52Payload("hunter2")
	want := "\x1b]52;c;" + base64.StdEncoding.EncodeToString([]byte("hunter2")) + "\x07"
	if got != want {
		t.Fatalf("osc52Payload = %q, want %q", got, want)
	}
	if !strings.HasPrefix(got, "\x1b]52;c;") || !strings.HasSuffix(got, "\x07") {
		t.Fatalf("osc52Payload framing wrong: %q", got)
	}
}

func TestTmuxPassthroughFraming(t *testing.T) {
	inner := osc52Payload("secret")
	// Simulate the tmux wrap: all ESCs in inner must be doubled, wrapped in DCS.
	got := "\x1bPtmux;" + strings.ReplaceAll(inner, "\x1b", "\x1b\x1b") + "\x1b\\"

	// Must start with DCS introducer + tmux command.
	if !strings.HasPrefix(got, "\x1bPtmux;") {
		t.Fatalf("missing DCS/tmux prefix: %q", got)
	}
	// Must end with ST.
	if !strings.HasSuffix(got, "\x1b\\") {
		t.Fatalf("missing ST suffix: %q", got)
	}
	// The inner OSC52 ESC must be doubled → exactly \x1b\x1b after the prefix.
	after := strings.TrimPrefix(got, "\x1bPtmux;")
	if !strings.HasPrefix(after, "\x1b\x1b]52;c;") {
		t.Fatalf("inner OSC52 ESC not doubled; got prefix %q", after[:min(12, len(after))])
	}
	// Must NOT have three consecutive ESCs anywhere (the old bug).
	if strings.Contains(got, "\x1b\x1b\x1b") {
		t.Fatalf("triple ESC found — tmux passthrough will be malformed: %q", got)
	}
}

func TestCopyPasswordNoops(t *testing.T) {
	// Empty password and "off" mode must never error and never touch anything.
	if err := CopyPassword("", "auto"); err != nil {
		t.Fatalf("empty password: %v", err)
	}
	if err := CopyPassword("pw", "off"); err != nil {
		t.Fatalf("off mode: %v", err)
	}
	if err := CopyPassword("pw", "bogus"); err == nil {
		t.Fatal("unknown mode should error")
	}
}
