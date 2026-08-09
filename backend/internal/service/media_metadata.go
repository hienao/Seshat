package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	appLogging "seshat/internal/logging"
	"seshat/internal/model"
	"seshat/internal/webhook"
	"seshat/pkg/database"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	defaultTMDBAPIBaseURL = "https://api.themoviedb.org/3"
	tmdbImageBaseURL      = "https://image.tmdb.org/t/p/w342"
	defaultProxyTestURL   = "https://api.themoviedb.org/3/configuration"
)

type MediaMetadataService struct {
	apiBaseURL   string
	proxyTestURL string
	httpClient   *http.Client
	settings     *SettingService
	adapters     map[string]mediaServerMetadataAdapter
}

type mediaMetadataLookup struct {
	cacheKey      string
	externalID    string
	mediaType     string
	path          string
	seriesName    string
	seriesYear    string
	seasonNumber  int
	episodeNumber int
}

type tmdbMedia struct {
	ID            int64  `json:"id"`
	Title         string `json:"title"`
	Name          string `json:"name"`
	Overview      string `json:"overview"`
	PosterPath    string `json:"poster_path"`
	StillPath     string `json:"still_path"`
	MediaType     string `json:"media_type"`
	ShowID        int64  `json:"show_id"`
	SeasonNumber  int    `json:"season_number"`
	EpisodeNumber int    `json:"episode_number"`
}

type tmdbTVSearchResponse struct {
	Results []tmdbMedia `json:"results"`
}

type tmdbAPIError struct {
	statusCode int
}

func (e *tmdbAPIError) Error() string {
	return fmt.Sprintf("TMDB 返回状态码 %d", e.statusCode)
}

type TestTMDBConnectionRequest struct {
	TMDBAPIKey   string `json:"tmdb_api_key" binding:"required"`
	HTTPProxyURL string `json:"http_proxy_url"`
	UseProxy     bool   `json:"use_proxy"`
}

type TestHTTPProxyRequest struct {
	HTTPProxyURL string `json:"http_proxy_url" binding:"required"`
}

func NewMediaMetadataService() *MediaMetadataService {
	service := &MediaMetadataService{
		apiBaseURL:   defaultTMDBAPIBaseURL,
		proxyTestURL: defaultProxyTestURL,
		httpClient:   &http.Client{Timeout: 4 * time.Second},
		settings:     NewSettingService(),
	}
	service.registerAdapter(newJellyfinMediaMetadataAdapter())
	service.registerAdapter(newEmbyMediaMetadataAdapter())
	return service
}

func (s *MediaMetadataService) registerAdapter(adapter mediaServerMetadataAdapter) {
	if s.adapters == nil {
		s.adapters = make(map[string]mediaServerMetadataAdapter)
	}
	s.adapters[adapter.Code()] = adapter
}

func (s *MediaMetadataService) adapter(appCode string) (mediaServerMetadataAdapter, bool) {
	adapter, ok := s.adapters[appCode]
	return adapter, ok
}

func (s *MediaMetadataService) TestMediaServer(ctx context.Context, appCode, serverURL, apiKey string) error {
	adapter, ok := s.adapter(appCode)
	if !ok {
		return ErrMediaAPIUnsupported
	}
	settings, err := normalizeMediaServerSettings(serverURL, apiKey)
	if err != nil {
		return err
	}
	return adapter.Test(ctx, settings)
}

