package components

import (
	"sync"
)

// CommandHistory tracks command history for a player
type CommandHistory struct {
	History    []string
	Position   int // Current position in history (0 = most recent)
	MaxHistory int
	mu         sync.RWMutex
}

// NewCommandHistory creates a new command history with default settings
func NewCommandHistory() *CommandHistory {
	return &CommandHistory{
		History:    make([]string, 0),
		Position:   0,
		MaxHistory: 100, // Keep last 100 commands
	}
}

// AddCommand adds a new command to the history
func (ch *CommandHistory) AddCommand(cmd string) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	// Don't add empty commands or duplicate consecutive commands
	if cmd == "" {
		return
	}

	if len(ch.History) > 0 && ch.History[0] == cmd {
		return // Don't add duplicate consecutive commands
	}

	// Add to front of history
	ch.History = append([]string{cmd}, ch.History...)
	ch.Position = 0

	// Trim history if it gets too long
	if len(ch.History) > ch.MaxHistory {
		ch.History = ch.History[:ch.MaxHistory]
	}
}

// GetPrevious returns the previous command in history
func (ch *CommandHistory) GetPrevious() string {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	if ch.Position >= len(ch.History)-1 {
		return ""
	}

	ch.Position++
	return ch.History[ch.Position]
}

// GetNext returns the next command in history
func (ch *CommandHistory) GetNext() string {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	if ch.Position <= 0 {
		ch.Position = 0
		return ""
	}

	ch.Position--
	return ch.History[ch.Position]
}

// ResetPosition resets the history position to the beginning
func (ch *CommandHistory) ResetPosition() {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	ch.Position = 0
}

// GetHistory returns a copy of the command history
func (ch *CommandHistory) GetHistory() []string {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	history := make([]string, len(ch.History))
	copy(history, ch.History)
	return history
}

// ClearHistory clears the command history
func (ch *CommandHistory) ClearHistory() {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	ch.History = make([]string, 0)
	ch.Position = 0
} 