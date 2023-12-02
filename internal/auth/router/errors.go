package router

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var MissingEmailOrPassword = status.New(codes.InvalidArgument, "missing email or password").Err()
