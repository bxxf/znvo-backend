package router

import (
	"context"

	"connectrpc.com/connect"
	authv1 "github.com/bxxf/znvo-backend/gen/proto/go/api/auth/v1"
)

type AuthRouter struct {
}

func NewAuthRouter() *AuthRouter {
	return &AuthRouter{}
}

func (a *AuthRouter) RegisterUser(ctx context.Context, req *connect.Request[authv1.RegisterUserRequest]) (*connect.Response[authv1.RegisterUserResponse], error) {
	email := req.Msg.Email
	password := req.Msg.Password

	if email == "" || password == "" {
		return nil, MissingEmailOrPassword
	}

	return nil, nil
}
