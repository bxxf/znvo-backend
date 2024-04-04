package key

import (
	"crypto"
	"crypto/rsa"
	"log"

	"github.com/golang-jwt/jwt"

	"github.com/bxxf/znvo-backend/internal/config"
)

type Repository struct {
	key Key
}

type Key struct {
	Private *rsa.PrivateKey
	Public  crypto.PublicKey
}

func NewKeyRepository(config *config.Config) *Repository {
	rsaKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(config.JWTSecret))
	if err != nil {
		log.Fatal(err)
	}

	public := rsaKey.Public()

	return &Repository{
		key: Key{
			Private: rsaKey,
			Public:  public,
		},
	}
}

func (r *Repository) GetPublicKey() *rsa.PublicKey {
	return r.key.Public.(*rsa.PublicKey)
}

func (r *Repository) SignedString(token *jwt.Token) (string, error) {
	return token.SignedString(r.key.Private)
}
