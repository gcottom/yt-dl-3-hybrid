package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/gcottom/retry"
	"github.com/gcottom/yt-dl-3-hybrid/yt-dl-lambda/yt-dl-lambda-go/pkg/http_client"
	"github.com/gcottom/yt-dl-3-hybrid/yt-dl-lambda/yt-dl-lambda-go/service/aws/dynamodb"
	"github.com/gcottom/yt-dl-3-hybrid/yt-dl-lambda/yt-dl-lambda-go/service/aws/s3"
	"github.com/gcottom/yt-dl-3-hybrid/yt-dl-lambda/yt-dl-lambda-go/service/aws/sqs"
	"github.com/gcottom/yt-dl-3-hybrid/yt-dl-lambda/yt-dl-lambda-go/service/converter"
	"github.com/gcottom/yt-dl-3-hybrid/yt-dl-lambda/yt-dl-lambda-go/service/meta"
	"golang.org/x/oauth2/clientcredentials"
)

type InitiatorResponse struct {
	State string `json:"state"`
}

func Initiate(req events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	id := req.QueryStringParameters["id"]
	if err := sqs.SQSSendMessage(sqs.SQSConverterURL, id); err != nil {
		return &events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf("Failed to send message: %v", err),
		}, nil
	}
	response := InitiatorResponse{State: "ACK"}
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		return &events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf("Failed to marshal response: %v", err),
		}, nil
	}
	return &events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       string(jsonResponse),
	}, nil
}

func Convert(sqsEvent events.SQSEvent) error {
	for _, record := range sqsEvent.Records {
		id := record.Body
		res, err := retry.Retry(retry.NewAlgSimpleDefault(), 3, s3.DownloadFromS3File, id, ".temp", s3.YTDLS3Bucket)
		if err != nil {
			return err
		}
		data := res[0].(*os.File)
		defer os.Remove(data.Name())
		defer data.Close()

		dynamoClient := dynamodb.CreateDynamoClient(context.Background())
		if err := converter.Convert(data, id); err != nil {
			if re := dynamoClient.PutTrack(context.Background(), &dynamodb.DBTrack{ID: id, Status: dynamodb.StatusFailed}); re != nil {
				return re
			}
			return err
		}
		if _, err := retry.Retry(retry.NewAlgSimpleDefault(), 3, s3.DeleteFromS3, id, s3.YTDLS3Bucket); err != nil {
			return err
		}
		if _, err := retry.Retry(retry.NewAlgSimpleDefault(), 3, sqs.SQSDeleteMessage, sqs.SQSConverterURL, record); err != nil {
			return err
		}
		if _, err := retry.Retry(retry.NewAlgSimpleDefault(), 3, sqs.SQSSendMessage, sqs.SQSGenreURL, id); err != nil {
			return err
		}
	}
	return nil
}

func Meta(sqsEvent events.SQSEvent) error {
	for _, record := range sqsEvent.Records {
		var recordData sqs.MetaQueueSQSMessage
		if err := json.Unmarshal([]byte(record.Body), &recordData); err != nil {
			return err
		}
		httpClient := http_client.NewHTTPClient()
		dynamoClient := dynamodb.CreateDynamoClient(context.Background())
		metaService := &meta.Service{HTTPClient: httpClient, DBClient: dynamoClient,
			SpotifyConfig: &clientcredentials.Config{ClientID: os.Getenv("SPOTIFY_CLIENT_ID"), ClientSecret: os.Getenv("SPOTIFY_CLIENT_SECRET")}}
		res, err := retry.Retry(retry.NewAlgSimpleDefault(), 3, s3.DownloadFromS3Buf, fmt.Sprintf("%s.mp3", recordData.ID), s3.YTDLS3Bucket)
		if err != nil {
			return err
		}
		data := res[0].(*aws.WriteAtBuffer)
		if err := metaService.SaveMeta(context.Background(), data.Bytes(), recordData.ID, recordData.Genre); err != nil {
			if re := dynamoClient.PutTrack(context.Background(), &dynamodb.DBTrack{ID: recordData.ID, Status: dynamodb.StatusFailed}); re != nil {
				return re
			}
			return err
		}
	}
	return nil
}

func Status(req events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	dynamoClient := dynamodb.CreateDynamoClient(context.Background())
	id, ok := req.QueryStringParameters["id"]
	if !ok {
		return &events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       "Missing id parameter",
		}, nil
	}
	track, err := dynamoClient.GetTrackByID(context.Background(), id)
	if err != nil {
		var notFoundError *dynamodb.DBNotFoundError
		if errors.As(err, &notFoundError) {
			track := dynamodb.DBTrack{ID: id, Status: dynamodb.StatusProcessing}
			response, er := json.Marshal(track)
			if er == nil {
				return &events.APIGatewayProxyResponse{
					StatusCode: 200,
					Body:       string(response),
				}, nil
			}
		}
		return &events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf("Failed to get tracks: %v", err),
		}, nil
	}
	if track.URL != "" {
		url, err := s3.GeneratePresignedDownloadURL(track.URL, s3.YTDLS3Bucket)
		if err != nil {
			return &events.APIGatewayProxyResponse{
				StatusCode: 500,
				Body:       fmt.Sprintf("Failed to get presigned url: %v", err),
			}, nil
		}
		track.URL = url.URL
	}
	response, err := json.Marshal(track)
	if err != nil {
		return &events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf("Failed to marshal response: %v", err),
		}, nil
	}
	return &events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       string(response),
	}, nil
}

func GetPresignedUploadURL(req events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	id, ok := req.QueryStringParameters["id"]
	if !ok {
		return &events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       "Missing id parameter",
		}, nil
	}
	url, err := s3.GeneratePresignedUploadURL(id, s3.YTDLS3Bucket)
	if err != nil {
		return &events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf("Failed to get presigned url: %v", err),
		}, nil
	}
	jsonResponse, err := json.Marshal(url)
	if err != nil {
		return &events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf("Failed to marshal presigned url to json: %v", err),
		}, nil
	}
	return &events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       string(jsonResponse),
	}, nil
}

func UnhandledMethod() (*events.APIGatewayProxyResponse, error) {
	return nil, nil
}
