package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const maxMediaServerImageBytes = 5 << 20

var (
	ErrMediaAPIUnsupported = errors.New("当前 App 不支持媒体 API 配置")
	ErrInvalidMediaAPIURL  = errors.New("媒体服务器地址必须是有效的 http:// 或 https:// URL")
	ErrInvalidMediaAPIKey  = errors.New("媒体服务器 API Key 不能为空")
)

type mediaServerSettings struct {
	ServerURL string
	APIKey    string
}

type mediaServerMetadata struct {
	Fields     map[string]interface{}
	ImageData  []byte
	ImageType  string
	ExternalID string
	MediaType  string
}

type mediaServerMetadataAdapter interface {
	Code() string
	Test(context.Context, mediaServerSettings) error
	Fetch(context.Context, mediaServerSettings, string) (*mediaServerMetadata, error)
}

type mediaServerItem struct {
	ID                string            `json:"Id"`
	Name              string            `json:"Name"`
	Type              string            `json:"Type"`
	SeriesName        string            `json:"SeriesName"`
	ParentIndexNumber *int              `json:"ParentIndexNumber"`
	IndexNumber       *int              `json:"IndexNumber"`
	ProductionYear    *int              `json:"ProductionYear"`
	Overview          string            `json:"Overview"`
	RunTimeTicks      int64             `json:"RunTimeTicks"`
	ProviderIDs       map[string]string `json:"ProviderIds"`
	ImageTags         map[string]string `json:"ImageTags"`
}

func normalizeMediaServerSettings(serverURL, apiKey string) (mediaServerSettings, error) {
	normalizedURL, err := normalizeMediaServerURL(serverURL)
	if err != nil {
		return mediaServerSettings{}, err
	}
	key := strings.TrimSpace(apiKey)
	if key == "" || len(key) > 2000 {
		return mediaServerSettings{}, ErrInvalidMediaAPIKey
	}
	return mediaServerSettings{ServerURL: normalizedURL, APIKey: key}, nil
}

func normalizeMediaServerURL(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", ErrInvalidMediaAPIURL
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	parsed.RawPath = ""
	return strings.TrimRight(parsed.String(), "/"), nil
}

func mediaServerEndpoint(baseURL, path string) string {
	return strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(path, "/")
}

func mediaServerJSON(ctx context.Context, client *http.Client, settings mediaServerSettings, path string, target interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, mediaServerEndpoint(settings.ServerURL, path), nil)
	if err != nil {
		return errors.New("无法创建媒体服务器请求")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Emby-Token", settings.APIKey)
	response, err := client.Do(req)
	if err != nil {
		return errors.New("无法连接媒体服务器")
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
		return ErrInvalidMediaAPIKey
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("媒体服务器返回状态码 %d", response.StatusCode)
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(target); err != nil {
		return errors.New("媒体服务器返回的数据格式无效")
	}
	return nil
}

func mediaServerImage(ctx context.Context, client *http.Client, settings mediaServerSettings, path string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, mediaServerEndpoint(settings.ServerURL, path), nil)
	if err != nil {
		return nil, "", errors.New("无法创建媒体图片请求")
	}
	req.Header.Set("Accept", "image/*")
	req.Header.Set("X-Emby-Token", settings.APIKey)
	response, err := client.Do(req)
	if err != nil {
		return nil, "", errors.New("无法获取媒体图片")
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return nil, "", nil
	}
	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
		return nil, "", ErrInvalidMediaAPIKey
	}
	if response.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("媒体图片接口返回状态码 %d", response.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, maxMediaServerImageBytes+1))
	if err != nil {
		return nil, "", errors.New("读取媒体图片失败")
	}
	if len(data) == 0 {
		return nil, "", nil
	}
	if len(data) > maxMediaServerImageBytes {
		return nil, "", errors.New("媒体图片超过 5 MiB 限制")
	}
	contentType := strings.TrimSpace(strings.Split(response.Header.Get("Content-Type"), ";")[0])
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}
	if contentType != "image/jpeg" && contentType != "image/png" && contentType != "image/webp" && contentType != "image/gif" {
		return nil, "", errors.New("媒体图片格式不受支持")
	}
	return data, contentType, nil
}

func metadataFromMediaServerItem(item mediaServerItem, imageData []byte, imageType string) *mediaServerMetadata {
	fields := map[string]interface{}{}
	putMetadataString(fields, "id", item.ID)
	putMetadataString(fields, "name", item.Name)
	putMetadataString(fields, "type", item.Type)
	putMetadataString(fields, "series", item.SeriesName)
	putMetadataString(fields, "overview", item.Overview)
	if item.ParentIndexNumber != nil {
		fields["season"] = strconv.Itoa(*item.ParentIndexNumber)
	}
	if item.IndexNumber != nil {
		fields["episode"] = strconv.Itoa(*item.IndexNumber)
	}
	if item.ProductionYear != nil {
		fields["year"] = strconv.Itoa(*item.ProductionYear)
	}
	if item.RunTimeTicks > 0 {
		seconds := float64(item.RunTimeTicks) / 10_000_000
		fields["duration_seconds"] = seconds
		fields["duration_label"] = formatMetadataDuration(seconds)
	}
	providers := map[string]interface{}{}
	for key, value := range item.ProviderIDs {
		normalized := strings.ToLower(strings.TrimSpace(key))
		if (normalized == "tmdb" || normalized == "tvdb" || normalized == "imdb") && strings.TrimSpace(value) != "" {
			providers[normalized] = strings.TrimSpace(value)
		}
	}
	if len(providers) > 0 {
		fields["provider_ids"] = providers
	}
	return &mediaServerMetadata{Fields: fields, ImageData: imageData, ImageType: imageType, ExternalID: item.ID, MediaType: item.Type}
}

func putMetadataString(target map[string]interface{}, key, value string) {
	if value = strings.TrimSpace(value); value != "" {
		target[key] = value
	}
}

func formatMetadataDuration(seconds float64) string {
	total := int64(seconds + 0.5)
	if total < 0 {
		total = 0
	}
	if total >= 3600 {
		return fmt.Sprintf("%d:%02d:%02d", total/3600, total%3600/60, total%60)
	}
	return fmt.Sprintf("%d:%02d", total/60, total%60)
}

func newMediaServerHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 8 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return errors.New("媒体服务器重定向次数过多")
			}
			if len(via) > 0 && (req.URL.Scheme != via[0].URL.Scheme || !strings.EqualFold(req.URL.Host, via[0].URL.Host)) {
				return errors.New("媒体服务器禁止重定向到其他地址")
			}
			return nil
		},
	}
}
