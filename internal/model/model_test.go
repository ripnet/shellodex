package model

import "testing"

func TestEffectiveCredential(t *testing.T) {
	cfg := &Config{
		Credentials: []Credential{
			{ID: "c1", Name: "shared", Username: "admin", Password: "shp"},
		},
	}

	t.Run("linked credential wins", func(t *testing.T) {
		h := &Host{CredentialID: "c1", Username: "inline", Password: "inp"}
		got := cfg.EffectiveCredential(h)
		if got == nil || got.Username != "admin" || got.Password != "shp" {
			t.Fatalf("got %+v, want shared credential", got)
		}
	})

	t.Run("inline when no link", func(t *testing.T) {
		h := &Host{Username: "root", Password: "secret", KeyPath: "/k"}
		got := cfg.EffectiveCredential(h)
		if got == nil || got.Username != "root" || got.Password != "secret" || got.KeyPath != "/k" {
			t.Fatalf("got %+v, want inline credential", got)
		}
	})

	t.Run("nil when neither", func(t *testing.T) {
		if got := cfg.EffectiveCredential(&Host{}); got != nil {
			t.Fatalf("got %+v, want nil", got)
		}
	})

	t.Run("linked but missing id returns nil", func(t *testing.T) {
		if got := cfg.EffectiveCredential(&Host{CredentialID: "nope"}); got != nil {
			t.Fatalf("got %+v, want nil for unknown credential id", got)
		}
	})
}
