package downloader

import (
	"context"

	"github.com/gcottom/semaphore"
	"github.com/gcottom/yt-dl-3-hybrid/yd-dl-local-service/config"
	"github.com/gcottom/yt-dl-3-hybrid/yd-dl-local-service/pkg/http_client"
	"github.com/gcottom/yt-dl-3-hybrid/yd-dl-local-service/pkg/youtube_v2"
)

type DownloaderService interface {
	InitiateDownload(ctx context.Context, id string) error
	GetStatus(ctx context.Context, id string) (*StatusUpdate, error)
}

type Service struct {
	Config          *config.Config
	HTTPClient      *http_client.HTTPClient
	DownloadLimiter *semaphore.Semaphore
	SaveFileLimiter *semaphore.Semaphore
	DownloadQueue   chan string
	StatusQueue     chan StatusUpdate
	StatusMap       map[string]StatusUpdate
	YoutubeClient   youtube_v2.YoutubeClient
}

func NewDownloaderService(cfg *config.Config, httpClient *http_client.HTTPClient) *Service {
	return &Service{
		Config:          cfg,
		HTTPClient:      httpClient,
		DownloadLimiter: semaphore.NewSemaphore(cfg.ConcurrentDownloads),
		SaveFileLimiter: semaphore.NewSemaphore(cfg.ConcurrentDownloads),
		DownloadQueue:   make(chan string, 5000),
		StatusQueue:     make(chan StatusUpdate, 5000),
		StatusMap:       make(map[string]StatusUpdate),
		YoutubeClient:   youtube_v2.NewYoutubeClient(cfg, httpClient),
	}
}

type StatusUpdate struct {
	ID                 string `json:"id"`
	Status             string `json:"status"`
	PlaylistTrackCount int    `json:"playlist_track_count,omitempty"`
	PlaylistTrackDone  int    `json:"playlist_track_done,omitempty"`
}

type ProcessingStatus struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	FileURL string `json:"file_url"`
}

const (
	StatusQueued      = "queued"
	StatusDownloading = "downloading"
	StatusProcessing  = "processing"
	StatusComplete    = "complete"
	StatusFailed      = "failed"
)

const tempDir = "ytdl-temp"
