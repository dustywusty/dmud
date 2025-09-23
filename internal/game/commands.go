package game

import (
	"fmt"
	"strings"

	"dmud/internal/components"
)

// handleHistory shows the player's command history
func handleHistory(player *components.Player, args []string, game *Game) {
	history := player.CommandHistory.GetHistory()

	if len(history) == 0 {
		player.Broadcast("No command history.")
		return
	}

	player.Broadcast("Command History:")
	for i, cmd := range history {
		player.Broadcast(fmt.Sprintf("%d: %s", i+1, cmd))
	}
}

// handleClear clears the player's command history
func handleClear(player *components.Player, args []string, game *Game) {
	player.CommandHistory.ClearHistory()
	player.Broadcast("Command history cleared.")
}

// handleSuggest provides auto-complete suggestions
func handleSuggest(player *components.Player, args []string, game *Game) {
	if len(args) == 0 {
		player.Broadcast("Usage: suggest <partial_command_or_name>")
		return
	}

	partial := strings.Join(args, " ")

	// Get command suggestions
	cmdSuggestions := player.AutoComplete.GetCommandSuggestions(partial)

	// Get player suggestions
	playerSuggestions := player.AutoComplete.GetPlayerSuggestions(partial)

	if len(cmdSuggestions) == 0 && len(playerSuggestions) == 0 {
		player.Broadcast(fmt.Sprintf("No suggestions found for '%s'", partial))
		return
	}

	player.Broadcast(fmt.Sprintf("Suggestions for '%s':", partial))

	if len(cmdSuggestions) > 0 {
		player.Broadcast("Commands: " + strings.Join(cmdSuggestions, ", "))
	}

	if len(playerSuggestions) > 0 {
		player.Broadcast("Players: " + strings.Join(playerSuggestions, ", "))
	}
}

// handleComplete provides instant auto-completion (best match only)
func handleComplete(player *components.Player, args []string, game *Game) {
	if len(args) == 0 {
		player.Broadcast("Usage: complete <partial_command_or_name>")
		return
	}

	partial := strings.Join(args, " ")

	// Get best match
	bestMatch := player.AutoComplete.GetBestMatch(partial)

	if bestMatch == partial {
		player.Broadcast(fmt.Sprintf("No completion found for '%s'", partial))
		return
	}

	player.Broadcast(fmt.Sprintf("Completion: %s -> %s", partial, bestMatch))
}

// handleHelp shows all available commands with descriptions
func handleHelp(player *components.Player, args []string, game *Game) {
	if len(args) == 0 {
		var b strings.Builder
		b.WriteString("Basic Commands:\n")
		b.WriteString("  look          - Look around your current location\n")
		b.WriteString("  who            - List online players\n")
		b.WriteString("  say <message>  - Say something to players in the same area\n")
		b.WriteString("  shout <message> - Shout to nearby areas\n")
		b.WriteString("  examine <target> - Examine something or someone\n")
		b.WriteString("  kill <target>  - Attack a player or NPC\n")
		b.WriteString("  exit           - Leave the game\n")
		b.WriteString("\n")
		b.WriteString("Movement Commands:\n")
		b.WriteString("  north/n        - Move north\n")
		b.WriteString("  south/s        - Move south\n")
		b.WriteString("  east/e         - Move east\n")
		b.WriteString("  west/w         - Move west\n")
		b.WriteString("  up/u           - Move up\n")
		b.WriteString("  down/d         - Move down\n")

		b.WriteString("\n")
		b.WriteString("Player Management:\n")
		b.WriteString("  name <new_name> - Change your player name\n")

		b.WriteString("\n")
		b.WriteString("Utility Commands:\n")
		b.WriteString("  help           - Show this help message\n")
		b.WriteString("  history/hist   - Show your command history\n")
		b.WriteString("  clear          - Clear your command history\n")
		b.WriteString("  suggest <partial> - Get suggestions for commands or players\n")
		b.WriteString("  complete <partial> - Get instant auto-completion\n")

		b.WriteString("\n")
		b.WriteString("Type 'help <command>' for detailed information about a specific command.\n")

		player.Broadcast(b.String())

		return
	}

	// Show help for specific command
	commandName := args[0]

	// Define command help
	commandHelp := map[string]string{
		"look":     "Look around your current location to see the area description, exits, and other players/NPCs present.",
		"who":      "List all players currently online in the game.",
		"say":      "Say something to all players in the same area. Usage: say <message>",
		"shout":    "Shout a message that can be heard in nearby areas. Usage: shout <message>",
		"examine":  "Examine something or someone in detail. Usage: examine <target>",
		"kill":     "Attack another player or NPC. Usage: kill <target>",
		"exit":     "Leave the game and disconnect from the server.",
		"north":    "Move north to the adjacent area (if an exit exists).",
		"south":    "Move south to the adjacent area (if an exit exists).",
		"east":     "Move east to the adjacent area (if an exit exists).",
		"west":     "Move west to the adjacent area (if an exit exists).",
		"up":       "Move up to the adjacent area (if an exit exists).",
		"down":     "Move down to the adjacent area (if an exit exists).",
		"name":     "Change your player name. Usage: name <new_name>",
		"help":     "Show help information. Usage: help [command]",
		"history":  "Show your command history (last 100 commands).",
		"clear":    "Clear your command history.",
		"suggest":  "Get suggestions for commands or player names. Usage: suggest <partial>",
		"complete": "Get instant auto-completion for commands or player names. Usage: complete <partial>",
	}

	if help, exists := commandHelp[commandName]; exists {
		player.Broadcast(fmt.Sprintf("Help for '%s':", commandName))
		player.Broadcast("=" + strings.Repeat("=", len(commandName)+8))
		player.Broadcast(help)
	} else {
		player.Broadcast(fmt.Sprintf("No help available for command '%s'", commandName))
		player.Broadcast("Type 'help' to see all available commands.")
	}
}
