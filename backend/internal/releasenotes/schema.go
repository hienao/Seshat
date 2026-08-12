package releasenotes

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"golang.org/x/mod/semver"
)

const SchemaVersion = 1

type LocalizedText struct {
	English string `json:"en"`
	Chinese string `json:"zh-CN"`
}

type LocalizedList struct {
	English []string `json:"en"`
	Chinese []string `json:"zh-CN"`
}

type Change struct {
	ID   string        `json:"id"`
	Type string        `json:"type"`
	Text LocalizedText `json:"text"`
}

type Release struct {
	SchemaVersion int           `json:"schema_version"`
	Channel       string        `json:"channel"`
	Version       string        `json:"version"`
	Summary       LocalizedText `json:"summary"`
	Changes       []Change      `json:"changes"`
	UpgradeNotes  LocalizedList `json:"upgrade_notes"`
	PublishedAt   string        `json:"published_at,omitempty"`
	ReleaseURL    string        `json:"release_url,omitempty"`
	ImageTag      string        `json:"image_tag,omitempty"`
}

type Feed struct {
	SchemaVersion int       `json:"schema_version"`
	Channel       string    `json:"channel"`
	LatestVersion string    `json:"latest_version"`
	Releases      []Release `json:"releases"`
}

func ValidateRelease(release Release, expectedChannel, expectedVersion string) error {
	if release.SchemaVersion != SchemaVersion {
		return fmt.Errorf("schema_version must be %d", SchemaVersion)
	}
	if release.Channel != expectedChannel || (release.Channel != "beta" && release.Channel != "release") {
		return fmt.Errorf("channel must be %q", expectedChannel)
	}
	if !semver.IsValid(release.Version) || release.Version != expectedVersion {
		return fmt.Errorf("version must be %q", expectedVersion)
	}
	if strings.TrimSpace(release.Summary.English) == "" || strings.TrimSpace(release.Summary.Chinese) == "" {
		return errors.New("summary must include non-empty en and zh-CN text")
	}
	if len(release.Changes) == 0 {
		return errors.New("changes must contain at least one item")
	}
	seen := make(map[string]bool, len(release.Changes))
	for index, change := range release.Changes {
		if strings.TrimSpace(change.ID) == "" || seen[change.ID] {
			return fmt.Errorf("changes[%d].id must be non-empty and unique", index)
		}
		seen[change.ID] = true
		switch change.Type {
		case "feature", "fix", "security", "change", "deprecated":
		default:
			return fmt.Errorf("changes[%d].type is unsupported", index)
		}
		if strings.TrimSpace(change.Text.English) == "" || strings.TrimSpace(change.Text.Chinese) == "" {
			return fmt.Errorf("changes[%d].text must include non-empty en and zh-CN text", index)
		}
	}
	if len(release.UpgradeNotes.English) != len(release.UpgradeNotes.Chinese) {
		return errors.New("upgrade_notes en and zh-CN lists must have the same length")
	}
	for index := range release.UpgradeNotes.English {
		if strings.TrimSpace(release.UpgradeNotes.English[index]) == "" || strings.TrimSpace(release.UpgradeNotes.Chinese[index]) == "" {
			return fmt.Errorf("upgrade_notes[%d] must include non-empty en and zh-CN text", index)
		}
	}
	return nil
}

func ValidateFeed(feed Feed, expectedChannel, expectedLatest string) error {
	if feed.SchemaVersion != SchemaVersion {
		return fmt.Errorf("schema_version must be %d", SchemaVersion)
	}
	if feed.Channel != expectedChannel {
		return fmt.Errorf("feed channel must be %q", expectedChannel)
	}
	if !semver.IsValid(feed.LatestVersion) || feed.LatestVersion != expectedLatest {
		return fmt.Errorf("latest_version must be %q", expectedLatest)
	}
	if len(feed.Releases) == 0 {
		return errors.New("feed must contain releases")
	}
	seen := make(map[string]bool, len(feed.Releases))
	for _, release := range feed.Releases {
		if seen[release.Version] {
			return fmt.Errorf("duplicate release version %s", release.Version)
		}
		seen[release.Version] = true
		if err := ValidateRelease(release, expectedChannel, release.Version); err != nil {
			return fmt.Errorf("release %s: %w", release.Version, err)
		}
		if semver.Compare(release.Version, feed.LatestVersion) > 0 {
			return fmt.Errorf("release %s is newer than latest_version", release.Version)
		}
	}
	if !seen[feed.LatestVersion] {
		return errors.New("feed does not contain latest_version")
	}
	return nil
}

func Sort(releases []Release) {
	sort.Slice(releases, func(i, j int) bool { return semver.Compare(releases[i].Version, releases[j].Version) < 0 })
}
