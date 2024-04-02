package config

// Config - configuration for the application, it defines which environment variables must be defined and fetches them into a struct
import (
	"github.com/bxxf/znvo-backend/internal/logger"
)

// ENV_VALUES - list of environment variables that must be defined
var ENV_VALUES = []string{"PORT", "FRONTEND_PORT"}

type Config struct {
	FrontendPort string
	Port         string
	Env          string
}

func NewConfig(logger *logger.LoggerInstance) *Config {
	values := load(logger)

	// Fallback ENV to development
	if values["ENV"] == "" {
		values["ENV"] = "development"
	}

	return &Config{
		Port:         values["PORT"],
		FrontendPort: values["FRONTEND_PORT"],
		Env:          values["ENV"],
	}
}
