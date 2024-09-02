package dynamodb

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/gcottom/go-zaplog"
)

func CreateDynamoClient(ctx context.Context) *DynamoClient {
	zaplog.InfoC(ctx, "creating dynamo client")
	conf := aws.Config{Region: region}
	sess := session.Must(session.NewSession(&conf))
	svc := dynamodb.New(sess)
	dc := DynamoClient{svc}
	return &dc
}
