package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/mod/semver"

	"seshat/internal/buildinfo"
	"seshat/internal/releasenotes"
)

const (
	defaultGitHubReleasesURL = "https://api.github.com/repos/hienao/Seshat/releases?per_page=100"
	updateFeedAssetName      = "update-feed.json"
	maxUpdateResponseBytes   = 1 << 20
	defaultUpdateCacheTTL    = 6 * time.Hour
)

var (
	ErrUpdateCheckUnavailable = errors.New("无法获取更新信息")
	ErrInvalidUpdateFeed      = errors.New("更新信息格式无效")
)

type UpdateStatusResponse struct {
	Supported       bool                   `json:"supported"`
	Current         buildinfo.Info         `json:"current"`
	LatestVersion   string                 `json:"latest_version"`
	UpdateAvailable bool                   `json:"update_available"`
	ImageTag        string                 `json:"image_tag"`
	Releases        []releasenotes.Release `json:"releases"`
	CheckedAt       time.Time              `json:"checked_at"`
}

type githubUpdateRelease struct {
	TagName     string              `json:"tag_name"`
	HTMLURL     string              `json:"html_url"`
	Draft       bool                `json:"draft"`
	Prerelease  bool                `json:"prerelease"`
	PublishedAt string              `json:"published_at"`
	Assets      []githubUpdateAsset `json:"assets"`
}

type githubUpdateAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

type updateCache struct {
	response  *UpdateStatusResponse
	expiresAt time.Time
}

type UpdateService struct {
	current      buildinfo.Info
	releasesURL  string
	httpClient   *http.Client
	settings     *SettingService
	cacheTTL     time.Duration
	now          func() time.Time
	allowedHosts map[string]bool
	allowHTTP    bool
	mu           sync.Mutex
	cache        updateCache
}

func NewUpdateService() *UpdateService {
	return newUpdateService(buildinfo.Current(), defaultGitHubReleasesURL, &http.Client{Timeout: 5 * time.Second}, NewSettingService())
}

func newUpdateService(current buildinfo.Info, releasesURL string, client *http.Client, settings *SettingService) *UpdateService {
	allowedHosts := map[string]bool{
		"api.github.com":                       true,
		"github.com":                           true,
		"objects.githubusercontent.com":        true,
		"release-assets.githubusercontent.com": true,
	}
	parsed, _ := url.Parse(releasesURL)
	if parsed != nil && parsed.Hostname() != "" {
		allowedHosts[strings.ToLower(parsed.Hostname())] = true
	}
	return &UpdateService{
		current:      current,
		releasesURL:  releasesURL,
		httpClient:   client,
		settings:     settings,
		cacheTTL:     defaultUpdateCacheTTL,
		now:          time.Now,
		allowedHosts: allowedHosts,
		allowHTTP:    parsed != nil && parsed.Scheme == "http",
	}
}

func (s *UpdateService) CurrentVersion() buildinfo.Info {
	return s.current
}

func (s *UpdateService) Check(ctx context.Context, refresh bool) (*UpdateStatusResponse, error) {
	current := s.current
	if current.Channel != "beta" && current.Channel != "release" {
		return &UpdateStatusResponse{Supported: false, Current: current, Releases: []releasenotes.Release{}}, nil
	}
	if !semver.IsValid(current.Version) {
		return nil, ErrInvalidUpdateFeed
	}
	now := s.now()
	if !refresh {
		s.mu.Lock()
		if s.cache.response != nil && now.Before(s.cache.expiresAt) {
			cached := cloneUpdateStatus(s.cache.response)
			s.mu.Unlock()
			return cached, nil
		}
		s.mu.Unlock()
	}

	releases, err := s.fetchGitHubReleases(ctx)
	if err != nil {
		return nil, err
	}
	latest, latestVersion, err := latestChannelRelease(releases, current.Channel)
	if err != nil {
		return nil, err
	}
	feedAsset, ok := updateFeedAsset(latest.Assets)
	if !ok || feedAsset.Size > maxUpdateResponseBytes {
		return nil, ErrInvalidUpdateFeed
	}
	feed, err := s.fetchUpdateFeed(ctx, feedAsset.BrowserDownloadURL)
	if err != nil {
		return nil, err
	}
	if err := releasenotes.ValidateFeed(feed, current.Channel, latestVersion); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidUpdateFeed, err)
	}

	metadata := releaseMetadataByVersion(releases, current.Channel)
	updates := make([]releasenotes.Release, 0, len(feed.Releases))
	for _, release := range feed.Releases {
		if semver.Compare(release.Version, current.Version) <= 0 || semver.Compare(release.Version, latestVersion) > 0 {
			continue
		}
		if remote, exists := metadata[release.Version]; exists {
			release.PublishedAt = remote.PublishedAt
			release.ReleaseURL = remote.HTMLURL
		}
		release.ImageTag = imageTagForChannel(current.Channel, release.Version)
		updates = append(updates, release)
	}
	releasenotes.Sort(updates)
	response := &UpdateStatusResponse{
		Supported:       true,
		Current:         current,
		LatestVersion:   latestVersion,
		UpdateAvailable: semver.Compare(latestVersion, current.Version) > 0,
		ImageTag:        imageTagForChannel(current.Channel, latestVersion),
		Releases:        updates,
		CheckedAt:       now.UTC(),
	}
	s.mu.Lock()
	s.cache = updateCache{response: cloneUpdateStatus(response), expiresAt: now.Add(s.cacheTTL)}
	s.mu.Unlock()
	return response, nil
}

