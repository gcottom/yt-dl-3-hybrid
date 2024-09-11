package downloader

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"sync"
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
			if status.ShouldCallback {
				status.Callback(s.StatusMap[status.ID])
				continue
			}
			s.StatusMap[status.ID] = status
			if status.Status == StatusComplete || status.Status == StatusFailed {
				zaplog.InfoC(context.Background(), "final status for download", zap.String("id", status.ID), zap.String("status", status.Status))
			}

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
	path := fmt.Sprintf("%s/%s", s.Config.TempDir, id)
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer os.Remove(path)
	defer file.Close()
	reqBody, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	req, err := s.HTTPClient.CreateRequest(http.MethodGet, fmt.Sprintf("https://%s/s3signer?id=%s", s.Config.LambdaDomain, id), nil)
	if err != nil {
		return err
	}
	resp, code, err := s.HTTPClient.DoRequest(req)
	if err != nil {
		return fmt.Errorf("failed to get signed URL: %w", err)
	}
	if code != http.StatusOK {
		return fmt.Errorf("failed to get signed URL: %d", code)
	}
	var data struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(resp, &data); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}
	zaplog.Info("uploading file", zap.String("filepath", path), zap.String("id", id))
	req, err = s.HTTPClient.CreateOctetStreamRequest(http.MethodPut, data.URL, reqBody)
	if err != nil {
		fmt.Println(err)
		return err
	}
	_, code, err = s.HTTPClient.DoRequest(req)
	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}
	if code != http.StatusOK {
		return fmt.Errorf("failed to upload file: %d", code)
	}
	trackMeta, err := s.MetaServiceClient.GetBestMeta(ctx, id)
	if err != nil {
		return err
	}
	trackMeta.ID = id
	jsonData, err := json.Marshal(trackMeta)
	if err != nil {
		return err
	}
	req, err = s.HTTPClient.CreateRequest(http.MethodPost, fmt.Sprintf("https://%s/initiator", s.Config.LambdaDomain), jsonData)
	if err != nil {
		return err
	}
	res, code, err := s.HTTPClient.DoRequest(req)
	if err != nil {
		zaplog.Error("failed to initiate processing", zap.Error(err))
		return fmt.Errorf("failed to initiate processing: %w", err)
	}
	if code != http.StatusOK {
		zaplog.Error("failed to initiate processing", zap.Int("code", code), zap.String("response", string(res)))
		return fmt.Errorf("failed to initiate processing: %d", code)
	}
	return nil
}

func (s *Service) ScheduledProcessingCallback(ctx context.Context, id string) {
	start := time.Now()
	for {
		s.StatusQueue <- StatusUpdate{ID: id, Status: StatusProcessing}
		if time.Since(start) > 3600*time.Second {
			zaplog.ErrorC(ctx, "processing timed out", zap.String("id", id))
			s.StatusQueue <- StatusUpdate{ID: id, Status: StatusFailed}
			return
		}
		zaplog.InfoC(ctx, "processing callback running - getting processing status", zap.String("id", id))
		status, err := retry.Retry(retry.NewAlgSimpleDefault(), 3, s.GetProcessingStatus, ctx, id)
		if err != nil {
			zaplog.ErrorC(ctx, "failed to get status", zap.String("id", id), zap.Error(err))
			s.StatusQueue <- StatusUpdate{ID: id, Status: StatusFailed}
			return
		}
		if status[0] != nil && status[0].(*ProcessingStatus).Status == StatusComplete {
			zaplog.InfoC(ctx, "processing callback running - got processing status", zap.String("id", id), zap.String("status", status[0].(*ProcessingStatus).Status))
			s.SaveFileLimiter.Acquire()
			defer s.SaveFileLimiter.Release()
			if _, err := retry.Retry(retry.NewAlgSimpleDefault(), 3, s.SaveProcessedFile, ctx, status[0].(*ProcessingStatus).FileName, status[0].(*ProcessingStatus).FileURL); err != nil {
				zaplog.ErrorC(ctx, "failed to save processed file", zap.String("id", id), zap.Error(err))
				s.StatusQueue <- StatusUpdate{ID: id, Status: StatusFailed}
				return
			}
		}
		if status[0] != nil {
			zaplog.InfoC(ctx, "processing callback running - got processing status", zap.String("id", id), zap.String("status", status[0].(*ProcessingStatus).Status))
			s.StatusQueue <- StatusUpdate{ID: id, Status: status[0].(*ProcessingStatus).Status}
		}
		if status[0] != nil && status[0].(*ProcessingStatus).Status == StatusComplete || status[0].(*ProcessingStatus).Status == StatusFailed {
			zaplog.InfoC(ctx, "processing callback exiting", zap.String("id", id), zap.String("status", status[0].(*ProcessingStatus).Status))
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
		wg := sync.WaitGroup{}
		for _, entry := range entries {
			wg.Add(1)
			s.StatusQueue <- StatusUpdate{ID: entry, ShouldCallback: true, Callback: func(stat StatusUpdate) {
				if stat.Status != StatusComplete && stat.Status != StatusFailed {
					isProcesssing = true
				} else {
					countDone++
				}
				wg.Done()
			}}
		}
		wg.Wait()
		s.StatusQueue <- StatusUpdate{ID: id, Status: StatusProcessing, PlaylistTrackCount: len(entries), PlaylistTrackDone: countDone}
		if !isProcesssing {
			s.StatusQueue <- StatusUpdate{ID: id, Status: StatusComplete}
			return
		}
		time.Sleep(10 * time.Second)
	}
}

func (s *Service) GetStatus(ctx context.Context, id string) (*StatusUpdate, error) {
	if status, ok := s.StatusMap[id]; ok {
		return &status, nil
	}
	return nil, nil
}

func (s *Service) GetProcessingStatus(ctx context.Context, id string) (*ProcessingStatus, error) {
	req, err := s.HTTPClient.CreateRequest(http.MethodGet, fmt.Sprintf("https://%s/status?id=%s", s.Config.LambdaDomain, id), nil)
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

func (s *Service) SaveProcessedFile(ctx context.Context, name string, url string) error {
	zaplog.InfoC(ctx, "requesting processed file", zap.String("name", name))
	req, err := s.HTTPClient.CreateRequest(http.MethodGet, url, nil)
	if err != nil {
		zaplog.ErrorC(ctx, "failed to create request", zap.Error(err))
		return err
	}
	resp, code, err := s.HTTPClient.DoRequest(req)
	if err != nil {
		zaplog.ErrorC(ctx, "failed to get processed file", zap.Error(err))
		return fmt.Errorf("failed to get processed file: %w", err)
	}
	if code != http.StatusOK {
		return fmt.Errorf("failed to get processed file, code: %d", code)
	}
	zaplog.InfoC(ctx, "retrieved processed file", zap.String("name", name))
	zaplog.InfoC(ctx, "saving processed file", zap.String("name", name))
	if err = os.Mkdir(s.Config.SaveDir, 0755); err != nil && !os.IsExist(err) {
		panic(err)
	}
	return os.WriteFile(fmt.Sprintf("%s/%s", s.Config.SaveDir, name), resp, 0644)
}
