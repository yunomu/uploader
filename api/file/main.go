package main

import (
	"context"
	"log/slog"
	"math/rand"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/yunomu/uploader/lib/filedb"
	"github.com/yunomu/uploader/lib/randstr"
	"github.com/yunomu/uploader/lib/storage"
	"github.com/yunomu/uploader/lib/userdb"

	"github.com/yunomu/uploader/api/file/internal/handler"
)

var (
	logger *slog.Logger
	debug  bool
)

func init() {
	debug = os.Getenv("DEBUG") != ""

	levelVar := new(slog.LevelVar)
	if debug {
		levelVar.Set(slog.LevelDebug)
	}

	logger = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: levelVar,
	}))
}

type userdbLogger struct {
	logger *slog.Logger
}

func (l *userdbLogger) Error(msg string) {
	l.logger.Error(msg)
}

func (l *userdbLogger) NameConflict(record *userdb.DynamoDBRecord) {
	l.logger.Error("Name conflict", "conflicted_record", record)
}

type filedbLogger struct {
	logger *slog.Logger
}

func (l *filedbLogger) Error(msg string) {
	l.logger.Error(msg)
}

type handlerLogger struct {
	logger *slog.Logger
}

func (h *handlerLogger) HandlerError(err error, req *handler.Request) {
	h.logger.Error("Handler", "err", err, "request", req)
}

func (h *handlerLogger) Error(err error, msg string) {
	h.logger.Error(msg, "err", err)
}

func main() {
	ctx := context.Background()

	bucket := os.Getenv("FILE_BUCKET")
	fileTable := os.Getenv("FILE_TABLE")
	fileUserIndex := os.Getenv("FILE_USER_INDEX")
	userTable := os.Getenv("USER_TABLE")
	userNameIndex := os.Getenv("USER_NAME_INDEX")
	region := os.Getenv("REGION")

	seed := time.Now().UnixNano()

	slog.Info("Init",
		"bucket", bucket,
		"fileTable", fileTable,
		"userTable", userTable,
		"region", region,
		"debug", debug,
		"seed", seed,
	)

	cfg, err := config.LoadDefaultConfig(ctx, func(opt *config.LoadOptions) error {
		opt.Region = region
		return nil
	})
	if err != nil {
		slog.Error("LoadDefaultConfig", "err", err)
		return
	}

	h := handler.NewHandler(
		userdb.NewDynamoDB(
			dynamodb.NewFromConfig(cfg),
			userTable,
			userNameIndex,
			userdb.SetDynamoDBLogger(
				&userdbLogger{
					logger: logger.With("module", "userdb"),
				},
			),
		),
		filedb.NewDynamoDB(
			dynamodb.NewFromConfig(cfg),
			fileTable,
			fileUserIndex,
			filedb.SetDynamoDBLogger(
				&filedbLogger{
					logger: logger.With("module", "filedb"),
				},
			),
		),
		storage.NewS3(
			s3.NewFromConfig(cfg),
			bucket,
		),
		randstr.NewMathRand(
			rand.New(rand.NewSource(seed)),
			16,
		),
		handler.SetLogger(&handlerLogger{
			logger: logger.With("module", "handler"),
		}),
	)

	lambda.StartWithContext(ctx, h.Serve)
}
