package dynamodb

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

var region *string

func init() {
	region = aws.String(os.Getenv("AWS_REGION"))
}

const (
	TableNameTrack string = "YTDL3_Tracks"
)

type DBNotFoundError struct {
	ID        string
	TableName string
}

type DBTrack struct {
	ID          string `dynamodbav:"id"`
	Status      string `dynamodbav:"status"`
	URL         string `dynamodbav:"url"`
	Title       string `dynamodbav:"title"`
	Artist      string `dynamodbav:"artist"`
	Album       string `dynamodbav:"album"`
	CoverArtURL string `dynamodbav:"cover_art_url"`
}

type DynamoClient struct {
	Client *dynamodb.DynamoDB
}

func (e *DBNotFoundError) Error() string {
	return fmt.Sprintf("id {%s} not found in table {%s}", e.ID, e.TableName)
}

const (
	StatusQueued      = "queued"
	StatusDownloading = "downloading"
	StatusProcessing  = "processing"
	StatusComplete    = "complete"
	StatusFailed      = "failed"
)
