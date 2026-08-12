package service

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
)

type embyMediaMetadataAdapter struct{ client *http.Client }

func newEmbyMediaMetadataAdapter() mediaServerMetadataAdapter {
	return &embyMediaMetadataAdapter{client: newMediaServerHTTPClient()}
}

func (a *embyMediaMetadataAdapter) Code() string { return "emby" }

func (a *embyMediaMetadataAdapter) Test(ctx context.Context, settings mediaServerSettings) error {
	var systemInfo map[string]interface{}
	return mediaServerJSON(ctx, a.client, settings, a.path(settings.ServerURL, "/System/Info"), &systemInfo)
}

func (a *embyMediaMetadataAdapter) Fetch(ctx context.Context, settings mediaServerSettings, itemID string) (*mediaServerMetadata, error) {
	var item mediaServerItem
	itemPath := a.path(settings.ServerURL, "/Items/"+url.PathEscape(itemID))
	if err := mediaServerJSON(ctx, a.client, settings, itemPath, &item); err != nil {
		return nil, err
	}
	if item.ID == "" {
		return nil, errors.New("Emby 未返回有效的媒体信息")
	}
	var imageData []byte
	var imageType string
	if item.ImageTags["Primary"] != "" {
		var err error
		imageData, imageType, err = mediaServerImage(ctx, a.client, settings, itemPath+"/Images/Primary?maxWidth=640&quality=90")
		if err != nil {
			return nil, err
		}
	}
	return metadataFromMediaServerItem(item, imageData, imageType), nil
}

func (a *embyMediaMetadataAdapter) path(serverURL, path string) string {
	if strings.HasSuffix(strings.ToLower(strings.TrimRight(serverURL, "/")), "/emby") {
		return path
	}
	return "/emby" + path
}
