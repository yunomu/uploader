package filedb

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type MockDynamoDB struct {
	QueryFn              func(context.Context, *dynamodb.QueryInput) (*dynamodb.QueryOutput, error)
	TransactWriteItemsFn func(context.Context, *dynamodb.TransactWriteItemsInput) (*dynamodb.TransactWriteItemsOutput, error)
}

func (m *MockDynamoDB) DeleteItem(_ context.Context, _ *dynamodb.DeleteItemInput) (*dynamodb.DeleteItemOutput, error) {
	panic("not implemented") // TODO: Implement
}

func (m *MockDynamoDB) PutItem(_ context.Context, _ *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	panic("not implemented") // TODO: Implement
}

func (m *MockDynamoDB) Query(ctx context.Context, in *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	if f := m.QueryFn; f != nil {
		return f(ctx, in)
	}
	panic("not assigned")
}
func (m *MockDynamoDB) UpdateItem(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
	panic("not implemented") // TODO: Implement
}

func (m *MockDynamoDB) TransactWriteItems(ctx context.Context, in *dynamodb.TransactWriteItemsInput, optFns ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error) {
	if f := m.TransactWriteItemsFn; f != nil {
		return f(ctx, in)
	}
	panic("not assigned")
}
