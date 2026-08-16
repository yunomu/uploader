package handler

import (
	"context"
	"errors"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	apipb "github.com/yunomu/uploader/proto/api"

	"github.com/yunomu/uploader/lib/userdb"
)

type BadRequest struct {
	status  int
	message string
}

func (r *BadRequest) Error() string {
	return r.message
}

func (r *BadRequest) Status() int {
	if r.status == 0 {
		return 400
	}
	return r.status
}

type Logger interface {
	HandlerError(err error, req *Request)
	Error(err error, msg string)
	InitUser(*userdb.User)
}

type defaultLogger struct{}

func (l *defaultLogger) Error(err error, msg string)          {}
func (l *defaultLogger) HandlerError(err error, req *Request) {}
func (l *defaultLogger) InitUser(*userdb.User)                {}

type Request events.APIGatewayV2HTTPRequest
type Response events.APIGatewayV2HTTPResponse

type Handler struct {
	db userdb.DB

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
	db userdb.DB,
	opts ...Option,
) *Handler {
	h := &Handler{
		db: db,

		marshaler: protojson.MarshalOptions{
			EmitUnpopulated: true,
		},
		unmarshaler: protojson.UnmarshalOptions{
			DiscardUnknown: true,
		},

		logger: &defaultLogger{},
	}
	for _, f := range opts {
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

func (h *Handler) create(ctx context.Context, req *Request) (proto.Message, error) {
	userId, err := getUserId(req)
	if err != nil {
		return nil, err
	}

	var r apipb.InitUserRequest
	if err := h.unmarshaler.Unmarshal([]byte(req.Body), &r); err != nil {
		h.logger.Error(err, "unmarshal in create user")
		return nil, &BadRequest{
			message: "unknown json format",
		}
	}

	if r.Name == "" {
		return nil, &BadRequest{
			message: "name is empty",
		}
	}

	user, err := h.db.Create(ctx, userId, r.Name)
	if err != nil {
		return nil, err
	}

	h.logger.InitUser(user)

	return nil, nil
}

func (h *Handler) get(ctx context.Context, req *Request) (proto.Message, error) {
	userId, err := getUserId(req)
	if err != nil {
		return nil, err
	}

	user, err := h.db.Get(ctx, userId)
	if err == userdb.ErrNotFound {
		return nil, &BadRequest{
			status:  404,
			message: "user not initialized",
		}
	} else if err != nil {
		return nil, err
	}

	return &apipb.GetUserResponse{
		Id:   userId,
		Name: user.Name,
	}, nil
}

func (h *Handler) Serve(ctx context.Context, req *Request) (*Response, error) {
	handlers := map[string]func(context.Context, *Request) (proto.Message, error){
		"POST /v1/user": h.create,
		"GET /v1/user":  h.get,
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
				StatusCode: badReq.Status(),
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
