package main

import (
	"context"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/gcottom/go-zaplog"
	"github.com/gcottom/yt-dl-3-hybrid/yt-dl-lambda/yt-dl-lambda-go/handlers"
)

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	ctx = zaplog.CreateAndInject(ctx)
	switch req.HTTPMethod {
	case http.MethodGet:
		return handlers.Initiate(ctx, req)
	default:
		return handlers.UnhandledMethod()
	}
}
