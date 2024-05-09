package database

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/tursodatabase/go-libsql"

	datav1 "github.com/bxxf/znvo-backend/gen/api/data/v1"
	"github.com/bxxf/znvo-backend/internal/envconfig"
)

type Database struct {
	config *envconfig.EnvConfig
	db     *sql.DB
}

func NewDatabase(config *envconfig.EnvConfig) (*Database, error) {
	if config == nil {
		return nil, errors.New("database configuration is nil")
	}

	dbPath := getDbPath()
	syncInterval := 5 * time.Minute
	connector, err := libsql.NewEmbeddedReplicaConnector(dbPath, config.TursoURL,
		libsql.WithAuthToken(config.TursoToken),
		libsql.WithSyncInterval(syncInterval),
	)
	if err != nil {
		return nil, fmt.Errorf("error creating connector: %v", err)
	}

	db := sql.OpenDB(connector)
	return &Database{config: config, db: db}, nil
}

func (d *Database) Close() error {
	return d.db.Close()
}

func (d *Database) InsertUser(ctx context.Context, userId, publicKey string) error {
	_, err := d.db.ExecContext(ctx, "INSERT INTO users (id, publickey) VALUES (?, ?)", userId, publicKey)
	return err
}

func (d *Database) UploadSharedData(ctx context.Context, sender, receiver, data string) (string, string, error) {
	encryptedData, key, err := d.encryptDataForUser(ctx, receiver, data)
	if err != nil {
		return "", "", err
	}
	createdAt := time.Now().Unix()
	_, err = d.db.ExecContext(ctx, "INSERT INTO shared_data (user_id, receiver_id, data, key, created_at) VALUES (?, ?, ?, ?, ?)", sender, receiver, encryptedData, key, createdAt)
	return encryptedData, key, err
}

func (d *Database) GetSharedData(ctx context.Context, userId string) ([]*datav1.SharedDataItem, error) {
	rows, err := d.db.QueryContext(ctx, "SELECT user_id, receiver_id, data, key, created_at FROM shared_data WHERE receiver_id = ?", userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*datav1.SharedDataItem
	for rows.Next() {
		type sharedDataItem struct {
			SenderId   string
			Data       string
			CreatedAt  int64
			ReceiverId string
			Key        string
		}
		var item sharedDataItem
		if err := rows.Scan(&item.SenderId, &item.ReceiverId, &item.Data, &item.Key, &item.CreatedAt); err != nil {
			return nil, err
		}
		results = append(results, &datav1.SharedDataItem{
			SenderId:  item.SenderId,
			Data:      item.Data,
			CreatedAt: item.CreatedAt,
			Key:       item.Key,
		})
	}
	return results, rows.Err()
}

func (d *Database) encryptDataForUser(ctx context.Context, userId, data string) (string, string, error) {
	publicKey, err := d.GetPublicKey(ctx, userId)
	if err != nil {
		return "", "", err
	}
	encryptedData, key, err := hybridEncrypt(publicKey, data)

	return encryptedData, key, err
}

func (d *Database) GetPublicKey(ctx context.Context, userId string) (string, error) {
	var publicKey string
	err := d.db.QueryRowContext(ctx, "SELECT publickey FROM users WHERE id = ?", userId).Scan(&publicKey)
	return publicKey, err
}

func encryptData(publicKeySpki string, data string) (string, error) {
	publicKeyBytes, err := base64.StdEncoding.DecodeString(publicKeySpki)
	if err != nil {
		return "", err
	}
	pub, err := x509.ParsePKIXPublicKey(publicKeyBytes)
	if err != nil {
		return "", err
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return "", errors.New("public key type assertion to RSA public key failed")
	}
	ciphertext, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, rsaPub, []byte(data), nil)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}
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
	aesKey := make([]byte, 32) // 256-bit AES key
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

func getDbPath() string {
	dir, err := os.MkdirTemp("", "libsql-*")
	if err != nil {
		fmt.Println("Error creating temporary directory:", err)
		return ""
	}
	return filepath.Join(dir, "local.db")
}
