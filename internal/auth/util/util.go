package util

import (
	"strings"

	"github.com/go-webauthn/webauthn/webauthn"

	authv1 "github.com/bxxf/znvo-backend/gen/api/auth/v1"
	"github.com/bxxf/znvo-backend/internal/auth/model"
)

// Base64 utils for webauthn - webauthn usually sends base64 encoded binary data

func Base64ToUrlSafe(base64 string) string {
	base64 = strings.ReplaceAll(base64, "+", "-")
	base64 = strings.ReplaceAll(base64, "/", "_")
	return strings.TrimRight(base64, "=")
}

var (
	UserCredentials = make(map[string]*webauthn.Credential)
	DemoUser        = model.WebAuthnUser{Id: []byte("user_id_12345"), DisplayName: "testUser"}
)

func TransformRegisterMsgToBody(req *authv1.FinishRegisterRequest) map[string]interface{} {
	// fake response body - we do not have http.Request in grpc connect request - it is a parameter in webauthn library
	resBody := make(map[string]interface{}, 4)

	resBody["type"] = "public-key"
	resBody["id"] = req.GetCredid()
	resBody["rawId"] = req.GetCredid()

	resBody["response"] = map[string]interface{}{
		"clientDataJSON":    req.GetClientdata(),
		"attestationObject": req.GetAttestation(),
	}

	return resBody

}
