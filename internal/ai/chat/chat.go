package chat

import (
	"context"
	"encoding/json"
	"fmt"

	kms "cloud.google.com/go/kms/apiv1"
	"github.com/go-redis/redis/v8"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"

	"github.com/bxxf/znvo-backend/internal/envconfig"
	rds "github.com/bxxf/znvo-backend/internal/redis"
)

type ChatService struct {
	kmsClient   *kms.KeyManagementClient // Google KMS client to manage encryption keys securely.
	kekName     string                   // Identifier for the key in KMS used to encrypt the DEK.
	redisClient *redis.Client
}

// NewChatService sets up a new service to encrypt chat messages using keys managed in Google Cloud KMS.
func NewChatService(redisClient *rds.RedisService, config *envconfig.EnvConfig) *ChatService {
	ctx := context.Background()

	var sa ServiceAccount
	if err := json.Unmarshal([]byte(config.GCPCredentials), &sa); err != nil {
		fmt.Printf("Error parsing GCP credentials: %v\n", err)
		return nil
	}

	creds, err := google.CredentialsFromJSON(ctx, []byte(config.GCPCredentials), kms.DefaultAuthScopes()...)
	if err != nil {
		fmt.Printf("Failed to create credentials from JSON: %v", err)
		return nil
	}

	client, err := kms.NewKeyManagementClient(ctx, option.WithCredentials(creds))
	if err != nil {
		fmt.Printf("Failed to create KMS client: %v", err)
		return nil
	}
	return &ChatService{
		kmsClient:   client,
		redisClient: redisClient.GetClient(),
		kekName:     "projects/finalproject-kms/locations/global/keyRings/chat/cryptoKeys/kek",
	}
}

type ServiceAccount struct {
	Type                    string `json:"type"`
	ProjectID               string `json:"project_id"`
	PrivateKeyID            string `json:"private_key_id"`
	PrivateKey              string `json:"private_key"`
	ClientEmail             string `json:"client_email"`
	ClientID                string `json:"client_id"`
	AuthURI                 string `json:"auth_uri"`
	TokenURI                string `json:"token_uri"`
	AuthProviderX509CertURL string `json:"auth_provider_x509_cert_url"`
	ClientX509CertURL       string `json:"client_x509_cert_url"`
}
