package redis

import (
	"crypto/tls"

	"github.com/go-redis/redis/v8"

	"github.com/bxxf/znvo-backend/internal/envconfig"
	"github.com/bxxf/znvo-backend/internal/logger"
)

type RedisService struct {
	redisClient *redis.Client
}

func NewRedisService(config *envconfig.EnvConfig, logger *logger.LoggerInstance) *RedisService {
	opt, err := redis.ParseURL(config.RedisURL)
	opt.TLSConfig = &tls.Config{
		InsecureSkipVerify: true,
	}
	if err != nil {
		logger.Error("Could not parse Redis URL: " + err.Error())
	}

	client := redis.NewClient(opt)

	_, err = client.Ping(client.Context()).Result()
	if err != nil {
		logger.Error("Could not connect to Redis: " + err.Error())
	}

	return &RedisService{
		redisClient: client,
	}
}

func (rs *RedisService) GetClient() *redis.Client {
	return rs.redisClient
}
