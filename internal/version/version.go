package version

// Version information injected at build time
var (
	// GitCommit is the git commit hash
	GitCommit = "dev"
	// BuildTime is when the binary was built
	BuildTime = "unknown"
)

// GetVersion returns the current version info
func GetVersion() string {
	return GitCommit
}

// GetBuildTime returns when the binary was built
func GetBuildTime() string {
	return BuildTime
}

// GetGitHubURL returns the GitHub URL for this commit
func GetGitHubURL() string {
	if GitCommit == "dev" {
		return "https://github.com/dustywusty/dmud"
	}
	return "https://github.com/dustywusty/dmud/commit/" + GitCommit
}
