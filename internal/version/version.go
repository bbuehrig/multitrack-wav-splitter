package version

// Version is the semver for this build (no leading "v").
// Release builds may override via:
//
//	-ldflags "-X github.com/bbu/multitrack-wav-splitter/internal/version.Version=0.2.0"
var Version = "0.2.0"
