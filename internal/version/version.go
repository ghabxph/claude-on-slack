package version

import "time"

const (
	BuildTime = "development"  // Set during build
	GitHash   = ""             // Set during build
)

func GetVersionInfo(appVersion string) map[string]string {
	return map[string]string{
		"version":    appVersion,
		"build_time": BuildTime,
		"git_hash":   GitHash,
		"timestamp":  time.Now().Format(time.RFC3339),
	}
}

func GetBuildInfo(appVersion string) string {
	if BuildTime == "development" {
		return appVersion + "-dev"
	}
	return appVersion + " (built " + BuildTime + ")"
}