func (s *MediaMetadataService) TestHTTPProxy(request *TestHTTPProxyRequest) error {
	proxyURL := strings.TrimSpace(request.HTTPProxyURL)
	if !validHTTPProxyURL(proxyURL) {
		return ErrInvalidHTTPProxyURL
	}
	client, err := httpClientWithProxy(s.httpClient, proxyURL)
	if err != nil {
		return ErrInvalidHTTPProxyURL
	}
	target := s.proxyTestURL
	if target == "" {
		target = defaultProxyTestURL
	}
	req, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		return errors.New("无法创建 HTTP 代理测试请求")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Seshat-Proxy-Test")
	response, err := client.Do(req)
	if err != nil {
		return errors.New("无法通过 HTTP 代理访问测试地址")
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusProxyAuthRequired {
		return errors.New("HTTP 代理认证失败")
	}
	if response.StatusCode >= http.StatusInternalServerError {
		return fmt.Errorf("HTTP 代理测试地址返回状态码 %d", response.StatusCode)
	}
	return nil
}

func (s *MediaMetadataService) TestConnection(request *TestTMDBConnectionRequest) error {
	apiKey := strings.TrimSpace(request.TMDBAPIKey)
	if apiKey == "" || len(apiKey) > 2000 {
		return ErrInvalidTMDBAPIKey
	}
	client := s.httpClient
	if request.UseProxy {
		proxyURL := strings.TrimSpace(request.HTTPProxyURL)
		if !validHTTPProxyURL(proxyURL) {
			return ErrInvalidHTTPProxyURL
		}
		configured, err := httpClientWithProxy(s.httpClient, proxyURL)
		if err != nil {
			return ErrInvalidHTTPProxyURL
		}
		client = configured
	}
	requestURL, err := url.Parse(strings.TrimRight(s.apiBaseURL, "/") + "/configuration")
	if err != nil {
		return errors.New("无法创建 TMDB 测试请求")
	}
	query := requestURL.Query()
	query.Set("api_key", apiKey)
	requestURL.RawQuery = query.Encode()
	req, err := http.NewRequest(http.MethodGet, requestURL.String(), nil)
	if err != nil {
		return errors.New("无法创建 TMDB 测试请求")
	}
	req.Header.Set("Accept", "application/json")
	response, err := client.Do(req)
	if err != nil {
		if request.UseProxy {
			return errors.New("无法通过 HTTP 代理连接 TMDB API")
		}
		return errors.New("无法连接 TMDB API")
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
		return ErrInvalidTMDBAPIKey
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("TMDB 返回状态码 %d", response.StatusCode)
	}
	return nil
}

func (s *MediaMetadataService) Enrich(presentation *webhook.Presentation, integrations ...*model.AppIntegration) error {
	if presentation == nil {
		return nil
	}
	media, ok := presentation.Data["media"].(map[string]interface{})
	if !ok {
		return nil
	}
	now := time.Now()
	retentionDays := s.settings.RetentionDays()
	retentionCutoff := now.AddDate(0, 0, -retentionDays)
	defer func() {
		_ = database.GetDB().Where("expires_at <= ? OR updated_at <= ?", now, retentionCutoff).Delete(&model.MediaMetadataCache{}).Error
	}()

	if len(integrations) > 0 && integrations[0] != nil {
		integration := integrations[0]
		used, err := s.enrichFromMediaServer(context.Background(), integration, presentation, media, now, retentionCutoff, retentionDays)
		if used {
			webhook.RefreshMediaPresentation(presentation)
			return nil
		}
		if err != nil {
			appLogging.Warn("webhook", "媒体服务器信息获取失败，将尝试 TMDB 回退", appLogging.Fields{"app_code": integration.AppCode, "integration_id": integration.ID, "error": err})
		}
	}
	return s.enrichFromTMDB(presentation, media, now, retentionCutoff, retentionDays)
}

func (s *MediaMetadataService) enrichFromTMDB(presentation *webhook.Presentation, media map[string]interface{}, now, retentionCutoff time.Time, retentionDays int) error {
	lookup := metadataLookupForMedia(media)
	if lookup == nil {
		return nil
	}
	var metadata model.MediaMetadataCache
	result := database.GetDB().Where("cache_key = ? AND expires_at > ? AND updated_at > ?", lookup.cacheKey, now, retentionCutoff).Limit(1).Find(&metadata)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		applyMediaMetadata(media, &metadata)
		appendMetadataLink(presentation, metadata.SourceURL)
		return nil
	}
	apiKey := s.settings.TMDBAPIKey()
	if apiKey == "" {
		return nil
	}
	fetched, err := s.fetchTMDB(lookup, apiKey)
	if err != nil {
		return err
	}
	fetched.CacheKey = lookup.cacheKey
	fetched.Provider = "tmdb"
	fetched.ExternalID = lookup.externalID
	fetched.MediaType = lookup.mediaType
	fetched.ExpiresAt = now.AddDate(0, 0, retentionDays)
	if err := database.GetDB().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "cache_key"}},
		DoUpdates: clause.AssignmentColumns([]string{"provider", "external_id", "media_type", "title", "overview", "image_url", "source_url", "expires_at", "updated_at"}),
	}).Create(fetched).Error; err != nil {
		return err
	}
	applyMediaMetadata(media, fetched)
	appendMetadataLink(presentation, fetched.SourceURL)
	return nil
}

