package main

import (
	"encoding/json"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
)

// Saved login ids, keyed by server address, so the client can replay
// `login <id>` on connect and resume the same character. Stored in
// $XDG_CONFIG_HOME/dmud-client/identity.json.

func identityPath() string {
	dir := configDir()
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, "identity.json")
}

func loadIdentities() map[string]string {
	ids := map[string]string{}
	path := identityPath()
	if path == "" {
		return ids
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ids
	}
	_ = json.Unmarshal(data, &ids)
	if ids == nil {
		ids = map[string]string{}
	}
	return ids
}

// loadIdentity returns the saved login id for a server address, or "".
func loadIdentity(addr string) string { return loadIdentities()[addr] }

func saveIdentity(addr, id string) error {
	path := identityPath()
	if path == "" || addr == "" || id == "" {
		return nil
	}
	ids := loadIdentities()
	if ids[addr] == id {
		return nil // unchanged
	}
	ids[addr] = id
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(ids, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func saveIdentityCmd(addr, id string) tea.Cmd {
	return func() tea.Msg {
		_ = saveIdentity(addr, id)
		return nil
	}
}
