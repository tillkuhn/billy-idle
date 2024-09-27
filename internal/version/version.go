package version

/*
# Make sure you have the following exclude rule in your .golangci.yml
issues:

	exclude-rules:
	  # this is the exception for commit, data and version variable which with pass in using ldflags
	  - path: internal/version/version.go
	    linters: [gochecknoglobals]
*/
var (
	// Version is the current version of the oras.
	Version = "latest"
	// BuildMetadata is the extra build time data
	BuildMetadata = "unreleased"
	// GitCommit is the git sha1
	GitCommit = ""
	// GitTreeState is the state of the git tree
	GitTreeState = ""
)
