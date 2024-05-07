package database

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/tursodatabase/go-libsql"

	"github.com/bxxf/znvo-backend/internal/envconfig"
)

type Database struct {
	config *envconfig.EnvConfig
}

// NewDatabase creates a new instance of Database with the given configuration.
func NewDatabase(config *envconfig.EnvConfig) *Database {
	if config == nil {
		fmt.Println("Database configuration is nil")
		return nil
	}
	return &Database{config: config}
}

// openDB opens a new database connection using the configuration stored in the Database struct.
func (d *Database) openDB() (*sql.DB, error) {
	if d == nil {
		return nil, fmt.Errorf("Database instance is nil")
	}

	dbPath := d.getDbPath()
	if dbPath == "" {
		return nil, fmt.Errorf("failed to get or create database path")
	}

	syncInterval := 5 * time.Minute
	connector, err := libsql.NewEmbeddedReplicaConnector(dbPath, d.config.TursoURL,
		libsql.WithAuthToken(d.config.TursoToken),
		libsql.WithSyncInterval(syncInterval),
	)
	if err != nil {
		return nil, fmt.Errorf("error creating connector: %v", err)
	}

	dbConn := sql.OpenDB(connector)
	if dbConn == nil {
		return nil, fmt.Errorf("failed to open database connection")
	}

	return dbConn, nil
}

func (d *Database) GetPublicKey(userId string) (string, error) {
	db, err := d.openDB()
	if err != nil {
		return "", err
	}
	defer db.Close()

	var publicKey string
	err = db.QueryRow("SELECT publickey FROM users WHERE id = ?", userId).Scan(&publicKey)
	return publicKey, err
}

func (d *Database) InsertUser(userId, publicKey string) error {
	db, err := d.openDB()
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec("INSERT INTO users (id, publickey) VALUES (?, ?)", userId, publicKey)
	return err
}

func (d *Database) UploadSharedData(sender, receiver, data string) error {
	db, err := d.openDB()
	if err != nil {
		return err
	}
	defer db.Close()

	encryptedData, err := d.encryptDataForUser(receiver, data)
	if err != nil {
		return err
	}

	_, err = db.Exec("INSERT INTO shared_data (user_id, receiver_id, data) VALUES (?, ?, ?)", sender, receiver, encryptedData)
	return err
}

func (d *Database) GetSharedData(userId string) (string, error) {
	db, err := d.openDB()
	if err != nil {
		return "", err
	}
	defer db.Close()

	var encryptedData string
	err = db.QueryRow("SELECT data FROM shared_data WHERE receiver_id = ?", userId).Scan(&encryptedData)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "[]", nil
		}

		return "", err
	}

	fmt.Printf("Encrypted data: %s\n", encryptedData)
	return encryptedData, err
}

func (d *Database) encryptDataForUser(userId, data string) (string, error) {
	pkey, err := d.GetPublicKey(userId)
	if err != nil {
		return "", err
	}
	return encryptData(pkey, data)
}

func encryptData(publicKeySpki string, data string) (string, error) {
	// Decode the base64-encoded public key
	publicKeyBytes, err := base64.StdEncoding.DecodeString(publicKeySpki)
	if err != nil {
		return "", err
	}

	// Parse the public key
	pub, err := x509.ParsePKIXPublicKey(publicKeyBytes)
	if err != nil {
		return "", err
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return "", errors.New("public key type assertion to RSA public key failed")
	}

	// Encrypt the data using RSA-OAEP with SHA-256
	hash := sha256.New()
	ciphertext, err := rsa.EncryptOAEP(hash, rand.Reader, rsaPub, []byte(data), nil)
	if err != nil {
		return "", err
	}

	// Encode the encrypted data to base64
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}
func base64ToPEM(base64Key string) (string, error) {
	// Decode the base64 string to bytes
	derBytes, err := base64.StdEncoding.DecodeString(base64Key)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 string: %v", err)
	}

	// Construct a pem.Block struct for the public key
	pemBlock := &pem.Block{
		Type:  "PUBLIC KEY", // Type of the public key to be included in the PEM header
		Bytes: derBytes,
	}

	// Encode the DER bytes into a PEM format
	pemBytes := pem.EncodeToMemory(pemBlock)
	if pemBytes == nil {
		return "", fmt.Errorf("failed to encode PEM data")
	}

	return string(pemBytes), nil
}

func (d *Database) getDbPath() string {
	dir, err := os.MkdirTemp("", "libsql-*")
	if err != nil {
		fmt.Println("Error creating temporary directory:", err)
		return ""
	}
	return filepath.Join(dir, "local.db")
}
