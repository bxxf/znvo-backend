package util

import (
	"strings"

	"github.com/go-webauthn/webauthn/webauthn"

	authv1 "github.com/bxxf/znvo-backend/gen/api/auth/v1"
)

// Base64 utils for webauthn - webauthn usually sends base64 encoded binary data

func Base64ToUrlSafe(base64 string) string {
	base64 = strings.ReplaceAll(base64, "+", "-")
	base64 = strings.ReplaceAll(base64, "/", "_")
	return strings.TrimRight(base64, "=")
}

var (
	UserCredentials = make(map[string]*webauthn.Credential)
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

func TransformLoginMsgToBody(req *authv1.FinishLoginRequest) map[string]interface{} {
	// fake response body - we do not have http.Request in grpc connect request - it is a parameter in webauthn library
	resBody := make(map[string]interface{}, 5)
	resBody["id"] = req.GetCredid()
	resBody["type"] = "public-key"
	resBody["rawId"] = req.GetCredid()
	resBody["response"] = map[string]interface{}{
		"authenticatorData": req.GetAuthdata(),
		"signature":         req.GetSignature(),
		"clientDataJSON":    req.GetClientdata(),
	}

	return resBody
}
