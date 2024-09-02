package sqs

import (
	"fmt"
	"os"
)

const (
	SQSConverter string = "yt-dl-3-convert"
	SQSGenre     string = "yt-dl-3-genre"
	SQSMeta      string = "yt-dl-3-meta"
)

var (
	SQSFmtBaseURL   string = "https://sqs.%s.amazonaws.com/%s/%s"
	SQSConverterURL string
	SQSGenreURL     string
	SQSMetaURL      string
	region          string
)

func init() {
	region := os.Getenv("AWS_REGION")
	account := os.Getenv("AWS_ACCOUNT_ID")
	SQSConverterURL = fmt.Sprintf(SQSFmtBaseURL, region, account, SQSConverter)
	SQSGenreURL = fmt.Sprintf(SQSFmtBaseURL, region, account, SQSGenre)
	SQSMetaURL = fmt.Sprintf(SQSFmtBaseURL, region, account, SQSMeta)
}

type MetaQueueSQSMessage struct {
	ID    string `json:"id"`
	Genre string `json:"genre"`
}
