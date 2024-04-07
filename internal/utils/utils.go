package utils

import (
	"os"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bxxf/znvo-backend/internal/envconfig"
	"github.com/bxxf/znvo-backend/internal/logger"
)

func GetOriginAndRpId(config *envconfig.EnvConfig) (rpId string, origin string) {
	// Setup current origin based on if app is hosted on fly
	var isOnFly bool

	// If FLY_APP_NAME env variable is not set, then app is not hosted on fly
	if val := os.Getenv("FLY_APP_NAME"); val == "" {
		isOnFly = false
	} else {
		isOnFly = true
	}

	var currentOrigin string

	if !isOnFly {
		currentOrigin = "http://localhost:3000"
	} else {
		currentOrigin = "https://znvo.co.uk"
	}

	// Setup RPID based on if app is hosted on fly
	var rpid string

	if !isOnFly {
		rpid = "localhost"
	} else {
		rpid = "znvo.co.uk"
	}

	return rpid, currentOrigin

}

// Idea: Get rid of this and fetch origin from config

func HandleError(err error, msg string, logger logger.LoggerInstance) error {

	logger.Error(msg, "error", err)
	return status.Errorf(codes.Internal, "%s: %v", msg, err)
}
