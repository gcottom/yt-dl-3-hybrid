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
	ID          string `dynamodbav:"id" json:"id"`
	Status      string `dynamodbav:"status" json:"status,omitempty"`
	URL         string `dynamodbav:"url" json:"url,omitempty"`
	Title       string `dynamodbav:"title" json:"title"`
	Artist      string `dynamodbav:"artist" json:"artist"`
	Album       string `dynamodbav:"album" json:"album,omitempty"`
	CoverArtURL string `dynamodbav:"cover_art_url" json:"cover_art_url,omitempty"`
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
