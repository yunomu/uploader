package handler

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"

	"github.com/aws/aws-lambda-go/events"

	"github.com/yunomu/uploader/lib/cdn"
	"github.com/yunomu/uploader/lib/filedb"
	"github.com/yunomu/uploader/lib/image"
	"github.com/yunomu/uploader/lib/storage"
)

type Logger interface {
	ItemFormatError(item interface{})
	FileLockError(msg string, file *filedb.File)
}

type defaultLogger struct{}

func (l *defaultLogger) ItemFormatError(item interface{})            {}
func (l *defaultLogger) FileLockError(msg string, file *filedb.File) {}

type Handler struct {
	filedb  filedb.DB
	storage storage.Storage
	cdn     cdn.CDN

	logger Logger
}

type Option func(*Handler)

func SetLogger(l Logger) Option {
	return func(h *Handler) {
		if l == nil {
			h.logger = &defaultLogger{}
		} else {
			h.logger = l
		}
	}
}

func NewHandler(
	filedb filedb.DB,
	storage storage.Storage,
	cdn cdn.CDN,
	options ...Option,
) *Handler {
	h := &Handler{
		filedb:  filedb,
		storage: storage,
		cdn:     cdn,

		logger: &defaultLogger{},
	}
	for _, f := range options {
		f(h)
	}

	return h
}

type Request events.DynamoDBEvent

func imageToFile(img map[string]events.DynamoDBAttributeValue) (*filedb.File, error) {
	ts, err := img["TS"].Int64()
	if err != nil {
		return nil, err
	}

	size, err := img["Size"].Integer()
	if err != nil {
		return nil, err
	}

	width, err := img["W"].Integer()
	if err != nil {
		return nil, err
	}

	height, err := img["H"].Integer()
	if err != nil {
		return nil, err
	}

	return &filedb.File{
		Key:         img["Key"].String(),
		UserId:      img["UserId"].String(),
		ContentType: img["ContentType"].String(),
		Entities: []*filedb.Entity{
			{
				Name:      img["Name"].String(),
				Timestamp: ts,
				Size:      int(size),
				Width:     int(width),
				Height:    int(height),
				Status:    img["Status"].String(),
			},
		},
	}, nil
}

func (h *Handler) createReplica(ctx context.Context, file *filedb.File) error {
	switch file.ContentType {
	case "image/jpeg", "image/png", "image/gif":
		// continue
	default:
		return nil
	}

	r, err := h.storage.Get(ctx, fmt.Sprintf("file/%s/orig", file.Key))
	if err != nil {
		return err
	}

	resized, rect, err := image.Resize(r)
	if err == image.NoNeed {
		bs, err := ioutil.ReadAll(r)
		if err != nil {
			return err
		}
		resized = bytes.NewReader(bs)
	} else if err != nil {
		return err
	}
	r.Close()

	name := "mid"
	if err := h.storage.Put(ctx, fmt.Sprintf("file/%s/%s", file.Key, name), file.ContentType, resized); err != nil {
		return err
	}

	if err := h.filedb.CreateReplica(ctx, file.Key, name, file.UserId, file.ContentType, resized.Len(), rect.Max.X, rect.Max.Y); err != nil {
		return err
	}

	return nil
}

func (h *Handler) deleteCommit(ctx context.Context, file *filedb.File) error {
	ts := file.Entities[0].Timestamp
	file, err := h.filedb.Get(ctx, file.Key)
	if err != nil {
		return err
	}

	var paths []string
	for _, entity := range file.Entities {
		path := fmt.Sprintf("file/%s/%s", file.Key, entity.Name)
		if err := h.storage.Delete(ctx, path); err != nil {
			return err
		}
		paths = append(paths, path)
	}

	if err := h.cdn.Invalidate(ctx, paths); err != nil {
		return err
	}

	if err := h.filedb.DeleteCommit(ctx, file.Key, ts); err != nil {
		return err
	}

	return nil
}

func (h *Handler) Serve(ctx context.Context, req *Request) error {
	for _, record := range req.Records {
		file, err := imageToFile(record.Change.NewImage)
		if err != nil {
			h.logger.ItemFormatError(record.Change.NewImage)
			return err
		}
		entity := file.Entities[0]

		switch entity.Status {
		case filedb.Status_AVAILABLE:
			if err := h.createReplica(ctx, file); err != nil {
				if err == filedb.ErrLock {
					h.logger.FileLockError("create replica", file)
					continue
				}
				return err
			}
		case filedb.Status_DELETING:
			if err := h.deleteCommit(ctx, file); err != nil {
				if err == filedb.ErrLock {
					h.logger.FileLockError("delete commit", file)
					continue
				}
				return err
			}
		default:
			continue
		}
	}

	return nil
}
