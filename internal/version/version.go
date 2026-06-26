package version

// Version is set at build time via -ldflags "-X github.com/packetmind/packetmind/internal/version.Version=..."
var Version = "dev"

// BuildTime is set at build time via -ldflags
var BuildTime = ""

// Commit is set at build time via -ldflags
var Commit = ""
