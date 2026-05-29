package model

import (
	"time"

	"github.com/google/uuid"
)

type Protocol string

const (
	SSH    Protocol = "ssh"
	Telnet Protocol = "telnet"
)

type SortField string

const (
	SortByName     SortField = "name"
	SortByHostname SortField = "hostname"
	SortByLastConn SortField = "last_connected"
)

type Credential struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Username    string `json:"username"`
	Password    string `json:"password,omitempty"`
	KeyPath     string `json:"key_path,omitempty"`
	PasswordEnc string `json:"password_enc,omitempty"` // reserved for future encryption
}

type Host struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	GroupID      string   `json:"group_id,omitempty"`
	Protocol     Protocol `json:"protocol"`
	Hostname     string   `json:"hostname"`
	Port         int      `json:"port"`
	CredentialID string   `json:"credential_id,omitempty"`
	// Inline credential for one-off hosts (used only when CredentialID is empty).
	Username   string `json:"username,omitempty"`
	Password   string `json:"password,omitempty"`
	KeyPath    string `json:"key_path,omitempty"`
	JumpHostID    string     `json:"jump_host_id,omitempty"`
	Notes         string     `json:"notes,omitempty"`
	Tags          []string   `json:"tags,omitempty"`
	LastConnected *time.Time `json:"last_connected,omitempty"`
}

type Group struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	ParentID string `json:"parent_id,omitempty"`
}

type SyncConfig struct {
	Remote    string `json:"remote"`
	Direction string `json:"direction"` // "push" | "pull" | "sync"
}

type Config struct {
	Warning       string       `json:"_warning,omitempty"`
	Version       int          `json:"version"`
	Credentials   []Credential `json:"credentials"`
	Groups        []Group      `json:"groups"`
	Hosts         []Host       `json:"hosts"`
	Sync          SyncConfig   `json:"sync,omitempty"`
	ClipboardMode string       `json:"clipboard_mode,omitempty"` // "off"|"auto"|"osc52"|"native" (empty = auto)
	DefaultSort   SortField    `json:"default_sort,omitempty"`
}

func NewID() string {
	return uuid.New().String()
}

func DefaultPort(p Protocol) int {
	switch p {
	case Telnet:
		return 23
	default:
		return 22
	}
}

// GroupName returns the name of the group with the given ID, or empty string.
func (c *Config) GroupName(id string) string {
	for _, g := range c.Groups {
		if g.ID == id {
			return g.Name
		}
	}
	return ""
}

// EffectiveCredential returns the credential to use for a host: the linked
// shared credential if set, otherwise an ad-hoc credential built from the
// host's inline username/password/key, or nil if neither is set.
func (c *Config) EffectiveCredential(h *Host) *Credential {
	if h.CredentialID != "" {
		return c.CredentialByID(h.CredentialID)
	}
	if h.Username != "" || h.Password != "" || h.KeyPath != "" {
		return &Credential{
			Username: h.Username,
			Password: h.Password,
			KeyPath:  h.KeyPath,
		}
	}
	return nil
}

// CredentialByID returns the credential with the given ID, or nil.
func (c *Config) CredentialByID(id string) *Credential {
	for i := range c.Credentials {
		if c.Credentials[i].ID == id {
			return &c.Credentials[i]
		}
	}
	return nil
}

// HostByID returns the host with the given ID, or nil.
func (c *Config) HostByID(id string) *Host {
	for i := range c.Hosts {
		if c.Hosts[i].ID == id {
			return &c.Hosts[i]
		}
	}
	return nil
}

// GroupByID returns the group with the given ID, or nil.
func (c *Config) GroupByID(id string) *Group {
	for i := range c.Groups {
		if c.Groups[i].ID == id {
			return &c.Groups[i]
		}
	}
	return nil
}

// GroupPath returns the full breadcrumb path for a group ID (e.g. "Lab / Core").
func (c *Config) GroupPath(groupID string) string {
	if groupID == "" {
		return ""
	}
	var segments []string
	id := groupID
	visited := make(map[string]bool)
	for id != "" && !visited[id] {
		visited[id] = true
		for _, g := range c.Groups {
			if g.ID == id {
				segments = append([]string{g.Name}, segments...)
				id = g.ParentID
				break
			}
		}
	}
	result := ""
	for i, s := range segments {
		if i > 0 {
			result += " / "
		}
		result += s
	}
	return result
}
