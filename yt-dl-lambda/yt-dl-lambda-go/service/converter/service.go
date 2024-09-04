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

func Convert(id string) error {
	// Define input and output paths
	inputPath := fmt.Sprintf("/tmp/%s.temp", id)
	outputPath := fmt.Sprintf("/tmp/%s.mp3", id)
	ffmpegPath := path.Join(os.Getenv("LAMBDA_TASK_ROOT"), "ffmpeg")
	os.Remove(outputPath)
	// Define ffmpeg command arguments
	args := []string{"-i", inputPath, "-c:a", "libmp3lame", "-b:a", "256k", "-f", "mp3", outputPath}
	cmd := exec.Command(ffmpegPath, args...)

	// Capture stderr to get detailed ffmpeg error messages
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	// Log the start of the conversion
	zaplog.Info("converting file", zap.String("id", id))

	// Start the ffmpeg command
	if err := cmd.Start(); err != nil {
		zaplog.Error("Failed to start ffmpeg", zap.Error(err))
		return err
	}

	// Wait for the ffmpeg command to finish
	if err := cmd.Wait(); err != nil {
		zaplog.Error("FFmpeg failed", zap.Error(err), zap.String("stderr", stderr.String()))
		return err
	}

	// Read the converted file
	result, err := os.ReadFile(outputPath)
	if err != nil {
		zaplog.Error("Failed to read file", zap.Error(err))
		return err
	}

	// Clean up the output file after processing
	defer os.Remove(outputPath)

	// Upload the converted file to S3 with retry logic
	if _, err = retry.Retry(retry.NewAlgSimpleDefault(), 3, s3.UploadToS3,
		result, fmt.Sprintf("%s.mp3", id), s3.YTDLS3Bucket); err != nil {
		zaplog.Error("Failed to upload to S3", zap.Error(err))
		return err
	}

	return nil
}
