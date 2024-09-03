package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
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
	_, params, err := mime.ParseMediaType(req.Headers["Content-Type"])
	if err != nil {
		return &events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       fmt.Sprintf("Invalid Content-Type header: %v", err),
		}, nil
	}

	reader := multipart.NewReader(bytes.NewReader([]byte(req.Body)), params["boundary"])
	form, err := reader.ReadForm(50 << 20) // Increase buffer size to 50 MB
	if err != nil {
		return &events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       fmt.Sprintf("Failed to parse form: %v", err),
		}, nil
	}
	defer form.RemoveAll()
	fileHeaders := form.File["file"]
	if len(fileHeaders) == 0 {
		return &events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       "Missing file",
		}, nil
	}
	id := form.Value["id"]
	if len(id) == 0 {
		return &events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       "Missing id parameter",
		}, nil
	}
	file, err := fileHeaders[0].Open()
	if err != nil {
		return &events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf("Failed to open file: %v", err),
		}, nil
	}
	defer file.Close()
	var buf bytes.Buffer
	_, err = io.Copy(&buf, file)
	if err != nil {
		return &events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf("Failed to read file: %v", err),
		}, nil
	}
	if err := s3.UploadToS3(&buf, id[0], s3.YTDLS3Bucket); err != nil {
		return &events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf("Failed to upload file: %v", err),
		}, nil
	}
	dynamoClient := dynamodb.CreateDynamoClient(context.Background())
	if err := dynamoClient.PutTrack(context.Background(), &dynamodb.DBTrack{ID: id[0], Status: dynamodb.StatusProcessing}); err != nil {
		return &events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf("Failed to update db: %v", err),
		}, nil
	}
	if err := sqs.SQSSendMessage(sqs.SQSConverterURL, id[0]); err != nil {
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
		res, err := retry.Retry(retry.NewAlgSimpleDefault(), 3, s3.DownloadFromS3Buf, id, s3.YTDLS3Bucket)
		if err != nil {
			return err
		}
		data := res[0].(*aws.WriteAtBuffer)
		dynamoClient := dynamodb.CreateDynamoClient(context.Background())
		if err := converter.Convert(data.Bytes(), id); err != nil {
			_, re := retry.Retry(retry.NewAlgSimpleDefault(), 3, dynamoClient.PutTrack, context.Background(), &dynamodb.DBTrack{ID: id, Status: dynamodb.StatusFailed})
			if re != nil {
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
			_, re := retry.Retry(retry.NewAlgSimpleDefault(), 3, dynamoClient.PutTrack, context.Background(), &dynamodb.DBTrack{ID: recordData.ID, Status: dynamodb.StatusFailed})
			if re != nil {
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
	tracks, err := dynamoClient.GetTrackByID(context.Background(), id)
	if err != nil {
		return &events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       fmt.Sprintf("Failed to get tracks: %v", err),
		}, nil
	}
	response, err := json.Marshal(tracks)
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

func UnhandledMethod() (*events.APIGatewayProxyResponse, error) {
	return nil, nil
}
