package converter

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"

	"github.com/gcottom/go-zaplog"
	"github.com/gcottom/retry"
	"github.com/gcottom/yt-dl-3-hybrid/yt-dl-lambda/yt-dl-lambda-go/service/aws/s3"
	"go.uber.org/zap"
)

func Convert(input *os.File, id string) error {
	var args = []string{"-i", fmt.Sprintf("/tmp/%s.temp", id), "-c:a", "libmp3lame", "-b:a", "256k", "-f", "mp3", fmt.Sprintf("/tmp/%s-converted.mp3", id)}
	cmd := exec.Command(path.Join(os.Getenv("LAMBDA_TASK_ROOT"), "ffmpeg"), args...)

	if err := cmd.Start(); err != nil {
		zaplog.Error("Failed to start ffmpeg", zap.Error(err))
		return err
	}
	if err := cmd.Wait(); err != nil {
		zaplog.Error("Failed to wait for ffmpeg", zap.Error(err))
		return err
	}
	defer os.Remove(fmt.Sprintf("/tmp/%s-converted.mp3", id))
	data, err := os.ReadFile(fmt.Sprintf("/tmp/%s-converted.mp3", id))
	if err != nil {
		zaplog.Error("Failed to read file", zap.Error(err))
		return err
	}
	if _, err := retry.Retry(retry.NewAlgSimpleDefault(), 3, s3.UploadToS3,
		bytes.NewReader(data), fmt.Sprintf("%s.mp3", id), s3.YTDLS3Bucket); err != nil {
		zaplog.Error("Failed to upload to s3", zap.Error(err))
		return err
	}
	return nil
}
