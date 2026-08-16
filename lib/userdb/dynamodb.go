package userdb

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type DynamoDBRecord struct {
	Id   string `dynamodbav:"Id"`
	Attr string `dynamodbav:"Attr"`
	Name string `dynamodbav:"Name,omitempty"`
}

type DynamoDBLogger interface {
	Error(msg string)
	NameConflict(*DynamoDBRecord)
}

type defaultLogger struct{}

func (d *defaultLogger) Error(msg string)               {}
func (d *defaultLogger) NameConflict(_ *DynamoDBRecord) {}

type DynamoDBClient interface {
	Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	Scan(ctx context.Context, params *dynamodb.ScanInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error)
}

type DynamoDB struct {
	client    DynamoDBClient
	table     string
	nameIndex string

	logger DynamoDBLogger
}

var _ DB = (*DynamoDB)(nil)

type DynamoDBOption func(*DynamoDB)

func SetDynamoDBLogger(l DynamoDBLogger) DynamoDBOption {
	return func(db *DynamoDB) {
		if l != nil {
			db.logger = l
		} else {
			db.logger = &defaultLogger{}
		}
	}
}

func NewDynamoDB(
	client DynamoDBClient,
	table string,
	nameIndex string,
	options ...DynamoDBOption,
) *DynamoDB {
	ret := &DynamoDB{
		client:    client,
		table:     table,
		nameIndex: nameIndex,
	}
	for _, f := range options {
		f(ret)
	}

	return ret
}

func (db *DynamoDB) checkNameConflict(ctx context.Context, name string) error {
	expr, err := expression.NewBuilder().
		WithKeyCondition(expression.KeyEqual(expression.Key("Name"), expression.Value(name))).
		Build()
	if err != nil {
		return err
	}

	out, err := db.client.Query(ctx, &dynamodb.QueryInput{
		TableName: aws.String(db.table),
		IndexName: aws.String(db.nameIndex),

		Limit:  aws.Int32(1),
		Select: types.SelectAllProjectedAttributes,

		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
	if err != nil {
		return err
	}

	if len(out.Items) > 0 {
		var rec DynamoDBRecord
		recs := []DynamoDBRecord{}
		if err := attributevalue.UnmarshalListOfMaps(out.Items, &recs); err != nil {
			db.logger.Error("record unmarshal error at name conflict")
			return err
		} else {
			rec = recs[0]
		}
		db.logger.NameConflict(&rec)

		return ErrNameConflict
	}

	return nil
}

func (db *DynamoDB) Create(ctx context.Context, userId string, name string) (*User, error) {
	if err := db.checkNameConflict(ctx, name); err != nil {
		return nil, err
	}

	item, err := attributevalue.MarshalMap(&DynamoDBRecord{
		Id:   userId,
		Attr: "Main",
		Name: name,
	})
	if err != nil {
		db.logger.Error("record marshal error at create")
		return nil, err
	}

	expr, err := expression.NewBuilder().
		WithCondition(expression.AttributeNotExists(expression.Name("Name"))).
		Build()
	if err != nil {
		db.logger.Error("expression build error at create")
		return nil, err
	}

	if _, err := db.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(db.table),
		Item:      item,

		ConditionExpression:       expr.Condition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	}); err != nil {
		var condErr *types.ConditionalCheckFailedException
		if errors.As(err, &condErr) {
			return nil, ErrAlreadyExist
		}
		db.logger.Error("put item error at create")
		return nil, err
	}

	return &User{
		Id:   userId,
		Name: name,
	}, nil
}

func (db *DynamoDB) Get(ctx context.Context, id string) (*User, error) {
	expr, err := expression.NewBuilder().
		WithKeyCondition(expression.KeyEqual(expression.Key("Id"), expression.Value(id))).
		Build()
	if err != nil {
		db.logger.Error("expression build error at get")
		return nil, err
	}

	var ret User
	paginator := dynamodb.NewQueryPaginator(db.client, &dynamodb.QueryInput{
		TableName: aws.String(db.table),

		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
	for paginator.HasMorePages() {
		out, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, item := range out.Items {
			var rec DynamoDBRecord
			if err := attributevalue.UnmarshalMap(item, &rec); err != nil {
				db.logger.Error("record unmarshal error at get")
				return nil, err
			}

			switch rec.Attr {
			case "Main":
				ret.Id = rec.Id
				ret.Name = rec.Name
			}
		}

	}

	if ret.Id == "" {
		return nil, ErrNotFound
	}

	return &ret, nil
}

func (db *DynamoDB) List(ctx context.Context) ([]*User, error) {
	var ret []*User
	paginator := dynamodb.NewScanPaginator(db.client, &dynamodb.ScanInput{
		TableName: aws.String(db.table),
		IndexName: aws.String(db.nameIndex),
	})
	if paginator.HasMorePages() {
		out, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		var recs []*DynamoDBRecord
		if err := attributevalue.UnmarshalListOfMaps(out.Items, &recs); err != nil {
			return nil, err
		}

		for _, rec := range recs {
			ret = append(ret, &User{
				Id:   rec.Id,
				Name: rec.Name,
			})
		}
	}

	return ret, nil
}
