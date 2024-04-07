package envconfig

// Config - configuration for the application, it defines which environment variables must be defined and fetches them into a struct
import (
	"github.com/bxxf/znvo-backend/internal/logger"
)

// ENV_VALUES - list of environment variables that must be defined
var ENV_VALUES = []string{"PORT", "FRONTEND_PORT", "JWT_SECRET", "REDIS_URL"}

type EnvConfig struct {
	FrontendPort string
	Port         string
	Env          string
	JWTSecret    string
	RedisURL     string
}

func NewEnvConfig(logger *logger.LoggerInstance) *EnvConfig {
	values := load(logger)

	// Fallback ENV to development
	if values["ENV"] == "" {
		values["ENV"] = "development"
	}

	return &EnvConfig{
		Port:         values["PORT"],
		FrontendPort: values["FRONTEND_PORT"],
		Env:          values["ENV"],
		JWTSecret:    values["JWT_SECRET"],
		RedisURL:     values["REDIS_URL"],
	}
}
