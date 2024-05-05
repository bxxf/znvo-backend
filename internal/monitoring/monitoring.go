package monitoring

import (
	"fmt"

	"github.com/getsentry/sentry-go"

	"github.com/bxxf/znvo-backend/internal/envconfig"
	"github.com/bxxf/znvo-backend/internal/logger"
)

type MonitoringService struct {
	config *envconfig.EnvConfig
	logger *logger.LoggerInstance
}

func NewMonitoringService(config *envconfig.EnvConfig, logger *logger.LoggerInstance) *MonitoringService {
	err := sentry.Init(sentry.ClientOptions{
		Dsn:              config.SentryDSN,
		TracesSampleRate: 1.0,
	})
	if err != nil {
		logger.Error(fmt.Printf("sentry.Init: %s", err))
	}
	return &MonitoringService{
		config: config,
		logger: logger,
	}
}
