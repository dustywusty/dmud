package game

import (
	"fmt"
	"strings"

	"dmud/internal/components"
)

// commandHelpText provides detailed help for each command
var commandHelpText = map[string]string{
	"look":      "Look around your current location to see the area description, exits, and other players/NPCs present.",
	"who":       "List all players currently online in the game.",
	"say":       "Say something to all players in the same area. Usage: say <message>",
	"shout":     "Shout a message that can be heard in nearby areas. Usage: shout <message>",
	"examine":   "Examine something or someone in detail. Usage: examine <target>",
	"kill":      "Attack another player or NPC. Usage: kill <target> or kill all (to attack everything in the area)",
	"exit":      "Leave the game and disconnect from the server.",
	"north":     "Move north to the adjacent area (if an exit exists).",
	"south":     "Move south to the adjacent area (if an exit exists).",
	"east":      "Move east to the adjacent area (if an exit exists).",
	"west":      "Move west to the adjacent area (if an exit exists).",
	"up":        "Move up to the adjacent area (if an exit exists).",
	"down":      "Move down to the adjacent area (if an exit exists).",
	"name":      "Change your player name. Usage: name <new_name>",
	"recall":    "Return to the starting area instantly.",
	"help":      "Show help information. Usage: help [command]",
	"history":   "Show your command history (last 100 commands).",
	"clear":     "Clear your command history.",
	"suggest":   "Get suggestions for commands or player names. Usage: suggest <partial>",
	"complete":  "Get instant auto-completion for commands or player names. Usage: complete <partial>",
	"inventory": "View your inventory and see what items you are carrying. Usage: inventory (aliases: inv, i)",
	"loot":      "Loot items from a corpse. Usage: loot <corpse_name> or loot all (to loot all corpses in the area)",
	"get":       "Pick up an item from the ground. Usage: get <item_name> (aliases: pickup, take)",
	"drop":      "Drop an item from your inventory onto the ground. Usage: drop <item_name>",
}

// handleHistory shows the player's command history
func handleHistory(player *components.Player, args []string, game *Game) {
	history := player.CommandHistory.GetHistory()

	if len(history) == 0 {
		player.Broadcast("No command history.")
		return
	}

	var b strings.Builder
	b.WriteString("Command History:\n")
	for i, cmd := range history {
		b.WriteString(fmt.Sprintf("%d: %s\n", i+1, cmd))
	}
	player.Broadcast(b.String())
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

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Suggestions for '%s':\n", partial))

	if len(cmdSuggestions) > 0 {
		b.WriteString("Commands: " + strings.Join(cmdSuggestions, ", ") + "\n")
	}

	if len(playerSuggestions) > 0 {
		b.WriteString("Players: " + strings.Join(playerSuggestions, ", ") + "\n")
	}

	player.Broadcast(b.String())
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
		b.WriteString("==============================================\n")
		b.WriteString("               AVAILABLE COMMANDS             \n")
		b.WriteString("==============================================\n\n")

		b.WriteString("EXPLORATION\n")
		b.WriteString("  look              - View your current location\n")
		b.WriteString("  examine <target>  - Examine something closely\n\n")

		b.WriteString("COMMUNICATION\n")
		b.WriteString("  say <message>     - Speak to nearby players\n")
		b.WriteString("  shout <message>   - Shout to adjacent areas\n")
		b.WriteString("  who               - List online players\n\n")

		b.WriteString("COMBAT\n")
		b.WriteString("  kill <target>     - Attack a target\n")
		b.WriteString("  kill all          - Attack everything in the area\n\n")

		b.WriteString("MOVEMENT\n")
		b.WriteString("  north, south, east, west, up, down\n\n")

		b.WriteString("INVENTORY\n")
		b.WriteString("  inventory         - View your items (aliases: inv, i)\n")
		b.WriteString("  loot <corpse>     - Loot items from a corpse\n")
		b.WriteString("  loot all          - Loot all corpses in the area\n")
		b.WriteString("  get <item>        - Pick up an item (aliases: pickup, take)\n")
		b.WriteString("  drop <item>       - Drop an item\n\n")

		b.WriteString("CHARACTER\n")
		b.WriteString("  name <new_name>   - Change your name\n")
		b.WriteString("  recall            - Return to starting area\n\n")

		b.WriteString("UTILITY\n")
		b.WriteString("  help [command]    - Show help information\n")
		b.WriteString("  history           - View command history\n")
		b.WriteString("  clear             - Clear command history\n")
		b.WriteString("  suggest <partial> - Get command suggestions\n")
		b.WriteString("  complete <partial>- Auto-complete commands\n")
		b.WriteString("  exit              - Leave the game\n\n")

		b.WriteString("==============================================\n")
		b.WriteString("Type 'help <command>' for detailed information\n")
		b.WriteString("==============================================\n")

		player.Broadcast(b.String())

		return
	}

	// Show help for specific command
	commandName := args[0]

	if help, exists := commandHelpText[commandName]; exists {
		player.Broadcast(fmt.Sprintf("Help for '%s':", commandName))
		player.Broadcast("=" + strings.Repeat("=", len(commandName)+8))
		player.Broadcast(help)
	} else {
		player.Broadcast(fmt.Sprintf("No help available for command '%s'", commandName))
		player.Broadcast("Type 'help' to see all available commands.")
	}
}