func (s *MediaMetadataService) enrichFromMediaServer(ctx context.Context, integration *model.AppIntegration, presentation *webhook.Presentation, media map[string]interface{}, now, retentionCutoff time.Time, retentionDays int) (bool, error) {
	adapter, ok := s.adapter(integration.AppCode)
	if !ok {
		return false, nil
	}
	settings, configured := mediaSettingsFromIntegration(integration)
	if !configured {
		return false, nil
	}
	itemID := stringFromMap(media, "id")
	if itemID == "" {
		return false, errors.New("Webhook 消息中没有媒体 Item ID")
	}
	itemHash := sha256.Sum256([]byte(itemID))
	cacheKey := fmt.Sprintf("%s:%d:%x", adapter.Code(), integration.ID, itemHash)
	var cached model.MediaMetadataCache
	result := database.GetDB().Where("cache_key = ? AND expires_at > ? AND updated_at > ?", cacheKey, now, retentionCutoff).Limit(1).Find(&cached)
	if result.Error != nil {
		return false, result.Error
	}
	if result.RowsAffected > 0 {
		if err := applyMediaServerMetadata(media, &cached); err != nil {
			return false, err
		}
		return true, nil
	}
	fetched, err := adapter.Fetch(ctx, settings, itemID)
	if err != nil {
		return false, err
	}
	metadataJSON, err := json.Marshal(fetched.Fields)
	if err != nil {
		return false, errors.New("无法缓存媒体服务器返回的数据")
	}
	item := &model.MediaMetadataCache{
		CacheKey: cacheKey, Provider: adapter.Code(), ExternalID: firstNonEmpty(fetched.ExternalID, itemID), MediaType: fetched.MediaType,
		Metadata: metadataJSON, ImageData: fetched.ImageData, ImageType: fetched.ImageType, ExpiresAt: now.AddDate(0, 0, retentionDays),
	}
	if title, ok := fetched.Fields["name"].(string); ok {
		item.Title = title
	}
	if overview, ok := fetched.Fields["overview"].(string); ok {
		item.Overview = overview
	}
	if len(item.ImageData) > 0 {
		var previous model.MediaMetadataCache
		previousResult := database.GetDB().Select("image_token").Where("cache_key = ?", cacheKey).Limit(1).Find(&previous)
		if previousResult.Error != nil {
			return false, previousResult.Error
		}
		item.ImageToken = previous.ImageToken
		if item.ImageToken == "" {
			item.ImageToken = mediaImageToken(integration.Secret, cacheKey)
		}
		item.ImageURL = "/api/public/media-images/" + item.ImageToken
	}
	if err := database.GetDB().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "cache_key"}},
		DoUpdates: clause.AssignmentColumns([]string{"provider", "external_id", "media_type", "title", "overview", "image_url", "image_token", "image_data", "image_type", "metadata", "expires_at", "updated_at"}),
	}).Create(item).Error; err != nil {
		return false, err
	}
	if err := applyMediaServerMetadata(media, item); err != nil {
		return false, err
	}
	return true, nil
}

func mediaImageToken(integrationSecret, cacheKey string) string {
	digest := hmac.New(sha256.New, []byte(integrationSecret))
	_, _ = digest.Write([]byte(cacheKey))
	return base64.RawURLEncoding.EncodeToString(digest.Sum(nil))
}

func applyMediaServerMetadata(media map[string]interface{}, cached *model.MediaMetadataCache) error {
	fields := map[string]interface{}{}
	if len(cached.Metadata) > 0 {
		if err := json.Unmarshal(cached.Metadata, &fields); err != nil {
			return errors.New("媒体服务器缓存数据格式无效")
		}
	}
	for key, value := range fields {
		media[key] = value
	}
	if cached.ImageURL != "" {
		media["image_url"] = cached.ImageURL
	}
	media["metadata_source"] = cached.Provider
	return nil
}

func (s *MediaMetadataService) CachedImage(token string) (*model.MediaMetadataCache, error) {
	if len(token) < 32 || len(token) > 100 {
		return nil, gorm.ErrRecordNotFound
	}
	var item model.MediaMetadataCache
	result := database.GetDB().Select("image_data", "image_type", "expires_at").Where("image_token = ? AND expires_at > ?", token, time.Now()).Limit(1).Find(&item)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 || len(item.ImageData) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &item, nil
}

