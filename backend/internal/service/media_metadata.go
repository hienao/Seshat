package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"seshat/internal/model"
	"seshat/internal/webhook"
	"seshat/pkg/database"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	defaultTMDBAPIBaseURL = "https://api.themoviedb.org/3"
	tmdbImageBaseURL      = "https://image.tmdb.org/t/p/w342"
)

type MediaMetadataService struct {
	apiBaseURL string
	httpClient *http.Client
	settings   *SettingService
}

type mediaMetadataLookup struct {
	cacheKey   string
	provider   string
	externalID string
	mediaType  string
	path       string
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

type tmdbFindResponse struct {
	MovieResults     []tmdbMedia `json:"movie_results"`
	TVResults        []tmdbMedia `json:"tv_results"`
	TVSeasonResults  []tmdbMedia `json:"tv_season_results"`
	TVEpisodeResults []tmdbMedia `json:"tv_episode_results"`
}

func NewMediaMetadataService() *MediaMetadataService {
	return &MediaMetadataService{
		apiBaseURL: defaultTMDBAPIBaseURL,
		httpClient: &http.Client{Timeout: 4 * time.Second},
		settings:   NewSettingService(),
	}
}

func (s *MediaMetadataService) Enrich(presentation *webhook.Presentation) error {
	if presentation == nil {
		return nil
	}
	media, ok := presentation.Data["media"].(map[string]interface{})
	if !ok {
		return nil
	}
	lookup := metadataLookupForMedia(media)
	if lookup == nil {
		return nil
	}

	now := time.Now()
	retentionDays := s.settings.RetentionDays()
	retentionCutoff := now.AddDate(0, 0, -retentionDays)
	var cached model.MediaMetadataCache
	err := database.GetDB().Where("cache_key = ? AND expires_at > ? AND updated_at > ?", lookup.cacheKey, now, retentionCutoff).First(&cached).Error
	if err == nil {
		applyMediaMetadata(media, &cached)
		appendMetadataLink(presentation, cached.SourceURL)
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	token := s.settings.TMDBReadAccessToken()
	if token == "" {
		return nil
	}
	fetched, err := s.fetchTMDB(lookup, token)
	if err != nil {
		return err
	}
	fetched.CacheKey = lookup.cacheKey
	fetched.Provider = lookup.provider
	fetched.ExternalID = lookup.externalID
	fetched.MediaType = lookup.mediaType
	fetched.ExpiresAt = now.AddDate(0, 0, retentionDays)
	if err := database.GetDB().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "cache_key"}},
		DoUpdates: clause.AssignmentColumns([]string{"provider", "external_id", "media_type", "title", "overview", "image_url", "source_url", "expires_at", "updated_at"}),
	}).Create(fetched).Error; err != nil {
		return err
	}
	_ = database.GetDB().Where("expires_at <= ? OR updated_at <= ?", now, retentionCutoff).Delete(&model.MediaMetadataCache{}).Error
	applyMediaMetadata(media, fetched)
	appendMetadataLink(presentation, fetched.SourceURL)
	return nil
}

func metadataLookupForMedia(media map[string]interface{}) *mediaMetadataLookup {
	providers, _ := media["provider_ids"].(map[string]interface{})
	mediaType := strings.ToLower(stringFromMap(media, "type"))
	if tmdbID := stringFromMap(providers, "tmdb"); digits(tmdbID) && (mediaType == "movie" || mediaType == "series") {
		kind := "movie"
		if mediaType == "series" {
			kind = "tv"
		}
		return &mediaMetadataLookup{cacheKey: "tmdb:" + kind + ":" + tmdbID, provider: "tmdb", externalID: tmdbID, mediaType: mediaType, path: "/" + kind + "/" + tmdbID}
	}
	if imdbID := stringFromMap(providers, "imdb"); validIMDbProviderID(imdbID) {
		return &mediaMetadataLookup{cacheKey: "imdb:" + imdbID + ":" + mediaType, provider: "imdb", externalID: imdbID, mediaType: mediaType, path: "/find/" + imdbID + "?external_source=imdb_id"}
	}
	if tvdbID := stringFromMap(providers, "tvdb"); digits(tvdbID) {
		return &mediaMetadataLookup{cacheKey: "tvdb:" + tvdbID + ":" + mediaType, provider: "tvdb", externalID: tvdbID, mediaType: mediaType, path: "/find/" + tvdbID + "?external_source=tvdb_id"}
	}
	return nil
}

func (s *MediaMetadataService) fetchTMDB(lookup *mediaMetadataLookup, token string) (*model.MediaMetadataCache, error) {
	separator := "?"
	if strings.Contains(lookup.path, "?") {
		separator = "&"
	}
	requestURL := strings.TrimRight(s.apiBaseURL, "/") + lookup.path + separator + "language=zh-CN"
	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	response, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TMDB 返回状态码 %d", response.StatusCode)
	}

	var selected tmdbMedia
	if strings.Contains(lookup.path, "/find/") {
		var result tmdbFindResponse
		if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
			return nil, err
		}
		selected = selectTMDBFindResult(result, lookup.mediaType)
	} else if err := json.NewDecoder(response.Body).Decode(&selected); err != nil {
		return nil, err
	}
	if selected.ID == 0 {
		return nil, errors.New("TMDB 未找到匹配的媒体信息")
	}

	imagePath := selected.PosterPath
	if imagePath == "" {
		imagePath = selected.StillPath
	}
	item := &model.MediaMetadataCache{Title: firstNonEmpty(selected.Title, selected.Name), Overview: selected.Overview}
	if strings.HasPrefix(imagePath, "/") {
		item.ImageURL = tmdbImageBaseURL + imagePath
	}
	item.SourceURL = tmdbSourceURL(selected)
	return item, nil
}

func selectTMDBFindResult(result tmdbFindResponse, mediaType string) tmdbMedia {
	var groups [][]tmdbMedia
	switch mediaType {
	case "movie":
		groups = [][]tmdbMedia{result.MovieResults, result.TVResults}
	case "series":
		groups = [][]tmdbMedia{result.TVResults, result.MovieResults}
	case "season":
		groups = [][]tmdbMedia{result.TVSeasonResults, result.TVResults}
	default:
		groups = [][]tmdbMedia{result.TVEpisodeResults, result.TVResults, result.MovieResults}
	}
	for _, group := range groups {
		if len(group) > 0 {
			return group[0]
		}
	}
	return tmdbMedia{}
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

func validIMDbProviderID(value string) bool {
	return strings.HasPrefix(value, "tt") && digits(value[2:])
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
