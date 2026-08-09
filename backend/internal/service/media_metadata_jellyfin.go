package service

import (
	"context"
	"errors"
	"net/http"
	"net/url"
)

type jellyfinMediaMetadataAdapter struct{ client *http.Client }

func newJellyfinMediaMetadataAdapter() mediaServerMetadataAdapter {
	return &jellyfinMediaMetadataAdapter{client: newMediaServerHTTPClient()}
}

func (a *jellyfinMediaMetadataAdapter) Code() string { return "jellyfin" }

func (a *jellyfinMediaMetadataAdapter) Test(ctx context.Context, settings mediaServerSettings) error {
	var systemInfo map[string]interface{}
	return mediaServerJSON(ctx, a.client, settings, "/System/Info", &systemInfo)
}

func (a *jellyfinMediaMetadataAdapter) Fetch(ctx context.Context, settings mediaServerSettings, itemID string) (*mediaServerMetadata, error) {
	var item mediaServerItem
	itemPath := "/Items/" + url.PathEscape(itemID)
	if err := mediaServerJSON(ctx, a.client, settings, itemPath, &item); err != nil {
		return nil, err
	}
	if item.ID == "" {
		return nil, errors.New("Jellyfin 未返回有效的媒体信息")
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
