package connect

import (
	"fmt"

	"github.com/ripnet/shellodex/internal/model"
)

const maxJumpDepth = 8

// ResolveJumps walks the jump chain starting from host and returns a slice of
// "-J user@host:port" hop strings, outermost first.
// Returns an error if the chain exceeds maxJumpDepth or contains a cycle.
func ResolveJumps(host *model.Host, cfg *model.Config) ([]string, error) {
	var hops []string
	visited := make(map[string]bool)

	current := host
	for current.JumpHostID != "" {
		if visited[current.JumpHostID] {
			return nil, fmt.Errorf("jump chain cycle detected at host %q", current.JumpHostID)
		}
		if len(hops) >= maxJumpDepth {
			return nil, fmt.Errorf("jump chain exceeds maximum depth of %d", maxJumpDepth)
		}
		visited[current.JumpHostID] = true

		jump := cfg.HostByID(current.JumpHostID)
		if jump == nil {
			return nil, fmt.Errorf("jump host %q not found", current.JumpHostID)
		}

		hop := jumpHop(jump, cfg)
		hops = append([]string{hop}, hops...) // prepend so outermost is first
		current = jump
	}
	return hops, nil
}

func jumpHop(h *model.Host, cfg *model.Config) string {
	port := h.Port
	if port == 0 {
		port = int(model.DefaultPort(h.Protocol))
	}
	cred := cfg.CredentialByID(h.CredentialID)
	if cred != nil && cred.Username != "" {
		return fmt.Sprintf("%s@%s:%d", cred.Username, h.Hostname, port)
	}
	return fmt.Sprintf("%s:%d", h.Hostname, port)
}
