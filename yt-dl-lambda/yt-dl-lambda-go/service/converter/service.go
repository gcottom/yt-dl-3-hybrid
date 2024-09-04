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

func Convert(input []byte, id string) error {
	var args = []string{"-i", "pipe:0", "-acodec:a", "libmp3lame", "-b:a", "256k", "-f", "mp3", "-"}
	cmd := exec.Command(path.Join(os.Getenv("LAMBDA_TASK_ROOT"), "ffmpeg"), args...)
	resultBuffer := bytes.NewBuffer(make([]byte, 0))
	errBuffer := bytes.NewBuffer(make([]byte, 0))
	cmd.Stderr = errBuffer
	cmd.Stdout = resultBuffer
	stdin, err := cmd.StdinPipe()
	if err != nil {
		zaplog.Error("Failed to create stdin pipe", zap.Error(err))
		return err
	}
	if err = cmd.Start(); err != nil {
		zaplog.Error("Failed to start ffmpeg", zap.Error(err), zap.String("stderr", errBuffer.String()))
		return err
	}
	if _, err = stdin.Write(input); err != nil {
		zaplog.Error("Failed to write to stdin", zap.Error(err))
		return err
	}
	if err = stdin.Close(); err != nil {
		zaplog.Error("Failed to close stdin", zap.Error(err))
		return err
	}
	if err = cmd.Wait(); err != nil {
		zaplog.Error("Failed to wait for ffmpeg", zap.Error(err), zap.String("stderr", errBuffer.String()))
		return err
	}
	if _, err = retry.Retry(retry.NewAlgSimpleDefault(), 3, s3.UploadToS3,
		resultBuffer, fmt.Sprintf("%s.mp3", id), s3.YTDLS3Bucket); err != nil {
		zaplog.Error("Failed to upload to s3", zap.Error(err))
		return err
	}
	return nil
}
