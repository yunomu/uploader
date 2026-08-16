package handler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"strings"
	"time"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/aws/aws-lambda-go/events"

	apipb "github.com/yunomu/uploader/proto/api"

	"github.com/yunomu/uploader/lib/filedb"
	"github.com/yunomu/uploader/lib/randstr"
	"github.com/yunomu/uploader/lib/storage"
	"github.com/yunomu/uploader/lib/userdb"
)

type BadRequest struct {
	message string
}

func (r *BadRequest) Error() string {
	return r.message
}

type Request events.APIGatewayV2HTTPRequest
type Response events.APIGatewayV2HTTPResponse

type Logger interface {
	HandlerError(err error, req *Request)
	Error(err error, msg string)
}

type defaultLogger struct{}

func (l *defaultLogger) Error(err error, msg string)          {}
func (l *defaultLogger) HandlerError(err error, req *Request) {}

type Handler struct {
	userdb  userdb.DB
	filedb  filedb.DB
	storage storage.Storage
	rand    randstr.Generator

	marshaler   protojson.MarshalOptions
	unmarshaler protojson.UnmarshalOptions

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
	userdb userdb.DB,
	filedb filedb.DB,
	storage storage.Storage,
	rand randstr.Generator,
	options ...Option,
) *Handler {
	h := &Handler{
		userdb:  userdb,
		filedb:  filedb,
		storage: storage,
		rand:    rand,

		marshaler: protojson.MarshalOptions{
			EmitUnpopulated: true,
		},
		unmarshaler: protojson.UnmarshalOptions{
			DiscardUnknown: true,
		},

		logger: &defaultLogger{},
	}
	for _, f := range options {
		f(h)
	}

	return h
}

func getUserId(req *Request) (string, error) {
	auth := req.RequestContext.Authorizer
	if auth == nil {
		return "", errors.New("authorizer description is not found in request context")
	}

	jwt := auth.JWT
	if jwt == nil {
		return "", errors.New("JWT is not found in authorizer description")
	}

	sub, ok := jwt.Claims["sub"]
	if !ok {
		return "", errors.New("sub is not found in JWT claims")
	}

	return sub, nil
}

func (h *Handler) reserveKey(ctx context.Context, userId string) (string, int64, error) {
	for {
		select {
		case <-ctx.Done():
			return "", 0, ctx.Err()
		default:
			// do nothing
		}

		key, err := h.rand.Generate()
		if err != nil {
			return "", 0, err
		}

		ts, err := h.filedb.Reserve(ctx, key, userId)
		if err == filedb.ErrConflictKey {
			continue
		} else if err != nil {
			return "", 0, err
		}

		return key, ts, nil
	}
}

func (h *Handler) upload(ctx context.Context, req *Request) (proto.Message, error) {
	userId, err := getUserId(req)
	if err != nil {
		return nil, err
	}

	var r apipb.UploadRequest
	if err := h.unmarshaler.Unmarshal([]byte(req.Body), &r); err != nil {
		h.logger.Error(err, "unmarshal error in upload funcion")
		return nil, &BadRequest{
			message: "invalid json format",
		}
	}

	ctx, _ = context.WithTimeout(ctx, 5*time.Second)
	key, ts, err := h.reserveKey(ctx, userId)
	if err != nil {
		return nil, err
	}

	var width, height int
	if img, format, err := image.Decode(bytes.NewReader(r.Blob)); err == image.ErrFormat {
		// do nothing
	} else if err != nil {
		return nil, err
	} else {
		switch format {
		case "png", "jpeg", "gif":
			if strings.TrimPrefix(r.ContentType, "image/") != format {
				return nil, errors.New("contentType mismatch")
			}
			rect := img.Bounds()
			width = rect.Max.X
			height = rect.Max.Y
		default:
			// do nothing
		}
	}

	s3Key := fmt.Sprintf("files/%s/orig", key)

	if err := h.storage.Put(ctx, s3Key, r.ContentType, bytes.NewReader(r.Blob)); err != nil {
		return nil, err
	}

	if err := h.filedb.CreateCommit(ctx, key, r.ContentType, ts, len(r.Blob), width, height); err != nil {
		return nil, err
	}

	return &apipb.UploadResponse{
		Key:       key,
		Timestamp: ts,
	}, nil
}

