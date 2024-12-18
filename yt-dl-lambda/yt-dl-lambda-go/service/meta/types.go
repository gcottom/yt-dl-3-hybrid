package meta

import (
	"github.com/gcottom/yt-dl-3-hybrid/yt-dl-lambda/yt-dl-lambda-go/pkg/http_client"
	"github.com/gcottom/yt-dl-3-hybrid/yt-dl-lambda/yt-dl-lambda-go/service/aws/dynamodb"
	"golang.org/x/oauth2/clientcredentials"
)

type MetaService interface {
}

type Service struct {
	HTTPClient    *http_client.HTTPClient
	SpotifyConfig *clientcredentials.Config
	DBClient      *dynamodb.DynamoClient
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
