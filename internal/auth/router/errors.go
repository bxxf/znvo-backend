package router

// Errors - predefine errors for router for faster access and cleaner code
import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var MissingEmailOrPassword = status.New(codes.InvalidArgument, "missing email or password").Err()

var MissingSession = status.New(codes.InvalidArgument, "missing session").Err()

var MissingUser = status.New(codes.InvalidArgument, "missing user").Err()
