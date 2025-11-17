package game

import (
	"dmud/internal/components"
	"dmud/internal/version"
)

func handleVersion(player *components.Player, args []string, game *Game) {
	gitCommit := version.GetVersion()
	buildTime := version.GetBuildTime()
	githubURL := version.GetGitHubURL()

	var output string
	output += "==============================================\n"
	output += "               VERSION INFO                   \n"
	output += "==============================================\n\n"

	if gitCommit == "dev" {
		output += "Version:    development build\n"
		output += "Build time: " + buildTime + "\n"
		output += "Repository: " + githubURL + "\n"
	} else {
		output += "Commit:     " + gitCommit + "\n"
		output += "Build time: " + buildTime + "\n"
		output += "View:       " + githubURL + "\n"
	}

	output += "\n==============================================\n"

	player.Broadcast(output)
}
