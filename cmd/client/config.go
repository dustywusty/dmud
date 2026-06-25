package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// clientConfig is the persisted client configuration:
// $XDG_CONFIG_HOME/dmud-client/config.json (falling back to ~/.config).
type clientConfig struct {
	Layout     layoutPrefs       `json:"layout"`
	Compact    bool              `json:"compact,omitempty"` // no blank line between messages
	Aliases    map[string]string `json:"aliases,omitempty"`
	Macros     map[string]macro  `json:"macros,omitempty"` // hotkey macros, keyed by slot
	Highlights []ruleConfig      `json:"highlights,omitempty"`
	Triggers   []ruleConfig      `json:"triggers,omitempty"`
}

// layoutPrefs is the persisted UI layout.
type layoutPrefs struct {
	LeftWidth  int  `json:"left_width"`
	RightWidth int  `json:"right_width"`
	MapHeight  int  `json:"map_height"`
	ShowRight  bool `json:"show_right"`
}

// ruleConfig is a highlight (Pattern → Value is a color) or a trigger
// (Pattern → Value is a command to send).
type ruleConfig struct {
	Pattern string `json:"pattern"`
	Value   string `json:"value"`
}

func defaultPrefs() layoutPrefs {
	return layoutPrefs{LeftWidth: 26, RightWidth: 44, MapHeight: 13, ShowRight: true}
}

func defaultConfig() clientConfig {
	return clientConfig{Layout: defaultPrefs()}
}

func configDir() string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "dmud-client")
}

func configPath() string {
	dir := configDir()
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, "config.json")
}

// loadConfig reads the saved config, returning defaults for anything missing or
// invalid so a corrupt file never breaks startup.
func loadConfig() clientConfig {
	c := defaultConfig()
	path := configPath()
	if path == "" {
		return c
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return c
	}
	var loaded clientConfig
	if err := json.Unmarshal(data, &loaded); err != nil {
		return c
	}
	if loaded.Layout.LeftWidth > 0 {
		c.Layout.LeftWidth = loaded.Layout.LeftWidth
	}
	if loaded.Layout.RightWidth > 0 {
		c.Layout.RightWidth = loaded.Layout.RightWidth
	}
	if loaded.Layout.MapHeight >= 0 {
		c.Layout.MapHeight = loaded.Layout.MapHeight
	}
	c.Layout.ShowRight = loaded.Layout.ShowRight
	c.Compact = loaded.Compact
	c.Aliases = loaded.Aliases
	c.Macros = loaded.Macros
	c.Highlights = loaded.Highlights
	c.Triggers = loaded.Triggers
	return c
}

// saveConfig writes the config atomically (temp file + rename).
func saveConfig(c clientConfig) error {
	path := configPath()
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
