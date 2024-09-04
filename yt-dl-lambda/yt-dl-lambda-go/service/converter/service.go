package converter

import (
	"fmt"
	"os"
	"os/exec"
	"path"

	"github.com/gcottom/go-zaplog"
	"github.com/gcottom/retry"
	"github.com/gcottom/yt-dl-3-hybrid/yt-dl-lambda/yt-dl-lambda-go/service/aws/s3"
	"go.uber.org/zap"
)

func Convert(id string) error {
	var args = []string{"-i", fmt.Sprintf("/tmp/%s.temp", id), "-c:a", "libmp3lame", "-b:a", "256k", "-f", "mp3", fmt.Sprintf("/tmp/%s.mp3", id)}
	cmd := exec.Command(path.Join(os.Getenv("LAMBDA_TASK_ROOT"), "ffmpeg"), args...)

	zaplog.Info("converting file", zap.String("id", id))
	err := cmd.Start() // Start a process on another goroutine
	if err != nil {
		zaplog.Error("conversion error", zap.Error(err))
		return err
	}
	err = cmd.Wait() // wait until ffmpeg finish
	if err != nil {
		zaplog.Error("conversion error", zap.Error(err))
		return err
	}
	defer os.Remove(fmt.Sprintf("/tmp/%s.mp3", id))
	result, err := os.ReadFile(fmt.Sprintf("/tmp/%s.mp3", id))
	if err != nil {
		zaplog.Error("Failed to read file", zap.Error(err))
		return err
	}
	if _, err = retry.Retry(retry.NewAlgSimpleDefault(), 3, s3.UploadToS3,
		result, fmt.Sprintf("%s.mp3", id), s3.YTDLS3Bucket); err != nil {
		zaplog.Error("Failed to upload to s3", zap.Error(err))
		return err
	}
	return nil
}
