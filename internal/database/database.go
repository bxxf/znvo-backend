package database

import (
	"context"
	"database/sql"
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

func (d *Database) GetPublicKey(ctx context.Context, userId string) (string, error) {
	var publicKey string
	err := d.db.QueryRowContext(ctx, "SELECT publickey FROM users WHERE id = ?", userId).Scan(&publicKey)
	return publicKey, err
}

func (d *Database) UserExists(userId string) (bool, error) {
	var exists bool
	err := d.db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = ?)", userId).Scan(&exists)
	return exists, err
}

func getDbPath() string {
	dir, err := os.MkdirTemp("", "libsql-*")
	if err != nil {
		fmt.Println("Error creating temporary directory:", err)
		return ""
	}
	return filepath.Join(dir, "local.db")
}
