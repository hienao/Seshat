package releasenotes

import "testing"

func validRelease() Release {
	return Release{
		SchemaVersion: SchemaVersion,
		Channel:       "beta",
		Version:       "v1.2.3",
		Summary:       LocalizedText{English: "Summary", Chinese: "摘要"},
		Changes:       []Change{{ID: "feature", Type: "feature", Text: LocalizedText{English: "Feature", Chinese: "功能"}}},
		UpgradeNotes:  LocalizedList{English: []string{"Restart"}, Chinese: []string{"重启"}},
	}
}

func TestValidateReleaseRequiresBothLanguages(t *testing.T) {
	release := validRelease()
	if err := ValidateRelease(release, "beta", "v1.2.3"); err != nil {
		t.Fatal(err)
	}
	release.Changes[0].Text.Chinese = ""
	if err := ValidateRelease(release, "beta", "v1.2.3"); err == nil {
		t.Fatal("release without Chinese change text was accepted")
	}
}

func TestValidateFeedRequiresChannelAndLatestVersion(t *testing.T) {
	release := validRelease()
	feed := Feed{SchemaVersion: SchemaVersion, Channel: "beta", LatestVersion: release.Version, Releases: []Release{release}}
	if err := ValidateFeed(feed, "beta", "v1.2.3"); err != nil {
		t.Fatal(err)
	}
	feed.Channel = "release"
	if err := ValidateFeed(feed, "beta", "v1.2.3"); err == nil {
		t.Fatal("cross-channel feed was accepted")
	}
}
