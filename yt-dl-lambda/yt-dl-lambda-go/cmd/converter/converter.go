package main

import (
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/gcottom/yt-dl-3-hybrid/yt-dl-lambda/yt-dl-lambda-go/handlers"
)

func main() {
	lambda.Start(handler)
}

func handler(sqsEvent events.SQSEvent) error {
	return handlers.Convert(sqsEvent)
}
