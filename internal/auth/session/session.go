package session

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/nrednav/cuid2"

	"github.com/bxxf/znvo-backend/internal/logger"
	rds "github.com/bxxf/znvo-backend/internal/redis"
)

var prefix = "session:"

type SessionRepository struct {
	redisClient *redis.Client
	logger      *logger.LoggerInstance
}

func NewSessionRepository(redisRepo *rds.RedisService, logger *logger.LoggerInstance) *SessionRepository {
	return &SessionRepository{
		redisClient: redisRepo.GetClient(),
		logger:      logger,
	}
}

func (r *SessionRepository) NewSession(data *webauthn.SessionData) (string, error) {

	id := prefix + cuid2.Generate()

	// transform session data to JSON
	dataJSON, err := json.Marshal(data)

	expiryTime := 5 * time.Minute

	// store session data in redis
	err = r.redisClient.Set(context.Background(), id, string(dataJSON), expiryTime).Err()

	if err != nil {
		r.logger.Error("Failed to store session data in redis", "error", err)
		return "", err
	}

	return id, nil
}

func (r *SessionRepository) GetSession(sessionID string) (*webauthn.SessionData, error) {

	data, err := r.redisClient.Get(context.Background(), sessionID).Result()

	if err != nil {
		r.logger.Error("Failed to retrieve session data from redis", "error", err)
		return nil, err
	}

	var sessionData webauthn.SessionData

	err = json.Unmarshal([]byte(data), &sessionData)

	if err != nil {
		r.logger.Error("Failed to unmarshal session data", "error", err)
		return nil, err
	}

	go func() {
		err := r.DeleteSession(sessionID)
		if err != nil {
			r.logger.Error("Failed to delete session data from redis", "error", err)
		}
	}()

	return &sessionData, nil
}

func (r *SessionRepository) DeleteSession(sessionID string) error {

	err := r.redisClient.Del(context.Background(), sessionID).Err()

	if err != nil {
		r.logger.Error("Failed to delete session data from redis", "error", err)
		return err
	}

	return nil
}
