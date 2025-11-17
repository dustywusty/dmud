package game

import (
	"dmud/internal/components"
	"dmud/internal/version"
)

func handleVersion(player *components.Player, args []string, game *Game) {
	gitCommit := version.GetVersion()
	githubURL := version.GetGitHubURL()

	var output string
	output += "==============================================\n"
	output += "               VERSION INFO                   \n"
	output += "==============================================\n\n"
	output += "Commit: " + gitCommit + "\n"
	output += "View:   " + githubURL + "\n"
	output += "\n==============================================\n"

	player.Broadcast(output)
}