func metadataLookupForMedia(media map[string]interface{}) *mediaMetadataLookup {
	providers, _ := media["provider_ids"].(map[string]interface{})
	tmdbID := stringFromMap(providers, "tmdb")
	if !digits(tmdbID) {
		return nil
	}
	mediaType := strings.ToLower(stringFromMap(media, "type"))
	if mediaType == "movie" || mediaType == "series" {
		kind := "movie"
		if mediaType == "series" {
			kind = "tv"
		}
		return &mediaMetadataLookup{cacheKey: "tmdb:" + kind + ":" + tmdbID, externalID: tmdbID, mediaType: mediaType, path: "/" + kind + "/" + tmdbID}
	}
	if mediaType != "episode" && mediaType != "season" {
		return nil
	}
	seriesName := stringFromMap(media, "series")
	seasonNumber, seasonErr := strconv.Atoi(stringFromMap(media, "season"))
	if seriesName == "" || seasonErr != nil || seasonNumber < 0 {
		return nil
	}
	lookup := &mediaMetadataLookup{cacheKey: "tmdb:" + mediaType + ":" + tmdbID, externalID: tmdbID, mediaType: mediaType, seriesName: seriesName, seriesYear: stringFromMap(media, "series_year"), seasonNumber: seasonNumber}
	if mediaType == "episode" {
		episodeNumber, err := strconv.Atoi(stringFromMap(media, "episode"))
		if err != nil || episodeNumber <= 0 {
			return nil
		}
		lookup.episodeNumber = episodeNumber
	}
	return lookup
}

func (s *MediaMetadataService) fetchTMDB(lookup *mediaMetadataLookup, apiKey string) (*model.MediaMetadataCache, error) {
	if lookup.path == "" {
		return s.fetchTMDBTVPart(lookup, apiKey)
	}
	var selected tmdbMedia
	if err := s.getTMDBJSON(lookup.path, nil, apiKey, &selected); err != nil {
		return nil, err
	}
	if selected.ID == 0 {
		return nil, errors.New("TMDB 未找到匹配的媒体信息")
	}
	return tmdbMetadataItem(selected, tmdbSourceURL(selected)), nil
}

func (s *MediaMetadataService) fetchTMDBTVPart(lookup *mediaMetadataLookup, apiKey string) (*model.MediaMetadataCache, error) {
	query := url.Values{"query": []string{lookup.seriesName}, "include_adult": []string{"false"}}
	if len(lookup.seriesYear) == 4 && digits(lookup.seriesYear) {
		query.Set("first_air_date_year", lookup.seriesYear)
	}
	var search tmdbTVSearchResponse
	if err := s.getTMDBJSON("/search/tv", query, apiKey, &search); err != nil {
		return nil, err
	}
	expectedID, _ := strconv.ParseInt(lookup.externalID, 10, 64)
	for index, series := range search.Results {
		if index >= 5 {
			break
		}
		path := fmt.Sprintf("/tv/%d/season/%d", series.ID, lookup.seasonNumber)
		if lookup.mediaType == "episode" {
			path += fmt.Sprintf("/episode/%d", lookup.episodeNumber)
		}
		var selected tmdbMedia
		err := s.getTMDBJSON(path, nil, apiKey, &selected)
		var apiErr *tmdbAPIError
		if errors.As(err, &apiErr) && apiErr.statusCode == http.StatusNotFound {
			continue
		}
		if err != nil {
			return nil, err
		}
		if selected.ID != expectedID {
			continue
		}
		if lookup.mediaType == "episode" {
			selected.ShowID = series.ID
			selected.SeasonNumber = lookup.seasonNumber
			selected.EpisodeNumber = lookup.episodeNumber
			return tmdbMetadataItem(selected, fmt.Sprintf("https://www.themoviedb.org/tv/%d/season/%d/episode/%d", series.ID, lookup.seasonNumber, lookup.episodeNumber)), nil
		}
		return tmdbMetadataItem(selected, fmt.Sprintf("https://www.themoviedb.org/tv/%d/season/%d", series.ID, lookup.seasonNumber)), nil
	}
	return nil, errors.New("TMDB 未找到与 Provider_tmdb 匹配的剧集信息")
}

