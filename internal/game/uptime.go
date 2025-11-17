package game

import (
	"dmud/internal/components"
	"fmt"
	"time"
)

func handleUptime(player *components.Player, args []string, game *Game) {
	// Calculate uptime
	uptime := time.Since(game.StartTime)

	// Format uptime nicely
	days := int(uptime.Hours() / 24)
	hours := int(uptime.Hours()) % 24
	minutes := int(uptime.Minutes()) % 60
	seconds := int(uptime.Seconds()) % 60

	var uptimeStr string
	if days > 0 {
		uptimeStr = fmt.Sprintf("%dd %dh %dm %ds", days, hours, minutes, seconds)
	} else if hours > 0 {
		uptimeStr = fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	} else if minutes > 0 {
		uptimeStr = fmt.Sprintf("%dm %ds", minutes, seconds)
	} else {
		uptimeStr = fmt.Sprintf("%ds", seconds)
	}

	// Get current player count
	game.playersMu.RLock()
	currentPlayers := len(game.players)
	game.playersMu.RUnlock()

	// Get unique IP count
	game.UniqueIPsMu.RLock()
	uniqueIPs := len(game.UniqueIPs)
	game.UniqueIPsMu.RUnlock()

	// Get total connections
	game.TotalConnectMu.RLock()
	totalConnects := game.TotalConnects
	game.TotalConnectMu.RUnlock()

	// Format output
	var output string
	output += "==============================================\n"
	output += "               SERVER STATUS                  \n"
	output += "==============================================\n\n"
	output += fmt.Sprintf("Server started:  %s\n", game.StartTime.Format("Mon Jan 2 15:04:05 MST 2006"))
	output += fmt.Sprintf("Uptime:          %s\n", uptimeStr)
	output += fmt.Sprintf("Current players: %d\n", currentPlayers)
	output += fmt.Sprintf("Total connects:  %d\n", totalConnects)
	output += fmt.Sprintf("Unique players:  %d\n", uniqueIPs)
	output += "\n==============================================\n"

	player.Broadcast(output)
}
