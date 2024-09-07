package meta

import (
	"github.com/gcottom/yt-dl-3-hybrid/yd-dl-local-services/yt-dl-local-services-go/config"
	"github.com/gcottom/yt-dl-3-hybrid/yd-dl-local-services/yt-dl-local-services-go/pkg/http_client"
	"golang.org/x/oauth2/clientcredentials"
)

type Service struct {
	Config        *config.Config
	HTTPClient    *http_client.HTTPClient
	SpotifyConfig *clientcredentials.Config
}

type TrackMeta struct {
	ID          string `dynamodbav:"id" json:"id"`
	Status      string `dynamodbav:"status" json:"status,omitempty"`
	URL         string `dynamodbav:"url" json:"url,omitempty"`
	Title       string `dynamodbav:"title" json:"title"`
	Artist      string `dynamodbav:"artist" json:"artist"`
	Album       string `dynamodbav:"album" json:"album,omitempty"`
	CoverArtURL string `dynamodbav:"cover_art_url" json:"cover_art_url,omitempty"`
}

type YTMMetaResponse struct {
	Title  string `json:"title"`
	Author string `json:"author"`
	Image  string `json:"image"`
	Type   string `json:"type"`
}
