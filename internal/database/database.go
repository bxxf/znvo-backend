package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/tursodatabase/go-libsql"

	"github.com/bxxf/znvo-backend/internal/envconfig"
)

type Database struct {
	db *sql.DB
}

func NewDatabase(config *envconfig.EnvConfig) *Database {
	dbPath := getDbPath()
	// sync with remote database each 5 mins
	syncInterval := time.Minute * 5
	connector, err := libsql.NewEmbeddedReplicaConnector(dbPath, config.TursoURL,
		libsql.WithAuthToken(config.TursoToken),
		libsql.WithSyncInterval(syncInterval),
	)
	if err != nil {
		fmt.Println("Error creating connector:", err)
		os.Exit(1)
	}

	defer connector.Close()

	db := sql.OpenDB(connector)
	defer db.Close()

	return &Database{
		db: db,
	}
}

func (d *Database) GetPublicKey(userId string) (string, error) {
	var publicKey string
	err := d.db.QueryRow("SELECT public_key FROM users WHERE user_id = ?", userId).Scan(&publicKey)
	if err != nil {
		return "", err
	}
	return publicKey, nil
}

func (d *Database) InsertUser(userId, publicKey string) error {
	_, err := d.db.Exec("INSERT INTO users (user_id, public_key) VALUES (?, ?)", userId, publicKey)
	if err != nil {
		return err
	}
	return nil
}

func getDbPath() string {
	dbName := "local.db"

	// create dir for embedded replica
	dir, err := os.MkdirTemp("", "libsql-*")
	if err != nil {
		fmt.Println("Error creating temporary directory:", err)
		os.Exit(1)
	}

	dbPath := filepath.Join(dir, dbName)
	return dbPath
}
