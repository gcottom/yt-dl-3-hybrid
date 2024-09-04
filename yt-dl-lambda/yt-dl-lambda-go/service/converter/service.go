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
	resultBuffer := bytes.NewBuffer(make([]byte, 20<<20)) // pre allocate 5MiB buffer

	cmd.Stderr = os.Stderr    // bind log stream to stderr
	cmd.Stdout = resultBuffer // stdout result will be written here

	stdin, err := cmd.StdinPipe() // Open stdin pipe
	if err != nil {
		return err
	}

	err = cmd.Start() // Start a process on another goroutine
	if err != nil {
		return err
	}

	_, err = stdin.Write(input) // pump audio data to stdin pipe
	if err != nil {
		return err
	}
	err = stdin.Close() // close the stdin, or ffmpeg will wait forever
	if err != nil {
		return err
	}
	err = cmd.Wait() // wait until ffmpeg finish
	if err != nil {
		return err
	}
	if _, err = retry.Retry(retry.NewAlgSimpleDefault(), 3, s3.UploadToS3,
		resultBuffer, fmt.Sprintf("%s.mp3", id), s3.YTDLS3Bucket); err != nil {
		zaplog.Error("Failed to upload to s3", zap.Error(err))
		return err
	}
	return nil
}
