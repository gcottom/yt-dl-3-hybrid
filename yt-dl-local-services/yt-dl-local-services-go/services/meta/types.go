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
	Title       string
	Artist      string
	Album       string
	Genre       string
	CoverArtURL string
}

type YTMMetaResponse struct {
	Title  string `json:"title"`
	Author string `json:"author"`
	Image  string `json:"image"`
	Type   string `json:"type"`
}
