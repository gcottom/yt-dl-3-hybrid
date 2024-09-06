package main

import (
	"context"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/gcottom/go-zaplog"
	"github.com/gcottom/yt-dl-3-hybrid/yt-dl-lambda/yt-dl-lambda-go/handlers"
)

func main() {
	lambda.Start(handler)
}

func handler(ctx context.Context, sqsEvent events.SQSEvent) error {
	ctx = zaplog.CreateAndInject(ctx)
	return handlers.Meta(ctx, sqsEvent)
}