func (s *MediaMetadataService) getTMDBJSON(path string, query url.Values, apiKey string, target interface{}) error {
	requestURL, err := url.Parse(strings.TrimRight(s.apiBaseURL, "/") + path)
	if err != nil {
		return err
	}
	values := requestURL.Query()
	for key, items := range query {
		for _, item := range items {
			values.Add(key, item)
		}
	}
	values.Set("language", "zh-CN")
	values.Set("api_key", apiKey)
	requestURL.RawQuery = values.Encode()
	req, err := http.NewRequest(http.MethodGet, requestURL.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	httpClient, err := s.tmdbHTTPClient()
	if err != nil {
		return err
	}
	response, err := httpClient.Do(req)
	if err != nil {
		return errors.New("请求 TMDB API 失败")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return &tmdbAPIError{statusCode: response.StatusCode}
	}
	return json.NewDecoder(response.Body).Decode(target)
}

func tmdbMetadataItem(selected tmdbMedia, sourceURL string) *model.MediaMetadataCache {
	imagePath := selected.PosterPath
	if imagePath == "" {
		imagePath = selected.StillPath
	}
	item := &model.MediaMetadataCache{Title: firstNonEmpty(selected.Title, selected.Name), Overview: selected.Overview}
	if strings.HasPrefix(imagePath, "/") {
		item.ImageURL = tmdbImageBaseURL + imagePath
	}
	item.SourceURL = sourceURL
	return item
}

func (s *MediaMetadataService) tmdbHTTPClient() (*http.Client, error) {
	if !s.settings.TMDBUsesHTTPProxy() {
		return s.httpClient, nil
	}
	proxyURL, err := url.Parse(s.settings.HTTPProxyURL())
	if err != nil || proxyURL.Hostname() == "" {
		return nil, errors.New("TMDB 已启用系统 HTTP 代理，但代理地址未配置")
	}
	return httpClientWithProxy(s.httpClient, proxyURL.String())
}

func httpClientWithProxy(base *http.Client, rawProxyURL string) (*http.Client, error) {
	proxyURL, err := url.Parse(rawProxyURL)
	if err != nil || proxyURL.Hostname() == "" {
		return nil, ErrInvalidHTTPProxyURL
	}
	client := *base
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if configured, ok := base.Transport.(*http.Transport); ok {
		transport = configured.Clone()
	}
	transport.Proxy = http.ProxyURL(proxyURL)
	client.Transport = transport
	return &client, nil
}

func tmdbSourceURL(item tmdbMedia) string {
	mediaType := item.MediaType
	if item.ShowID > 0 && item.SeasonNumber >= 0 && item.EpisodeNumber > 0 {
		return fmt.Sprintf("https://www.themoviedb.org/tv/%d/season/%d/episode/%d", item.ShowID, item.SeasonNumber, item.EpisodeNumber)
	}
	if mediaType == "movie" || item.Title != "" {
		return "https://www.themoviedb.org/movie/" + strconv.FormatInt(item.ID, 10)
	}
	return "https://www.themoviedb.org/tv/" + strconv.FormatInt(item.ID, 10)
}

func applyMediaMetadata(media map[string]interface{}, cached *model.MediaMetadataCache) {
	if stringFromMap(media, "overview") == "" && cached.Overview != "" {
		media["overview"] = cached.Overview
	}
	if stringFromMap(media, "image_url") == "" && cached.ImageURL != "" {
		media["image_url"] = cached.ImageURL
	}
	if cached.Title != "" {
		media["metadata_title"] = cached.Title
	}
	media["metadata_source"] = "tmdb"
	if cached.SourceURL != "" {
		media["metadata_source_url"] = cached.SourceURL
	}
}

func appendMetadataLink(presentation *webhook.Presentation, sourceURL string) {
	if sourceURL == "" {
		return
	}
	for _, link := range presentation.Links {
		if link["url"] == sourceURL {
			return
		}
	}
	presentation.Links = append(presentation.Links, map[string]string{"label": "TMDB", "url": sourceURL})
}

func stringFromMap(values map[string]interface{}, key string) string {
	value, _ := values[key].(string)
	return strings.TrimSpace(value)
}

func digits(value string) bool {
	if value == "" {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
