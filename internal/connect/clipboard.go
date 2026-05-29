package connect

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// nativeTools lists clipboard utilities in priority order. The first one found
// in PATH is used for "native" mode (and as the preferred path for "auto").
var nativeTools = [][]string{
	{"pbcopy"},                           // macOS
	{"wl-copy"},                          // Wayland
	{"xclip", "-selection", "clipboard"}, // X11
	{"xsel", "--clipboard", "--input"},   // X11
	{"clip.exe"},                         // WSL / Windows
}

// CopyPassword copies pw to the clipboard according to mode. It is best-effort:
// an empty password, mode "off", or an unavailable backend results in a no-op
// (or an error the caller is free to ignore). An empty mode is treated as "auto".
func CopyPassword(pw, mode string) error {
	if pw == "" {
		return nil
	}
	switch mode {
	case "off":
		return nil
	case "native":
		return copyNative(pw)
	case "osc52":
		return copyOSC52(pw)
	case "", "auto":
		if copyNative(pw) == nil {
			return nil
		}
		return copyOSC52(pw)
	default:
		return fmt.Errorf("unknown clipboard mode %q", mode)
	}
}

// copyNative pipes pw to the first available clipboard utility's stdin.
func copyNative(pw string) error {
	for _, tool := range nativeTools {
		path, err := exec.LookPath(tool[0])
		if err != nil {
			continue
		}
		cmd := exec.Command(path, tool[1:]...)
		cmd.Stdin = strings.NewReader(pw)
		cmd.Stdout = os.Stdout // needed for wrapper scripts that emit OSC52 to stdout
		if err := cmd.Run(); err == nil {
			return nil
		}
	}
	return fmt.Errorf("no usable native clipboard utility found")
}

// copyOSC52 writes an OSC52 clipboard escape sequence to stdout. When running
// inside tmux it is wrapped in a passthrough sequence so tmux forwards it to the
// outer terminal. Safe to call only after the alt-screen TUI has exited.
func copyOSC52(pw string) error {
	seq := osc52Payload(pw)
	if os.Getenv("TMUX") != "" {
		// tmux DCS passthrough: \ePtmux;<seq-with-doubled-ESCs>\e\\
		seq = "\x1bPtmux;" + strings.ReplaceAll(seq, "\x1b", "\x1b\x1b") + "\x1b\\"
	}
	_, err := fmt.Fprint(os.Stdout, seq)
	return err
}

// osc52Payload builds the raw OSC52 set-clipboard escape sequence for pw.
func osc52Payload(pw string) string {
	enc := base64.StdEncoding.EncodeToString([]byte(pw))
	return "\x1b]52;c;" + enc + "\x07"
}
