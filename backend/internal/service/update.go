package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/mod/semver"

	"seshat/internal/buildinfo"
	"seshat/internal/releasenotes"
)

const (
	defaultUpdateFeedBaseURL = "https://seshatapp.pages.dev/updates/v1"
	updateFeedBaseURLEnv     = "SESHAT_UPDATE_FEED_BASE_URL"
	maxUpdateResponseBytes   = 1 << 20
	defaultUpdateCacheTTL    = 6 * time.Hour
	githubReleasesBaseURL    = "https://github.com/hienao/Seshat/releases/tag/"
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

type updateCache struct {
	response  *UpdateStatusResponse
	expiresAt time.Time
}

type UpdateService struct {
	current      buildinfo.Info
	feedBaseURL  string
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
	feedBaseURL := strings.TrimSpace(os.Getenv(updateFeedBaseURLEnv))
	if feedBaseURL == "" {
		feedBaseURL = defaultUpdateFeedBaseURL
	}
	return newUpdateService(buildinfo.Current(), feedBaseURL, &http.Client{Timeout: 5 * time.Second}, NewSettingService())
}

func newUpdateService(current buildinfo.Info, feedBaseURL string, client *http.Client, settings *SettingService) *UpdateService {
	feedBaseURL = strings.TrimRight(strings.TrimSpace(feedBaseURL), "/")
	allowedHosts := map[string]bool{}
	parsed, _ := url.Parse(feedBaseURL)
	if parsed != nil && parsed.Hostname() != "" {
		allowedHosts[strings.ToLower(parsed.Hostname())] = true
	}
	return &UpdateService{
		current:      current,
		feedBaseURL:  feedBaseURL,
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
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	if !refresh && s.cache.response != nil && now.Before(s.cache.expiresAt) {
		return cloneUpdateStatus(s.cache.response), nil
	}

	feed, err := s.fetchUpdateFeed(ctx, current.Channel)
	if err != nil {
		if s.cache.response != nil {
			return cloneUpdateStatus(s.cache.response), nil
		}
		return nil, err
	}
	if err := releasenotes.ValidateFeed(feed, current.Channel, feed.LatestVersion); err != nil {
		if s.cache.response != nil {
			return cloneUpdateStatus(s.cache.response), nil
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidUpdateFeed, err)
	}

	latestVersion := feed.LatestVersion
	updates := make([]releasenotes.Release, 0, len(feed.Releases))
	for _, release := range feed.Releases {
		if semver.Compare(release.Version, current.Version) <= 0 || semver.Compare(release.Version, latestVersion) > 0 {
			continue
		}
		if strings.TrimSpace(release.ReleaseURL) == "" {
			release.ReleaseURL = releaseURLForChannel(current.Channel, release.Version)
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
	s.cache = updateCache{response: cloneUpdateStatus(response), expiresAt: now.Add(s.cacheTTL)}
	return response, nil
}

func (s *UpdateService) fetchUpdateFeed(ctx context.Context, channel string) (releasenotes.Feed, error) {
	var feed releasenotes.Feed
	endpoint := s.feedBaseURL + "/" + channel + ".json"
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
	req.Header.Set("Accept", "application/json")
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

func imageTagForChannel(channel, version string) string {
	if channel == "beta" {
		return "beta-" + version
	}
	return version
}

func releaseURLForChannel(channel, version string) string {
	tag := imageTagForChannel(channel, version)
	return githubReleasesBaseURL + url.PathEscape(tag)
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
