package token

import "github.com/golang-jwt/jwt"

func (r *TokenRepository) generateJWT(userID string, expiry int64) (string, error) {
	claims := AccessTokenClaims{
		UserID: userID,
		Exp:    expiry,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(r.config.JWTSecret))
}
