package service

import (
	"context"
	"errors"
	"net/http"
	"net/url"
)

type jellyfinMediaMetadataAdapter struct{ client *http.Client }

type jellyfinItemsResponse struct {
	Items []mediaServerItem `json:"Items"`
}

func newJellyfinMediaMetadataAdapter() mediaServerMetadataAdapter {
	return &jellyfinMediaMetadataAdapter{client: newMediaServerHTTPClient()}
}

func (a *jellyfinMediaMetadataAdapter) Code() string { return "jellyfin" }

func (a *jellyfinMediaMetadataAdapter) Test(ctx context.Context, settings mediaServerSettings) error {
	var systemInfo map[string]interface{}
	return mediaServerJSON(ctx, a.client, settings, "/System/Info", &systemInfo)
}

func (a *jellyfinMediaMetadataAdapter) Fetch(ctx context.Context, settings mediaServerSettings, itemID string) (*mediaServerMetadata, error) {
	query := url.Values{}
	query.Set("Ids", itemID)
	query.Set("Limit", "1")
	query.Set("EnableImages", "true")
	query.Set("EnableUserData", "false")
	var result jellyfinItemsResponse
	if err := mediaServerJSON(ctx, a.client, settings, "/Items?"+query.Encode(), &result); err != nil {
		return nil, err
	}
	if len(result.Items) == 0 || result.Items[0].ID == "" {
		return nil, errors.New("Jellyfin 未返回有效的媒体信息")
	}
	item := result.Items[0]
	itemPath := "/Items/" + url.PathEscape(item.ID)
	imageData, imageType, err := mediaServerImage(ctx, a.client, settings, itemPath+"/Images/Primary?maxWidth=640&quality=90")
	if err != nil {
		return nil, err
	}
	return metadataFromMediaServerItem(item, imageData, imageType), nil
}
