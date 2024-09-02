package s3

import (
	"os"

	"github.com/aws/aws-sdk-go/aws"
)

var region *string
var YTDLS3Bucket string

func init() {
	region = aws.String(os.Getenv("AWS_REGION"))
	YTDLS3Bucket = os.Getenv("AWS_DOWNLOADS_BUCKET")
}
