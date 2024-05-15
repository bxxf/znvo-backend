package database

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"errors"
)

func hybridEncrypt(publicKeySpki string, data string) (string, string, error) {
	publicKeyBytes, err := base64.StdEncoding.DecodeString(publicKeySpki)
	if err != nil {
		return "", "", err
	}
	pub, err := x509.ParsePKIXPublicKey(publicKeyBytes)
	if err != nil {
		return "", "", err
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return "", "", errors.New("public key type assertion to RSA public key failed")
	}

	// Generate AES key
	aesKey := make([]byte, 32)
	if _, err := rand.Read(aesKey); err != nil {
		return "", "", err
	}

	// Encrypt data with AES
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return "", "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", "", err
	}
	ciphertext := gcm.Seal(nil, nonce, []byte(data), nil)
	fullCiphertext := append(nonce, ciphertext...) // Append nonce to ciphertext
	encData := base64.StdEncoding.EncodeToString(fullCiphertext)

	// Encrypt AES key with RSA
	encryptedAESKey, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, rsaPub, aesKey, nil)
	if err != nil {
		return "", "", err
	}
	encKey := base64.StdEncoding.EncodeToString(encryptedAESKey)

	return encData, encKey, nil
}

func (d *Database) encryptDataForUser(ctx context.Context, userId, data string) (string, string, error) {
	publicKey, err := d.GetPublicKey(ctx, userId)
	if err != nil {
		return "", "", err
	}
	encryptedData, key, err := hybridEncrypt(publicKey, data)

	return encryptedData, key, err
}
