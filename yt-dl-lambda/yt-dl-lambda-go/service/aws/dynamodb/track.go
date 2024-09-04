package dynamodb

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/gcottom/go-zaplog"
	"github.com/gcottom/retry"
	"go.uber.org/zap"
)

func (c *DynamoClient) GetTrackByID(ctx context.Context, id string) (*DBTrack, error) {
	zaplog.InfoC(ctx, "query db for track", zap.String("trackID", id))
	r, err := retry.Retry(retry.NewAlgSimpleDefault(), 3, c.Client.GetItem, &dynamodb.GetItemInput{
		TableName: aws.String(TableNameTrack),
		Key: map[string]*dynamodb.AttributeValue{
			"id": {
				S: &id,
			},
		},
	})
	if err != nil {
		zaplog.ErrorC(ctx, "fatal db error when retrieving track", zap.String("trackID", id), zap.Error(err))
		return nil, err
	}
	result := r[0].(*dynamodb.GetItemOutput)
	if result.Item == nil {
		return nil, &DBNotFoundError{id, TableNameTrack}
	}
	item := DBTrack{}
	if err = dynamodbattribute.UnmarshalMap(result.Item, &item); err != nil {
		return nil, err
	}
	return &item, nil
}

func (c *DynamoClient) PutTrack(ctx context.Context, track *DBTrack) error {
	zaplog.InfoC(ctx, "update db for track", zap.String("trackID", track.ID))
	av, err := dynamodbattribute.MarshalMap(*track)
	if err != nil {
		return err
	}
	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(TableNameTrack),
	}
	if _, err = retry.Retry(retry.NewAlgSimpleDefault(), 3, c.Client.PutItem, input); err != nil {
		zaplog.ErrorC(ctx, "fatal db error updating track", zap.String("trackID", track.ID), zap.Error(err))
		return err
	}
	return nil
}
