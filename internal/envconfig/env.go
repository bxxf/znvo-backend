package envconfig

// Config - configuration for the application, it defines which environment variables must be defined and fetches them into a struct
import (
	"github.com/bxxf/znvo-backend/internal/logger"
)

// ENV_VALUES - list of environment variables that must be defined
var ENV_VALUES = []string{"PORT", "JWT_SECRET", "REDIS_URL", "GCP_CREDENTIALS", "SENTRY_DSN"}

type EnvConfig struct {
	Port           string
	Env            string
	JWTSecret      string
	RedisURL       string
	GCPCredentials string
	SentryDSN      string
}

func NewEnvConfig(logger *logger.LoggerInstance) *EnvConfig {
	values := load(logger)

	// Fallback ENV to development
	if values["ENV"] == "" {
		values["ENV"] = "development"
	}

	return &EnvConfig{
		Port:           values["PORT"],
		Env:            values["ENV"],
		JWTSecret:      values["JWT_SECRET"],
		RedisURL:       values["REDIS_URL"],
		GCPCredentials: values["GCP_CREDENTIALS"],
		SentryDSN:      values["SENTRY_DSN"],
	}
}
