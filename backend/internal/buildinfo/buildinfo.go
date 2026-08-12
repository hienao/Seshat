package buildinfo

import "strings"

var (
	Version   = "dev"
	Channel   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

type Info struct {
	Version   string `json:"version"`
	Channel   string `json:"channel"`
	Commit    string `json:"commit"`
	BuildTime string `json:"build_time"`
}

func Current() Info {
	return Info{
		Version:   normalized(Version, "dev"),
		Channel:   normalized(strings.ToLower(Channel), "dev"),
		Commit:    normalized(Commit, "unknown"),
		BuildTime: normalized(BuildTime, "unknown"),
	}
}

func normalized(value, fallback string) string {
	if value = strings.TrimSpace(value); value != "" {
		return value
	}
	return fallback
}
