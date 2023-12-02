package config

import (
	"github.com/bxxf/znvo-backend/internal/logger"
	"go.uber.org/fx"
)

var ENV_VALUES = []string{"PORT"}

type Config struct {
	Port string
	Env  string
}

func NewConfig(lc fx.Lifecycle, logger *logger.LoggerInstance) *Config {
	values := load(logger)

	// Fallback ENV to development
	if values["ENV"] == "" {
		values["ENV"] = "development"
	}

	return &Config{
		Port: values["PORT"],
		Env:  values["ENV"],
	}
}
