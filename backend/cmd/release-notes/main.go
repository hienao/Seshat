package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"seshat/internal/releasenotes"
)

func main() {
	if len(os.Args) < 4 {
		fail(errors.New("usage: release-notes <validate|feed|markdown> <channel> <version> [root-or-output]"))
	}
	command, channel, version := os.Args[1], os.Args[2], os.Args[3]
	root := filepath.Join("..", "release-notes")
	if command == "validate" && len(os.Args) == 5 {
		root = os.Args[4]
	}
	releases, err := loadReleases(root, channel)
	if err != nil {
		fail(err)
	}
	if err := validateTarget(releases, channel, version); err != nil {
		fail(err)
	}
	switch command {
	case "validate":
		fmt.Printf("OK: %s %s release notes are valid\n", channel, version)
	case "feed":
		if len(os.Args) != 5 {
			fail(errors.New("usage: release-notes feed <channel> <version> <output>"))
		}
		feed := releasenotes.Feed{SchemaVersion: releasenotes.SchemaVersion, Channel: channel, LatestVersion: version, Releases: releases}
		if err := releasenotes.ValidateFeed(feed, channel, version); err != nil {
			fail(err)
		}
		writeJSON(os.Args[4], feed)
	case "markdown":
		if len(os.Args) != 5 {
			fail(errors.New("usage: release-notes markdown <channel> <version> <output>"))
		}
		for _, release := range releases {
			if release.Version == version {
				writeMarkdown(os.Args[4], release)
				return
			}
		}
		fail(errors.New("target release notes not found"))
	default:
		fail(errors.New("unsupported command: " + command))
	}
}

func loadReleases(root, channel string) ([]releasenotes.Release, error) {
	if channel != "beta" && channel != "release" {
		return nil, errors.New("channel must be beta or release")
	}
	paths, err := filepath.Glob(filepath.Join(root, channel, "v*.json"))
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, errors.New("no release notes found")
	}
	releases := make([]releasenotes.Release, 0, len(paths))
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var release releasenotes.Release
		decoder := json.NewDecoder(strings.NewReader(string(data)))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&release); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		if err := releasenotes.ValidateRelease(release, channel, strings.TrimSuffix(filepath.Base(path), ".json")); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		releases = append(releases, release)
	}
	releasenotes.Sort(releases)
	return releases, nil
}

func validateTarget(releases []releasenotes.Release, channel, version string) error {
	for _, release := range releases {
		if release.Version == version {
			return releasenotes.ValidateRelease(release, channel, version)
		}
	}
	return fmt.Errorf("missing release-notes/%s/%s.json", channel, version)
}

func writeJSON(path string, value interface{}) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		fail(err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		fail(err)
	}
}

func writeMarkdown(path string, release releasenotes.Release) {
	const body = `## English

{{.Summary.English}}
{{range .Changes}}- {{.Text.English}}
{{end}}{{if .UpgradeNotes.English}}
### Upgrade notes
{{range .UpgradeNotes.English}}- {{.}}
{{end}}{{end}}
## 中文

{{.Summary.Chinese}}
{{range .Changes}}- {{.Text.Chinese}}
{{end}}{{if .UpgradeNotes.Chinese}}
### 升级说明
{{range .UpgradeNotes.Chinese}}- {{.}}
{{end}}{{end}}`
	file, err := os.Create(path)
	if err != nil {
		fail(err)
	}
	defer file.Close()
	if err := template.Must(template.New("release").Parse(body)).Execute(file, release); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "ERROR:", err)
	os.Exit(1)
}
