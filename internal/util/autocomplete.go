package util

import (
	"sort"
	"strings"
)

// AutoComplete provides auto-complete functionality for commands and player names
type AutoComplete struct {
	commands map[string]bool
	players  map[string]bool
}

// NewAutoComplete creates a new auto-complete instance
func NewAutoComplete() *AutoComplete {
	return &AutoComplete{
		commands: make(map[string]bool),
		players:  make(map[string]bool),
	}
}

// AddCommand adds a command to the auto-complete list
func (ac *AutoComplete) AddCommand(cmd string) {
	ac.commands[strings.ToLower(cmd)] = true
}

// AddPlayer adds a player name to the auto-complete list
func (ac *AutoComplete) AddPlayer(name string) {
	ac.players[strings.ToLower(name)] = true
}

// RemovePlayer removes a player name from the auto-complete list
func (ac *AutoComplete) RemovePlayer(name string) {
	delete(ac.players, strings.ToLower(name))
}

// GetCommandSuggestions returns command suggestions based on partial input
func (ac *AutoComplete) GetCommandSuggestions(partial string) []string {
	if partial == "" {
		return []string{}
	}

	partial = strings.ToLower(partial)
	var suggestions []string

	for cmd := range ac.commands {
		if strings.HasPrefix(cmd, partial) {
			suggestions = append(suggestions, cmd)
		}
	}

	sort.Strings(suggestions)
	return suggestions
}

// GetPlayerSuggestions returns player name suggestions based on partial input
func (ac *AutoComplete) GetPlayerSuggestions(partial string) []string {
	if partial == "" {
		return []string{}
	}

	partial = strings.ToLower(partial)
	var suggestions []string

	for player := range ac.players {
		if strings.HasPrefix(player, partial) {
			suggestions = append(suggestions, player)
		}
	}

	sort.Strings(suggestions)
	return suggestions
}

// GetBestMatch returns the best match for a partial input
// Returns the input if no match is found
func (ac *AutoComplete) GetBestMatch(partial string) string {
	if partial == "" {
		return ""
	}

	partial = strings.ToLower(partial)

	// Check commands first
	for cmd := range ac.commands {
		if strings.HasPrefix(cmd, partial) {
			return cmd
		}
	}

	// Check players
	for player := range ac.players {
		if strings.HasPrefix(player, partial) {
			return player
		}
	}

	return partial
}

// GetMultipleMatches returns all matches for a partial input
func (ac *AutoComplete) GetMultipleMatches(partial string) []string {
	if partial == "" {
		return []string{}
	}

	partial = strings.ToLower(partial)
	var matches []string

	// Check commands
	for cmd := range ac.commands {
		if strings.HasPrefix(cmd, partial) {
			matches = append(matches, cmd)
		}
	}

	// Check players
	for player := range ac.players {
		if strings.HasPrefix(player, partial) {
			matches = append(matches, player)
		}
	}

	sort.Strings(matches)
	return matches
} 