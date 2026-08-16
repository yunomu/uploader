package userdb

import (
	"context"
	"errors"
)

type User struct {
	Id   string
	Name string
}

var (
	ErrNotFound     = errors.New("not found")
	ErrNameConflict = errors.New("name conflict")
	ErrAlreadyExist = errors.New("already exist")
)

type DB interface {
	Create(ctx context.Context, userId, name string) (*User, error)
	Get(ctx context.Context, id string) (*User, error)
	List(ctx context.Context) ([]*User, error)
}
