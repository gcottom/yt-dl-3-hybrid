package youtube_v2

import (
	"context"

	"github.com/gcottom/yt-dl-3-hybrid/yd-dl-local-services/yt-dl-local-services-go/config"
	"github.com/gcottom/yt-dl-3-hybrid/yd-dl-local-services/yt-dl-local-services-go/pkg/http_client"
	"github.com/kkdai/youtube/v2"
)

type YoutubeClient interface {
	Download(ctx context.Context, id string, useEmbedded bool) ([]byte, error)
	GetPlaylistEntries(ctx context.Context, playlistID string) ([]string, error)
	GetVideoInfo(ctx context.Context, videoID string, useEmbedded bool) (string, string, error)
}

type Client struct {
	Config           *config.Config
	HTTPClient       *http_client.HTTPClient
	YTClient         *youtube.Client
	YTEmbeddedClient *youtube.Client
}

func NewYoutubeClient(config *config.Config, httpClient *http_client.HTTPClient) *Client {
	youtube.DefaultClient = youtube.IOSClient
	embeddedClient := &youtube.Client{HTTPClient: httpClient.Client}
	embeddedClient.GetVideo("0P19rsu3jXY")
	youtube.DefaultClient = youtube.IOSClient
	androidClient := &youtube.Client{HTTPClient: httpClient.Client}
	androidClient.GetVideo("0P19rsu3jXY")
	return &Client{
		Config:           config,
		HTTPClient:       httpClient,
		YTClient:         androidClient,
		YTEmbeddedClient: embeddedClient,
	}
}
