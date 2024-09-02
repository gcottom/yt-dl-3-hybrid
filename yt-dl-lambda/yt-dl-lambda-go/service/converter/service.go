package converter

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"

	"github.com/gcottom/retry"
	"github.com/gcottom/yt-dl-3-hybrid/yt-dl-lambda/yt-dl-lambda-go/service/aws/s3"
)

func Convert(input []byte, id string) error {
	var args = []string{"-i", "pipe:0", "-acodec:a", "libmp3lame", "-b:a", "256k", "-f", "mp3", "-"}
	cmd := exec.Command(path.Join(os.Getenv("LAMBDA_TASK_ROOT"), "ffmpeg"), args...)
	resultBuffer := bytes.NewBuffer(make([]byte, 0))
	cmd.Stderr = nil
	cmd.Stdout = resultBuffer
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	if err = cmd.Start(); err != nil {
		return err
	}
	if _, err = stdin.Write(input); err != nil {
		return err
	}
	if err = stdin.Close(); err != nil {
		return err
	}
	if err = cmd.Wait(); err != nil {
		return err
	}
	if _, err = retry.Retry(retry.NewAlgSimpleDefault(), 3, s3.UploadToS3,
		resultBuffer, fmt.Sprintf("%s.mp3", id), s3.YTDLS3Bucket); err != nil {
		return err
	}
	return nil
}