func (s *UpdateService) fetchGitHubReleases(ctx context.Context) ([]githubUpdateRelease, error) {
	all := make([]githubUpdateRelease, 0, 100)
	for page := 1; page <= 10; page++ {
		endpoint, err := url.Parse(s.releasesURL)
		if err != nil {
			return nil, ErrUpdateCheckUnavailable
		}
		query := endpoint.Query()
		query.Set("per_page", "100")
		query.Set("page", fmt.Sprintf("%d", page))
		endpoint.RawQuery = query.Encode()
		var releases []githubUpdateRelease
		if err := s.getJSON(ctx, endpoint.String(), &releases); err != nil {
			return nil, err
		}
		all = append(all, releases...)
		if len(releases) < 100 {
			return all, nil
		}
	}
	return all, nil
}

func (s *UpdateService) fetchUpdateFeed(ctx context.Context, endpoint string) (releasenotes.Feed, error) {
	var feed releasenotes.Feed
	if err := s.getJSON(ctx, endpoint, &feed); err != nil {
		return feed, err
	}
	return feed, nil
}

func (s *UpdateService) getJSON(ctx context.Context, endpoint string, target interface{}) error {
	if !s.validUpdateURL(endpoint) {
		return ErrInvalidUpdateFeed
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return ErrUpdateCheckUnavailable
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "Seshat-Update-Checker")
	client, err := s.updateHTTPClient()
	if err != nil {
		return ErrUpdateCheckUnavailable
	}
	response, err := client.Do(req)
	if err != nil {
		return ErrUpdateCheckUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: HTTP %d", ErrUpdateCheckUnavailable, response.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, maxUpdateResponseBytes+1))
	if err != nil || len(data) > maxUpdateResponseBytes {
		return ErrInvalidUpdateFeed
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	if err := decoder.Decode(target); err != nil {
		return ErrInvalidUpdateFeed
	}
	return nil
}

func (s *UpdateService) updateHTTPClient() (*http.Client, error) {
	client := s.httpClient
	if s.settings != nil {
		if proxyURL := s.settings.HTTPProxyURL(); proxyURL != "" {
			configured, err := httpClientWithProxy(client, proxyURL)
			if err != nil {
				return nil, err
			}
			client = configured
		}
	}
	copy := *client
	copy.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return errors.New("更新检查重定向次数过多")
		}
		if !s.validUpdateURL(req.URL.String()) {
			return errors.New("更新检查禁止重定向到非信任地址")
		}
		return nil
	}
	return &copy, nil
}

func (s *UpdateService) validUpdateURL(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.User != nil || !s.allowedHosts[strings.ToLower(parsed.Hostname())] {
		return false
	}
	return parsed.Scheme == "https" || (s.allowHTTP && parsed.Scheme == "http")
}

func latestChannelRelease(releases []githubUpdateRelease, channel string) (githubUpdateRelease, string, error) {
	var selected githubUpdateRelease
	latestVersion := ""
	for _, release := range releases {
		version, ok := channelReleaseVersion(release, channel)
		if !ok || (latestVersion != "" && semver.Compare(version, latestVersion) <= 0) {
			continue
		}
		selected, latestVersion = release, version
	}
	if latestVersion == "" {
		return selected, "", ErrUpdateCheckUnavailable
	}
	return selected, latestVersion, nil
}

func channelReleaseVersion(release githubUpdateRelease, channel string) (string, bool) {
	if release.Draft {
		return "", false
	}
	tag := strings.TrimSpace(release.TagName)
	switch channel {
	case "beta":
		if !release.Prerelease || !strings.HasPrefix(tag, "beta-") {
			return "", false
		}
		tag = strings.TrimPrefix(tag, "beta-")
	case "release":
		if release.Prerelease || strings.HasPrefix(tag, "beta-") {
			return "", false
		}
	default:
		return "", false
	}
	return tag, semver.IsValid(tag)
}

func updateFeedAsset(assets []githubUpdateAsset) (githubUpdateAsset, bool) {
	for _, asset := range assets {
		if asset.Name == updateFeedAssetName && asset.BrowserDownloadURL != "" && asset.Size > 0 {
			return asset, true
		}
	}
	return githubUpdateAsset{}, false
}

func releaseMetadataByVersion(releases []githubUpdateRelease, channel string) map[string]githubUpdateRelease {
	result := make(map[string]githubUpdateRelease)
	for _, release := range releases {
		if version, ok := channelReleaseVersion(release, channel); ok {
			result[version] = release
		}
	}
	return result
}

func imageTagForChannel(channel, version string) string {
	if channel == "beta" {
		return "beta-" + version
	}
	return version
}

func cloneUpdateStatus(value *UpdateStatusResponse) *UpdateStatusResponse {
	if value == nil {
		return nil
	}
	copy := *value
	copy.Releases = append([]releasenotes.Release(nil), value.Releases...)
	for index := range copy.Releases {
		copy.Releases[index].Changes = append([]releasenotes.Change(nil), value.Releases[index].Changes...)
		copy.Releases[index].UpgradeNotes.English = append([]string(nil), value.Releases[index].UpgradeNotes.English...)
		copy.Releases[index].UpgradeNotes.Chinese = append([]string(nil), value.Releases[index].UpgradeNotes.Chinese...)
	}
	sort.SliceStable(copy.Releases, func(i, j int) bool { return semver.Compare(copy.Releases[i].Version, copy.Releases[j].Version) < 0 })
	return &copy
}
