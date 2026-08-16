package filedb

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func TestDelete(t *testing.T) {
	db := NewDynamoDB(
		&MockDynamoDB{
			QueryFn: func(_ context.Context, in *dynamodb.QueryInput) (*dynamodb.QueryOutput, error) {
				return &dynamodb.QueryOutput{
					Items: []map[string]types.AttributeValue{
						{
							"Key":     &types.AttributeValueMemberS{Value: "key"},
							"Attr":    &types.AttributeValueMemberS{Value: attr_ORIGIN},
							"UserId":  &types.AttributeValueMemberS{Value: "user-id"},
							"Name":    &types.AttributeValueMemberS{Value: "orig"},
							"CType":   &types.AttributeValueMemberS{Value: "image/jpeg"},
							"TS":      &types.AttributeValueMemberN{Value: "1"},
							"Size":    &types.AttributeValueMemberN{Value: "100"},
							"Status":  &types.AttributeValueMemberS{Value: Status_AVAILABLE},
							"UserIdx": &types.AttributeValueMemberS{Value: "user-id"},
						},
						{
							"Key":    &types.AttributeValueMemberS{Value: "key"},
							"Attr":   &types.AttributeValueMemberS{Value: "rep:1x2"},
							"UserId": &types.AttributeValueMemberS{Value: "user-id"},
							"Name":   &types.AttributeValueMemberS{Value: "1x2"},
							"CType":  &types.AttributeValueMemberS{Value: "image/jpeg"},
							"TS":     &types.AttributeValueMemberN{Value: "1"},
							"Size":   &types.AttributeValueMemberN{Value: "100"},
							"Status": &types.AttributeValueMemberS{Value: Status_AVAILABLE},
						},
					},
				}, nil
			},
			TransactWriteItemsFn: func(_ context.Context, in *dynamodb.TransactWriteItemsInput) (*dynamodb.TransactWriteItemsOutput, error) {
				if len(in.TransactItems) != 2 {
					return nil, errors.New("number of items mismatch")
				}

				for _, item := range in.TransactItems {
					update := item.Update
					if update == nil {
						return nil, errors.New("not update request")
					}

					attrAV, ok := update.Key["Attr"]
					if !ok {
						return nil, errors.New("Attr not found")
					}

					attrS, ok := attrAV.(*types.AttributeValueMemberS)
					if !ok {
						return nil, errors.New("Attr is not string value")
					}
					attr := attrS.Value

					if attr == attr_ORIGIN {
						if update.ConditionExpression == nil {
							return nil, errors.New("condition is empty in origin deletion")
						}
					} else {
						if update.ConditionExpression != nil {
							return nil, errors.New("condition is not empty in replica deletion")
						}
					}
				}

				return &dynamodb.TransactWriteItemsOutput{}, nil
			},
		},
		"table",
		"userIndex",
	)

	ctx := context.Background()
	ts, err := db.Delete(ctx, "key", "user-id", 1)
	if err != nil {
		t.Logf("new timestamp: %v", ts)
		t.Errorf("filedb.Delete: %v", err)
	}
}
