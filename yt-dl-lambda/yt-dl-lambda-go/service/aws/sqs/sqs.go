package sqs

import (
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

func SQSSendMessage(queueUrl string, message string) error {
	conf := aws.Config{Region: aws.String(region)}
	sess := session.Must(session.NewSession(&conf))
	sqsClient := sqs.New(sess)
	_, err := sqsClient.SendMessage(&sqs.SendMessageInput{
		QueueUrl:    &queueUrl,
		MessageBody: &message,
	})
	return err
}

func SQSDeleteMessage(queueUrl string, r events.SQSMessage) error {
	conf := aws.Config{Region: aws.String(region)}
	sess := session.Must(session.NewSession(&conf))
	sqsClient := sqs.New(sess)
	_, err := sqsClient.DeleteMessage(&sqs.DeleteMessageInput{
		QueueUrl:      &queueUrl,
		ReceiptHandle: &r.ReceiptHandle,
	})
	return err
}
