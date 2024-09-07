package meta

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"net/http"
	"regexp"
	"strings"

	"github.com/gcottom/go-zaplog"
	"github.com/gcottom/mp3meta"
	"github.com/gcottom/retry"
	"github.com/gcottom/yt-dl-3-hybrid/yt-dl-lambda/yt-dl-lambda-go/service/aws/dynamodb"
	"github.com/gcottom/yt-dl-3-hybrid/yt-dl-lambda/yt-dl-lambda-go/service/aws/s3"
	"go.uber.org/zap"
)

func (s *Service) SaveMeta(ctx context.Context, data []byte, id string, genre string) error {
	tag, err := mp3meta.ParseMP3(bytes.NewReader(data))
	if err != nil {
		zaplog.ErrorC(ctx, "failed to read mp3", zap.Error(err))
		return err
	}
	track, err := s.DBClient.GetTrackByID(ctx, id)
	if err != nil {
		zaplog.ErrorC(ctx, "failed to get track", zap.Error(err))
		return err
	}
	tag.SetTitle(track.Title)
	tag.SetArtist(track.Artist)
	tag.SetAlbum(track.Album)
	tag.SetGenre(genre)
	if track.CoverArtURL != "" {
		response, err := http.Get(track.CoverArtURL)
		if err != nil {
			zaplog.ErrorC(ctx, "failed to get cover art", zap.Error(err))
			return err
		}
		defer response.Body.Close()
		img, _, err := image.Decode(response.Body)
		if err != nil {
			zaplog.ErrorC(ctx, "failed to decode cover art", zap.Error(err))
			return err
		}
		tag.SetCoverArt(&img)
	}
	output := new(bytes.Buffer)
	if err := tag.Save(output); err != nil {
		zaplog.ErrorC(ctx, "failed to save tag", zap.Error(err))
		return err
	}
	fileName := s.SanitizeFilename(fmt.Sprintf("%s - %s.mp3", track.Artist, track.Title))
	if _, err = retry.Retry(retry.NewAlgSimpleDefault(), 3, s3.UploadToS3, output, fileName, s3.YTDLS3Bucket); err != nil {
		zaplog.ErrorC(ctx, "failed to upload to s3", zap.Error(err))
		return err
	}
	if _, err = retry.Retry(retry.NewAlgSimpleDefault(), 3, s.DBClient.PutTrack, ctx,
		&dynamodb.DBTrack{ID: id, Status: dynamodb.StatusComplete, URL: fileName, Title: track.Title, Artist: track.Artist, Album: track.Album}); err != nil {
		zaplog.ErrorC(ctx, "failed to update dynamodb", zap.Error(err))
		return err
	}
	return nil
}

func (s *Service) SanitizeFilename(str string) string {
	regex := regexp.MustCompile(`[\\/:*?"<>|\x00-\x1F]`)
	safeStr := regex.ReplaceAllString(str, "_")
	return strings.Trim(safeStr, " .")
}
