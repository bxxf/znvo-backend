package chat

import (
	"context"
	"fmt"

	kms "cloud.google.com/go/kms/apiv1"
	"github.com/go-redis/redis/v8"

	rds "github.com/bxxf/znvo-backend/internal/redis"
)

type ChatService struct {
	kmsClient   *kms.KeyManagementClient // Google KMS client to manage encryption keys securely.
	kekName     string                   // Identifier for the key in KMS used to encrypt the DEK.
	redisClient *redis.Client
}

// NewChatService sets up a new service to encrypt chat messages using keys managed in Google Cloud KMS.
func NewChatService(redisClient *rds.RedisService) *ChatService {
	ctx := context.Background()
	client, err := kms.NewKeyManagementClient(ctx)
	if err != nil {
		fmt.Printf("failed to create kms client: %v", err)
	}
	return &ChatService{
		kmsClient:   client,
		redisClient: redisClient.GetClient(),
		kekName:     "projects/finalproject-kms/locations/global/keyRings/chat/cryptoKeys/kek",
	}
}
