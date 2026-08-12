package service

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
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

func TestUpdateServiceReadsOnlyCurrentChannelStaticFeed(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		if request.URL.Path != "/updates/v1/beta.json" {
			t.Errorf("unexpected update feed path: %s", request.URL.Path)
			http.NotFound(writer, request)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
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
	}))
	defer server.Close()

	service := newUpdateService(buildinfo.Info{Version: "v0.7.0", Channel: "beta", Commit: "abc", BuildTime: "now"}, server.URL+"/updates/v1/", server.Client(), nil)
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
	if status.Releases[0].ImageTag != "beta-v0.8.0" || status.Releases[0].ReleaseURL != "https://github.com/hienao/Seshat/releases/tag/beta-v0.8.0" {
		t.Fatalf("missing generated release metadata: %+v", status.Releases)
	}
	if _, err := service.Check(t.Context(), false); err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 1 {
		t.Fatalf("cached update check performed %d requests, want 1", requests.Load())
	}
	if _, err := service.Check(t.Context(), true); err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 2 {
		t.Fatalf("forced update check performed %d requests, want 2", requests.Load())
	}
}

func TestUpdateServiceChecksOnlyStableFeed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/release.json" {
			t.Errorf("unexpected update feed path: %s", request.URL.Path)
		}
		_ = json.NewEncoder(writer).Encode(releasenotes.Feed{SchemaVersion: 1, Channel: "release", LatestVersion: "v0.2.0", Releases: []releasenotes.Release{localizedUpdateRelease("release", "v0.1.0"), localizedUpdateRelease("release", "v0.2.0")}})
	}))
	defer server.Close()
	service := newUpdateService(buildinfo.Info{Version: "v0.1.0", Channel: "release"}, server.URL, server.Client(), nil)
	status, err := service.Check(t.Context(), false)
	if err != nil {
		t.Fatal(err)
	}
	if status.LatestVersion != "v0.2.0" || status.ImageTag != "v0.2.0" || len(status.Releases) != 1 || status.Releases[0].Channel != "release" {
		t.Fatalf("unexpected release status: %+v", status)
	}
}

func TestUpdateServiceDoesNotCheckDevelopmentBuilds(t *testing.T) {
	service := newUpdateService(buildinfo.Info{Version: "dev", Channel: "dev"}, "https://invalid.example/updates/v1", &http.Client{}, nil)
	status, err := service.Check(t.Context(), false)
	if err != nil {
		t.Fatal(err)
	}
	if status.Supported || status.Current.Version != "dev" || status.UpdateAvailable {
		t.Fatalf("unexpected development status: %+v", status)
	}
}

func TestUpdateServiceRejectsCrossChannelFeed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_ = json.NewEncoder(writer).Encode(releasenotes.Feed{SchemaVersion: 1, Channel: "release", LatestVersion: "v0.8.0", Releases: []releasenotes.Release{localizedUpdateRelease("release", "v0.8.0")}})
	}))
	defer server.Close()
	service := newUpdateService(buildinfo.Info{Version: "v0.7.0", Channel: "beta"}, server.URL, server.Client(), nil)
	if _, err := service.Check(t.Context(), false); err == nil {
		t.Fatal("cross-channel update feed was accepted")
	}
}

func TestUpdateServiceReturnsLastSuccessWhenRefreshFails(t *testing.T) {
	var fail atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if fail.Load() {
			http.Error(writer, "temporary failure", http.StatusServiceUnavailable)
			return
		}
		_ = json.NewEncoder(writer).Encode(releasenotes.Feed{SchemaVersion: 1, Channel: "beta", LatestVersion: "v0.9.0", Releases: []releasenotes.Release{localizedUpdateRelease("beta", "v0.9.0")}})
	}))
	defer server.Close()
	service := newUpdateService(buildinfo.Info{Version: "v0.8.0", Channel: "beta"}, server.URL, server.Client(), nil)
	first, err := service.Check(t.Context(), false)
	if err != nil {
		t.Fatal(err)
	}
	fail.Store(true)
	stale, err := service.Check(t.Context(), true)
	if err != nil {
		t.Fatal(err)
	}
	if stale.LatestVersion != first.LatestVersion || !stale.CheckedAt.Equal(first.CheckedAt) {
		t.Fatalf("last successful status was not preserved: first=%+v stale=%+v", first, stale)
	}
}

func TestUpdateServiceCoalescesConcurrentCacheMisses(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		time.Sleep(20 * time.Millisecond)
		_ = json.NewEncoder(writer).Encode(releasenotes.Feed{SchemaVersion: 1, Channel: "beta", LatestVersion: "v0.9.0", Releases: []releasenotes.Release{localizedUpdateRelease("beta", "v0.9.0")}})
	}))
	defer server.Close()
	service := newUpdateService(buildinfo.Info{Version: "v0.8.0", Channel: "beta"}, server.URL, server.Client(), nil)

	var group sync.WaitGroup
	for range 8 {
		group.Add(1)
		go func() {
			defer group.Done()
			if _, err := service.Check(t.Context(), false); err != nil {
				t.Errorf("concurrent update check failed: %v", err)
			}
		}()
	}
	group.Wait()
	if requests.Load() != 1 {
		t.Fatalf("concurrent update checks performed %d requests, want 1", requests.Load())
	}
}
