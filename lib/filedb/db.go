package filedb

import (
	"context"
	"errors"
)

const (
	Status_RESERVED  = "reserved"
	Status_AVAILABLE = "available"
	Status_DELETING  = "deleting"
	Status_DELETED   = "deleted"
)

type Entity struct {
	Name      string
	Timestamp int64
	Size      int
	Width     int
	Height    int
	Status    string
}

type File struct {
	Key         string
	UserId      string
	ContentType string
	Entities    []*Entity
}

var (
	ErrNotFound                 = errors.New("not found")
	ErrEmptyEntities            = errors.New("empty entities")
	ErrConflictKey              = errors.New("conflict key")
	ErrInvalidContinuationToken = errors.New("invalid continuation token")
	ErrLock                     = errors.New("lock error")
)

type listOptions struct {
	limit             int
	continuationToken string
}

type ListOption func(*listOptions)

func SetLimit(limit int) ListOption {
	return func(l *listOptions) {
		l.limit = limit
	}
}

func SetContinuationToken(token string) ListOption {
	return func(l *listOptions) {
		l.continuationToken = token
	}
}

type DB interface {
	Reserve(ctx context.Context, key, userId string) (int64, error)
	CreateCommit(ctx context.Context, key, contentType string, timestamp int64, size, width, height int) error
	CreateReplica(ctx context.Context, key, name, userId, contentType string, size, width, height int) error
	List(ctx context.Context, userId string, options ...ListOption) ([]*File, string, error)
	Get(ctx context.Context, key string) (*File, error)
	Delete(ctx context.Context, key, userId string, ts int64) (int64, error)
	DeleteCommit(ctx context.Context, key string, ts int64) error
}
