package version

import "time"

const (
	Version   = "2.6.3"
	BuildTime = "development"  // Set during build
	GitHash   = ""             // Set during build
)

func GetVersionInfo() map[string]string {
	return map[string]string{
		"version":    Version,
		"build_time": BuildTime,
		"git_hash":   GitHash,
		"timestamp":  time.Now().Format(time.RFC3339),
	}
}

func GetVersion() string {
	return Version
}

func GetBuildInfo() string {
	if BuildTime == "development" {
		return Version + "-dev"
	}
	return Version + " (built " + BuildTime + ")"
}