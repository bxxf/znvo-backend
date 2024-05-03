package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/tmc/langchaingo/llms"
)

type CustomMessageContent struct {
	Role  string       `json:"Role"`
	Parts []CustomPart `json:"Parts"`
}

type CustomPart struct {
	Text string `json:"text"`
	Type string `json:"type"`
}

type SessionData struct {
	EncryptedMessages string
	EncryptedKey      string // Store encrypted DEK as a base64 string
}

var SessionMap = struct {
	sync.RWMutex
	sessions map[string]*SessionData
}{sessions: make(map[string]*SessionData)}

// SaveMessageHistory encrypts and saves message history in Redis.
func (cs *ChatService) SaveMessageHistory(messages *[]llms.MessageContent, id string) (string, error) {
	jsonData, err := json.Marshal(messages)
	if err != nil {
		return "", err
	}

	encryptedMessages, encryptedKey, err := cs.Encrypt(jsonData)
	if err != nil {
		return "", err
	}

	// Compose the session data as a single JSON object to store in Redis
	sessionData := &SessionData{
		EncryptedMessages: encryptedMessages,
		EncryptedKey:      string(encryptedKey),
	}
	sessionDataJSON, err := json.Marshal(sessionData)
	if err != nil {
		return "", err
	}

	// Use Redis SET command to store session data
	ctx := context.Background()
	// adter hour it will expire
	if err := cs.redisClient.Set(ctx, "chist:"+id, sessionDataJSON, time.Duration(time.Minute*60)).Err(); err != nil {
		return "", fmt.Errorf("failed to save in Redis: %v", err)
	}

	fmt.Printf("Saved message history for session %s\n", id)
	return id, nil
}

// LoadMessageHistory decrypts and loads message history from Redis.
func (cs *ChatService) LoadMessageHistory(sessionID string) (*[]llms.MessageContent, error) {
	ctx := context.Background()
	result, err := cs.redisClient.Get(ctx, "chist:"+sessionID).Bytes()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve session from Redis: %v", err)
	}

	var sessionData SessionData
	if err := json.Unmarshal(result, &sessionData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session data: %v", err)
	}

	decryptedData, err := cs.Decrypt(sessionData.EncryptedMessages, sessionData.EncryptedKey)
	if err != nil {
		fmt.Printf("Failed to decrypt message history: %v\n", err)
		return nil, err
	}

	var messages []CustomMessageContent
	if err := json.Unmarshal(decryptedData, &messages); err != nil {
		fmt.Printf("Failed to unmarshal message history: %v\n", err)
		return nil, err
	}

	realMessages := convertToLLMS(messages)
	return &realMessages, nil
}

func (cs *ChatService) DeleteChatHistory(sessionID string) error {
	ctx := context.Background()
	if err := cs.redisClient.Del(ctx, "chist:"+sessionID).Err(); err != nil {
		return fmt.Errorf("failed to delete session from Redis: %v", err)
	}

	fmt.Printf("Deleted message history for session %s\n", sessionID)
	return nil
}