func (h *Handler) list(ctx context.Context, req *Request) (proto.Message, error) {
	userId, err := getUserId(req)
	if err != nil {
		return nil, err
	}

	var ops []filedb.ListOption
	ops = append(ops, filedb.SetLimit(10))
	if token, ok := req.QueryStringParameters["token"]; ok {
		ops = append(ops, filedb.SetContinuationToken(token))
	}

	files, cToken, err := h.filedb.List(ctx, userId, ops...)
	if err != nil {
		return nil, err
	}

	var objects []*apipb.Object
	for _, file := range files {
		objects = append(objects, &apipb.Object{
			Key:         file.Key,
			ContentType: file.ContentType,
			Timestamp:   file.Entities[0].Timestamp,
		})
	}

	return &apipb.ListResponse{
		Objects:           objects,
		ContinuationToken: cToken,
	}, nil
}

func (h *Handler) get(ctx context.Context, req *Request) (proto.Message, error) {
	userId, err := getUserId(req)
	if err != nil {
		return nil, err
	}

	key, ok := req.PathParameters["key"]
	if !ok || key == "" {
		return nil, &BadRequest{
			message: "invalid key",
		}
	}

	file, err := h.filedb.Get(ctx, key)
	if err == filedb.ErrNotFound {
		return nil, &BadRequest{
			message: "not found",
		}
	} else if err != nil {
		return nil, err
	}

	if file.UserId != userId {
		return nil, &BadRequest{
			message: "not found",
		}
	}

	ret := &apipb.GetFileResponse{
		Key:         file.Key,
		ContentType: file.ContentType,
	}
	for _, entity := range file.Entities {
		ret.Files = append(ret.Files, &apipb.File{
			Path:      fmt.Sprintf("%s/%s", file.Key, entity.Name),
			Timestamp: entity.Timestamp,
			Size:      int32(entity.Size),
			Width:     int32(entity.Width),
			Height:    int32(entity.Height),
		})
	}

	return ret, nil
}

func (h *Handler) delete(ctx context.Context, req *Request) (proto.Message, error) {
	userId, err := getUserId(req)
	if err != nil {
		return nil, err
	}

	key, ok := req.PathParameters["key"]
	if !ok || key == "" {
		return nil, &BadRequest{
			message: "invalid key",
		}
	}

	var r apipb.DeleteFileRequest
	if err := h.unmarshaler.Unmarshal([]byte(req.Body), &r); err != nil {
		h.logger.Error(err, "unmarshal error in delete")
		return nil, &BadRequest{
			message: "invalid json format",
		}
	}

	ts, err := h.filedb.Delete(ctx, key, userId, r.Timestamp)
	if err != nil {
		return nil, err
	}

	return &apipb.DeleteFileResponse{
		Timestamp: ts,
	}, nil
}

func (h *Handler) Serve(ctx context.Context, req *Request) (*Response, error) {
	handlers := map[string]func(context.Context, *Request) (proto.Message, error){
		"POST /v1/file":         h.upload,
		"GET /v1/file":          h.list,
		"GET /v1/file/{key}":    h.get,
		"DELETE /v1/file/{key}": h.delete,
	}

	handler, ok := handlers[req.RouteKey]
	if !ok {
		return &Response{
			StatusCode: 404,
			Headers: map[string]string{
				"Content-Type": "text/plain",
			},
			Body: "NotFound",
		}, nil
	}

	res, err := handler(ctx, req)
	if err != nil {
		if badReq, ok := err.(*BadRequest); ok {
			return &Response{
				StatusCode: 400,
				Body:       badReq.message,
			}, nil
		}

		h.logger.HandlerError(err, req)
		return &Response{
			StatusCode: 503,
			Body:       "Internal server error",
		}, nil
	}

	var buf strings.Builder
	if res != nil {
		bs, err := h.marshaler.Marshal(res)
		if err != nil {
			h.logger.Error(err, "marshal error in Serve")
			return &Response{
				StatusCode: 503,
				Body:       "Internal server error",
			}, nil
		}

		if _, err := buf.Write(bs); err != nil {
			h.logger.Error(err, "buffer write error in Serve")
			return &Response{
				StatusCode: 503,
				Body:       "Internal server error",
			}, nil
		}
	}

	return &Response{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: buf.String(),
	}, nil
}
