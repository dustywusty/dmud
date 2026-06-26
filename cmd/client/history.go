package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// maxHistory caps how many recent commands are kept on disk.
const maxHistory = 1000

func historyPath() string {
	dir := configDir()
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, "history")
}

// loadHistory reads saved commands (most recent last) and trims the on-disk file
// to the most recent maxHistory lines.
func loadHistory() []string {
	path := historyPath()
	if path == "" {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	var lines []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() {
		if line := sc.Text(); strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}
	_ = f.Close()

	if len(lines) > maxHistory {
		lines = lines[len(lines)-maxHistory:]
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err == nil {
			_ = os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644)
		}
	}
	return lines
}

// persistableCommand reports whether a command should be written to the history
// file. Login lines carry a secret token, so they stay in-session only.
func persistableCommand(raw string) bool {
	return !strings.HasPrefix(strings.ToLower(strings.TrimSpace(raw)), "login ")
}

// appendHistoryCmd appends one command to the history file off the UI goroutine.
func appendHistoryCmd(cmd string) tea.Cmd {
	if !persistableCommand(cmd) {
		return nil
	}
	return func() tea.Msg {
		path := historyPath()
		if path == "" {
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil
		}
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return nil
		}
		_, _ = f.WriteString(cmd + "\n")
		_ = f.Close()
		return nil
	}
}
