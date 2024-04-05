package token

import "github.com/golang-jwt/jwt"

func (r *TokenRepository) generateJWT(userID string, expiry int64) (string, error) {
	claims := AccessTokenClaims{
		UserID: userID,
		Exp:    expiry,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return r.keyRepository.SignedString(token)
}
