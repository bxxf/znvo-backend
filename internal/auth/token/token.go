package token

import (
	"fmt"
	"log"
	"time"

	"github.com/golang-jwt/jwt"

	"github.com/bxxf/znvo-backend/internal/envconfig"
	"github.com/bxxf/znvo-backend/internal/logger"
)

type AccessTokenClaims struct {
	UserID    string `json:"userId"`
	Exp       int64  `json:"exp"`
	PublicKey string `json:"publicKey"`

	jwt.StandardClaims
}

type AccessToken struct {
	Token  string `json:"token"`
	UserID string `json:"userId"`
}

type TokenRepository struct {
	config *envconfig.EnvConfig
	logger *logger.LoggerInstance
}

func NewTokenRepository(config *envconfig.EnvConfig, logger *logger.LoggerInstance) *TokenRepository {
	return &TokenRepository{
		config: config,
		logger: logger,
	}
}

func (r *TokenRepository) CreateAccessToken(publicKey string, userID string) (string, error) {
	expiry := time.Now().Add(time.Minute * 15)
	token, err := r.generateJWT(userID, publicKey, expiry.Unix())
	if err != nil {
		log.Printf("could not generate token: %v", err)
		return "", fmt.Errorf("could not generate token: %w", err)
	}
	return token, nil
}

func (r *TokenRepository) ParseAccessToken(tokenString string) (*AccessToken, error) {
	r.logger.Debug("parsing access token " + tokenString)
	claims := &AccessTokenClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return r.config.JWTSecret, nil
	})

	if err != nil {
		r.logger.Error("could not parse token: %v", err)
		return nil, fmt.Errorf("could not parse token: %w", err)
	}

	if !token.Valid {
		r.logger.Error("token is invalid")
		return nil, fmt.Errorf("token is invalid")
	}

	if claims.Exp < time.Now().Unix() {
		r.logger.Error("token has expired")
		return nil, fmt.Errorf("token has expired")
	}

	return &AccessToken{
		Token:  tokenString,
		UserID: claims.UserID,
	}, nil

}
