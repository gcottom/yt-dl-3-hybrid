package youtube_v2

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gcottom/go-zaplog"
	"github.com/kkdai/youtube/v2"
	"go.uber.org/zap"
)

func (s *Client) Download(ctx context.Context, id string, useEmbedded bool) ([]byte, error) {
	zaplog.InfoC(ctx, "fetching video info", zap.String("id", id))
	var videoInfo *youtube.Video
	var err error
	if useEmbedded {
		videoInfo, err = s.YTEmbeddedClient.GetVideo(id)
	} else {
		videoInfo, err = s.YTClient.GetVideo(id)
	}
	if err != nil {
		zaplog.ErrorC(ctx, "failed to get video info", zap.String("id", id), zap.Error(err))
		if !useEmbedded {
			zaplog.InfoC(ctx, "retrying with embedded client", zap.String("id", id))
			return s.Download(ctx, id, true)
		}
		return nil, fmt.Errorf("failed to get video info: %w", err)
	}
	zaplog.InfoC(ctx, "video info fetched", zap.String("id", id))
	zaplog.InfoC(ctx, "getting best audio format", zap.String("id", id))
	bestFormat := getBestAudioFormat(videoInfo.Formats.Type("audio"))
	if bestFormat == nil {
		zaplog.ErrorC(ctx, "failed to get best audio format", zap.String("id", id))
		return nil, fmt.Errorf("failed to get best audio format")
	}
	zaplog.InfoC(ctx, "best audio format found", zap.String("id", id), zap.Int("bitrate", bestFormat.Bitrate))

	zaplog.InfoC(ctx, "downloading youtube stream", zap.String("id", id))
	var stream io.ReadCloser
	if useEmbedded {
		stream, _, err = s.YTEmbeddedClient.GetStreamContext(ctx, videoInfo, bestFormat)
	} else {
		stream, _, err = s.YTClient.GetStreamContext(ctx, videoInfo, bestFormat)
	}
	if err != nil {
		zaplog.ErrorC(ctx, "failed to get stream", zap.String("id", id), zap.Error(err))
		if !useEmbedded {
			zaplog.InfoC(ctx, "retrying with embedded client", zap.String("id", id))
			return s.Download(ctx, id, true)
		}
		return nil, fmt.Errorf("failed to get stream: %w", err)
	}
	defer stream.Close()
	streamBytes, err := io.ReadAll(stream)
	if err != nil {
		zaplog.ErrorC(ctx, "failed to read stream", zap.String("id", id), zap.Error(err))
		return nil, fmt.Errorf("failed to read stream: %w", err)
	}
	zaplog.InfoC(ctx, "successfully downloaded youtube stream", zap.String("id", id))
	return streamBytes, nil
}

func (s *Client) GetPlaylistEntries(ctx context.Context, playlistID string) ([]string, error) {
	zaplog.InfoC(ctx, "getting playlist entries", zap.String("playlistID", playlistID))
	playlist, err := s.YTClient.GetPlaylist(playlistID)
	if err != nil {
		zaplog.ErrorC(ctx, "failed to get playlist entries", zap.String("playlistID", playlistID), zap.Error(err))
		return s.GetPlaylistEntriesFromMusicAPI(ctx, playlistID)
	}
	entries := make([]string, 0)
	for _, entry := range playlist.Videos {
		entries = append(entries, entry.ID)
	}
	zaplog.InfoC(ctx, "successfully retrieved playlist entries", zap.String("playlistID", playlistID), zap.Int("count", len(entries)))
	return entries, nil
}

func (s *Client) GetPlaylistEntriesFromMusicAPI(ctx context.Context, playlistID string) ([]string, error) {
	zaplog.InfoC(ctx, "getting playlist entries from music API", zap.String("playlistID", playlistID))
	req, err := s.HTTPClient.CreateRequest(http.MethodGet, fmt.Sprintf("http://%s/playlist?id=%s", s.Config.LambdaDomain, playlistID), nil)
	if err != nil {
		zaplog.ErrorC(ctx, "failed to create request", zap.Error(err))
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	resp, code, err := s.HTTPClient.DoRequest(req)
	if err != nil {
		zaplog.ErrorC(ctx, "failed to do request", zap.Error(err))
		return nil, fmt.Errorf("failed to do request: %w", err)
	}
	if code != http.StatusOK {
		zaplog.ErrorC(ctx, "failed to get playlist entries from music API", zap.Int("code", code))
		return nil, fmt.Errorf("failed to get playlist entries from music API: %d", code)
	}
	var data struct {
		Tracks []struct {
			ID string `json:"id"`
		} `json:"tracks"`
	}
	if err := json.Unmarshal(resp, &data); err != nil {
		zaplog.ErrorC(ctx, "failed to unmarshal response", zap.Error(err))
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	entries := make([]string, 0)
	for _, entry := range data.Tracks {
		entries = append(entries, entry.ID)
	}
	zaplog.InfoC(ctx, "successfully retrieved playlist entries from music API", zap.String("playlistID", playlistID), zap.Int("count", len(entries)))
	return entries, nil
}

// GetVideoInfo returns the title and author of a video
func (s *Client) GetVideoInfo(ctx context.Context, videoID string, useEmbedded bool) (string, string, error) {
	zaplog.InfoC(ctx, "getting video info", zap.String("videoID", videoID))
	var video *youtube.Video
	var err error
	if useEmbedded {
		video, err = s.YTEmbeddedClient.GetVideoContext(ctx, videoID)
	} else {
		video, err = s.YTClient.GetVideoContext(ctx, videoID)
	}
	if err != nil {
		zaplog.ErrorC(ctx, "failed to get video info", zap.String("videoID", videoID), zap.Error(err))
		if !useEmbedded {
			zaplog.InfoC(ctx, "retrying with embedded client", zap.String("videoID", videoID))
			return s.GetVideoInfo(ctx, videoID, true)
		}
		return "", "", fmt.Errorf("failed to get video info: %w", err)
	}
	zaplog.InfoC(ctx, "successfully retrieved video info", zap.String("videoID", videoID))
	return video.Title, video.Author, nil
}

func getBestAudioFormat(formats youtube.FormatList) *youtube.Format {
	var bestFormat *youtube.Format
	maxBitrate := 0
	for _, format := range formats {
		if format.Bitrate > maxBitrate {
			best := format
			bestFormat = &best
			maxBitrate = format.Bitrate
		}
	}
	return bestFormat
}
