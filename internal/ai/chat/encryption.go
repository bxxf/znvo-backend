package chat

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"

	kmspb "google.golang.org/genproto/googleapis/cloud/kms/v1"
)

// GenerateKey creates a new random key for encrypting messages. This key is used once per session or message to ensure security.
func (cs *ChatService) GenerateKey() ([]byte, error) {
	key := make([]byte, 32) // Generate a 256-bit key for strong encryption.
	if _, err := rand.Read(key); err != nil {
		return nil, err // Error handling in case of failure to generate a key.
	}
	return key, nil
}

// EncryptDataKey encrypts the session key (DEK) using a master key stored in Google KMS. This way, the DEK is never exposed.
func (cs *ChatService) EncryptDataKey(dek []byte) (string, error) {
	ctx := context.Background()
	req := &kmspb.EncryptRequest{
		Name:      cs.kekName,
		Plaintext: dek,
	}
	resp, err := cs.kmsClient.Encrypt(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt data key: %v", err) // Ensures the DEK is safely encrypted.
	}
	return base64.StdEncoding.EncodeToString(resp.Ciphertext), nil
}

// DecryptDataKey decrypts the encrypted session key (DEK) using the master key from Google KMS. Only your service can decrypt it, securing the data end-to-end.
func (cs *ChatService) DecryptDataKey(encryptedDEK string) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(encryptedDEK)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	req := &kmspb.DecryptRequest{
		Name:       cs.kekName,
		Ciphertext: data,
	}
	resp, err := cs.kmsClient.Decrypt(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data key: %v", err)
	}
	return resp.Plaintext, nil
}

// Encrypt secures the actual chat messages using the session key (DEK), ensuring that each message's contents are protected.
func (cs *ChatService) Encrypt(data []byte) (string, []byte, error) {
	dek, err := cs.GenerateKey()
	if err != nil {
		return "", nil, err
	}

	block, err := aes.NewCipher(dek)
	if err != nil {
		return "", nil, err
	}
	// GCM (Galois/Counter Mode) is a mode of operation for symmetric key cryptographic block ciphers.
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", nil, err
	}
	// Nonce is a random number used only once in cryptographic communication.
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", nil, err
	}
	// Seal encrypts and authenticates plaintext, authenticates the additional data, and appends the result to dst, returning the updated slice.
	cipherText := gcm.Seal(nonce, nonce, data, nil)
	// Encrypt the DEK using the master key from Google KMS.
	encryptedDEK, err := cs.EncryptDataKey(dek)
	if err != nil {
		return "", nil, err
	}
	return base64.StdEncoding.EncodeToString(cipherText), []byte(encryptedDEK), nil
}

// Decrypt takes encrypted chat messages and the encrypted session key (DEK), decrypts the DEK, and uses it to decrypt the message, ensuring privacy.
func (cs *ChatService) Decrypt(cipherText string, encryptedDEK string) ([]byte, error) {
	dek, err := cs.DecryptDataKey(encryptedDEK)
	if err != nil {
		return nil, err
	}

	data, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(dek)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce, cipherTextBytes := data[:nonceSize], data[nonceSize:]
	return gcm.Open(nil, nonce, cipherTextBytes, nil)
}
