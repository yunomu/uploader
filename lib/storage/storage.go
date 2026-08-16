package storage

import (
	"context"
	"io"
)

type Storage interface {
	Put(ctx context.Context, key string, contentType string, blob io.ReadSeeker) error
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
}
