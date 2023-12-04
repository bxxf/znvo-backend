package utils

import (
	"os"

	"github.com/bxxf/znvo-backend/internal/config"
)

func GetOriginAndRpId(config *config.Config) (rpId string, origin string) {
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
		currentOrigin = "http://localhost:" + config.Port
	} else {
		currentOrigin = "https://" + os.Getenv("FLY_APP_NAME") + ".fly.dev"
	}

	// Setup RPID based on if app is hosted on fly
	var rpid string

	if !isOnFly {
		rpid = "localhost"
	} else {
		rpid = os.Getenv("FLY_APP_NAME") + ".fly.dev"
	}

	return rpid, currentOrigin

}

// Idea: Get rid of this and fetch origin from config
