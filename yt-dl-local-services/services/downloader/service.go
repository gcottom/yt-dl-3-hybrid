package downloader

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/gcottom/go-zaplog"
	"github.com/gcottom/retry"
	"go.uber.org/zap"
)

func (s *Service) InitiateDownload(ctx context.Context, id string) error {
	s.StatusQueue <- StatusUpdate{ID: id, Status: StatusQueued}
	s.DownloadQueue <- id
	return nil
}

func (s *Service) DLQueueProcessor() {
	for {
		select {
		case id := <-s.DownloadQueue:
			if s.IsTrack(id) {
				s.DownloadLimiter.Acquire()
				go func(id string) {
					defer s.DownloadLimiter.Release()
					s.StatusQueue <- StatusUpdate{ID: id, Status: StatusDownloading}
					if _, err := retry.Retry(retry.NewAlgSimpleDefault(), 3, s.RunDownload, context.Background(), id); err != nil {
						s.StatusQueue <- StatusUpdate{ID: id, Status: StatusFailed}
						return
					}
					if _, err := retry.Retry(retry.NewAlgSimpleDefault(), 3, s.ProcessDownload, context.Background(), id); err != nil {
						s.StatusQueue <- StatusUpdate{ID: id, Status: StatusFailed}
						return
					}
					go s.ScheduledProcessingCallback(context.Background(), id)
				}(id)
			} else {
				go s.PlaylistProcessingCallback(context.Background(), id)
			}
		}
	}
}

func (s *Service) StatusProcessor() {
	for {
		select {
		case status := <-s.StatusQueue:
			s.StatusMap[status.ID] = status
		}
	}
}

func (s *Service) IsTrack(id string) bool {
	return len(id) == 11
}

func (s *Service) RunDownload(ctx context.Context, id string) error {
	return exec.Command("./downloader", fmt.Sprintf("-id=%s", id)).Run()
}

func (s *Service) ProcessDownload(ctx context.Context, id string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	filepath := fmt.Sprintf("%s/%s/%s", homeDir, tempDir, id)
	file, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer file.Close()
	var reqBody bytes.Buffer
	writer := multipart.NewWriter(&reqBody)
	part, err := writer.CreateFormFile("file", filepath)
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, file); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	writer.WriteField("id", id)
	req, err := s.HTTPClient.CreateRequest(http.MethodPost, fmt.Sprintf("http://%s/initiator", s.Config.LambdaDomain), reqBody.Bytes())
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	_, code, err := s.HTTPClient.DoRequest(req)
	if err != nil {
		return fmt.Errorf("failed to process download: %w", err)
	}
	if code != http.StatusOK {
		return fmt.Errorf("failed to process download: %d", code)
	}
	return nil
}

func (s *Service) ScheduledProcessingCallback(ctx context.Context, id string) {
	start := time.Now()
	for {
		s.StatusQueue <- StatusUpdate{ID: id, Status: StatusProcessing}
		if time.Since(start) > 900*time.Second {
			zaplog.ErrorC(ctx, "processing timed out", zap.String("id", id))
			s.StatusQueue <- StatusUpdate{ID: id, Status: StatusFailed}
			return
		}
		status, err := retry.Retry(retry.NewAlgSimpleDefault(), 3, s.GetProcessingStatus, ctx, id)
		if err != nil {
			zaplog.ErrorC(ctx, "failed to get status", zap.String("id", id), zap.Error(err))
			s.StatusQueue <- StatusUpdate{ID: id, Status: StatusFailed}
			return
		}
		if status[0] != nil && status[0].(*ProcessingStatus).Status == StatusComplete {
			s.SaveFileLimiter.Acquire()
			defer s.SaveFileLimiter.Release()
			if _, err := retry.Retry(retry.NewAlgSimpleDefault(), 3, s.SaveProcessedFile, ctx, id, status[0].(*ProcessingStatus).FileURL); err != nil {
				zaplog.ErrorC(ctx, "failed to save processed file", zap.String("id", id), zap.Error(err))
				s.StatusQueue <- StatusUpdate{ID: id, Status: StatusFailed}
				return
			}
		}
		if status[0] != nil {
			s.StatusQueue <- StatusUpdate{ID: id, Status: status[0].(*ProcessingStatus).Status}
		}
		if status[0] != nil && status[0].(*ProcessingStatus).Status == StatusComplete || status[0].(*ProcessingStatus).Status == StatusFailed {
			return
		}
		time.Sleep(10 * time.Second)
	}
}

func (s *Service) PlaylistProcessingCallback(ctx context.Context, id string) {
	s.StatusQueue <- StatusUpdate{ID: id, Status: StatusQueued}
	entries, err := s.YoutubeClient.GetPlaylistEntries(ctx, id)
	if err != nil {
		zaplog.ErrorC(ctx, "failed to get playlist entries", zap.String("id", id), zap.Error(err))
		return
	}
	for _, entry := range entries {
		s.DownloadQueue <- entry
	}
	s.StatusQueue <- StatusUpdate{ID: id, Status: StatusDownloading, PlaylistTrackCount: len(entries)}
	for {
		isProcesssing := false
		countDone := 0
		for _, entry := range entries {
			if s.StatusMap[entry].Status != StatusComplete && s.StatusMap[entry].Status != StatusFailed {
				isProcesssing = true
			} else {
				countDone++
			}
		}
		s.StatusQueue <- StatusUpdate{ID: id, Status: StatusProcessing, PlaylistTrackCount: len(entries), PlaylistTrackDone: countDone}
		if !isProcesssing {
			s.StatusQueue <- StatusUpdate{ID: id, Status: StatusComplete}
			return
		}
	}
}

func (s *Service) GetStatus(ctx context.Context, id string) (*StatusUpdate, error) {
	if status, ok := s.StatusMap[id]; ok {
		return &status, nil
	}
	return nil, nil
}

func (s *Service) GetProcessingStatus(ctx context.Context, id string) (*ProcessingStatus, error) {
	req, err := s.HTTPClient.CreateRequest(http.MethodGet, fmt.Sprintf("http://%s/status?id=%s", s.Config.LambdaDomain, id), nil)
	if err != nil {
		return nil, err
	}
	resp, code, err := s.HTTPClient.DoRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get processing status: %w", err)
	}
	if code != http.StatusOK {
		return nil, fmt.Errorf("failed to get processing status: %d", code)
	}
	var status ProcessingStatus
	if err := json.Unmarshal(resp, &status); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return &status, nil
}

func (s *Service) SaveProcessedFile(ctx context.Context, id string, url string) error {
	req, err := s.HTTPClient.CreateRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, code, err := s.HTTPClient.DoRequest(req)
	if err != nil {
		return fmt.Errorf("failed to get processing status: %w", err)
	}
	if code != http.StatusOK {
		return fmt.Errorf("failed to get processing status: %d", code)
	}
	return os.WriteFile(fmt.Sprintf("%s/%s", s.Config.SaveDir, id), resp, 0644)
}
