package service

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"seshat/internal/buildinfo"
	"seshat/internal/releasenotes"
)

func localizedUpdateRelease(channel, version string) releasenotes.Release {
	return releasenotes.Release{
		SchemaVersion: releasenotes.SchemaVersion,
		Channel:       channel,
		Version:       version,
		Summary:       releasenotes.LocalizedText{English: "Summary " + version, Chinese: "摘要 " + version},
		Changes: []releasenotes.Change{{
			ID:   "change-" + version,
			Type: "feature",
			Text: releasenotes.LocalizedText{English: "English " + version, Chinese: "中文 " + version},
		}},
		UpgradeNotes: releasenotes.LocalizedList{English: []string{}, Chinese: []string{}},
	}
}

func TestUpdateServiceKeepsBetaAndReleaseIndependent(t *testing.T) {
	var requests atomic.Int32
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/releases":
			_ = json.NewEncoder(writer).Encode([]githubUpdateRelease{
				{TagName: "beta-v0.9.0", Prerelease: true, PublishedAt: "2026-08-09T10:00:00Z", HTMLURL: "https://github.com/hienao/Seshat/releases/tag/beta-v0.9.0", Assets: []githubUpdateAsset{{Name: updateFeedAssetName, BrowserDownloadURL: server.URL + "/beta-feed", Size: 512}}},
				{TagName: "v9.0.0", Prerelease: false, Assets: []githubUpdateAsset{{Name: updateFeedAssetName, BrowserDownloadURL: server.URL + "/release-feed", Size: 512}}},
				{TagName: "beta-v99.0.0", Prerelease: true, Draft: true, Assets: []githubUpdateAsset{{Name: updateFeedAssetName, BrowserDownloadURL: server.URL + "/beta-feed", Size: 512}}},
				{TagName: "beta-v0.8.0", Prerelease: true, PublishedAt: "2026-08-08T10:00:00Z", HTMLURL: "https://github.com/hienao/Seshat/releases/tag/beta-v0.8.0"},
			})
		case "/beta-feed":
			_ = json.NewEncoder(writer).Encode(releasenotes.Feed{
				SchemaVersion: releasenotes.SchemaVersion,
				Channel:       "beta",
				LatestVersion: "v0.9.0",
				Releases: []releasenotes.Release{
					localizedUpdateRelease("beta", "v0.9.0"),
					localizedUpdateRelease("beta", "v0.7.0"),
					localizedUpdateRelease("beta", "v0.8.0"),
				},
			})
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	service := newUpdateService(buildinfo.Info{Version: "v0.7.0", Channel: "beta", Commit: "abc", BuildTime: "now"}, server.URL+"/releases", server.Client(), nil)
	service.cacheTTL = time.Hour
	status, err := service.Check(t.Context(), false)
	if err != nil {
		t.Fatal(err)
	}
	if !status.Supported || !status.UpdateAvailable || status.LatestVersion != "v0.9.0" || status.ImageTag != "beta-v0.9.0" {
		t.Fatalf("unexpected beta update status: %+v", status)
	}
	if len(status.Releases) != 2 || status.Releases[0].Version != "v0.8.0" || status.Releases[1].Version != "v0.9.0" {
		t.Fatalf("unexpected beta release range: %+v", status.Releases)
	}
	if status.Releases[0].ImageTag != "beta-v0.8.0" || status.Releases[0].ReleaseURL == "" || status.Releases[1].PublishedAt == "" {
		t.Fatalf("missing release metadata: %+v", status.Releases)
	}
	requestCount := requests.Load()
	if _, err := service.Check(t.Context(), false); err != nil {
		t.Fatal(err)
	}
	if requests.Load() != requestCount {
		t.Fatal("cached update check performed another remote request")
	}
	if _, err := service.Check(t.Context(), true); err != nil {
		t.Fatal(err)
	}
	if requests.Load() <= requestCount {
		t.Fatal("forced update check did not refresh remote data")
	}
}

func TestUpdateServiceChecksOnlyStableReleases(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path == "/releases" {
			_ = json.NewEncoder(writer).Encode([]githubUpdateRelease{
				{TagName: "beta-v99.0.0", Prerelease: true, Assets: []githubUpdateAsset{{Name: updateFeedAssetName, BrowserDownloadURL: server.URL + "/beta", Size: 100}}},
				{TagName: "v0.2.0", HTMLURL: "https://github.com/hienao/Seshat/releases/tag/v0.2.0", Assets: []githubUpdateAsset{{Name: updateFeedAssetName, BrowserDownloadURL: server.URL + "/feed", Size: 100}}},
			})
			return
		}
		_ = json.NewEncoder(writer).Encode(releasenotes.Feed{SchemaVersion: 1, Channel: "release", LatestVersion: "v0.2.0", Releases: []releasenotes.Release{localizedUpdateRelease("release", "v0.1.0"), localizedUpdateRelease("release", "v0.2.0")}})
	}))
	defer server.Close()
	service := newUpdateService(buildinfo.Info{Version: "v0.1.0", Channel: "release"}, server.URL+"/releases", server.Client(), nil)
	status, err := service.Check(t.Context(), false)
	if err != nil {
		t.Fatal(err)
	}
	if status.LatestVersion != "v0.2.0" || status.ImageTag != "v0.2.0" || len(status.Releases) != 1 || status.Releases[0].Channel != "release" {
		t.Fatalf("unexpected release status: %+v", status)
	}
}

func TestUpdateServiceDoesNotCheckDevelopmentBuilds(t *testing.T) {
	service := newUpdateService(buildinfo.Info{Version: "dev", Channel: "dev"}, "https://invalid.example/releases", &http.Client{}, nil)
	status, err := service.Check(t.Context(), false)
	if err != nil {
		t.Fatal(err)
	}
	if status.Supported || status.Current.Version != "dev" || status.UpdateAvailable {
		t.Fatalf("unexpected development status: %+v", status)
	}
}

func TestUpdateServiceRejectsCrossChannelFeed(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path == "/releases" {
			_ = json.NewEncoder(writer).Encode([]githubUpdateRelease{{TagName: "beta-v0.8.0", Prerelease: true, Assets: []githubUpdateAsset{{Name: updateFeedAssetName, BrowserDownloadURL: server.URL + "/feed", Size: 100}}}})
			return
		}
		_ = json.NewEncoder(writer).Encode(releasenotes.Feed{SchemaVersion: 1, Channel: "release", LatestVersion: "v0.8.0", Releases: []releasenotes.Release{localizedUpdateRelease("release", "v0.8.0")}})
	}))
	defer server.Close()
	service := newUpdateService(buildinfo.Info{Version: "v0.7.0", Channel: "beta"}, server.URL+"/releases", server.Client(), nil)
	if _, err := service.Check(t.Context(), false); err == nil {
		t.Fatal("cross-channel update feed was accepted")
	}
}
