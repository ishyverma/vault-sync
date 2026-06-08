package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Credentials struct {
	NotionToken string `json:"notion_token,omitempty"`
}

func credentialsPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "credentials.json"), nil
}

func keyPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, ".vault-key"), nil
}

func loadOrGenerateKey() ([]byte, error) {
	kp, err := keyPath()
	if err != nil {
		return nil, err
	}
	if data, err := os.ReadFile(kp); err == nil && len(data) == 32 {
		return data, nil
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(kp), 0o700); err != nil {
		return nil, err
	}
	if err := os.WriteFile(kp, key, 0o600); err != nil {
		return nil, fmt.Errorf("write key: %w", err)
	}
	return key, nil
}

func encrypt(plaintext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return aesGCM.Seal(nonce, nonce, plaintext, nil), nil
}

func decrypt(ciphertext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return aesGCM.Open(nil, nonce, ciphertext, nil)
}

func LoadCredentials() (*Credentials, error) {
	cp, err := credentialsPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(cp)
	if err != nil {
		if os.IsNotExist(err) {
			return &Credentials{}, nil
		}
		return nil, err
	}

	key, err := loadOrGenerateKey()
	if err != nil {
		return nil, err
	}

	decrypted, err := decrypt(data, key)
	if err != nil {
		return nil, fmt.Errorf("decrypt credentials: %w", err)
	}

	var creds Credentials
	if err := json.Unmarshal(decrypted, &creds); err != nil {
		return nil, fmt.Errorf("parse credentials: %w", err)
	}
	return &creds, nil
}

func SaveCredentials(creds *Credentials) error {
	plaintext, err := json.Marshal(creds)
	if err != nil {
		return fmt.Errorf("marshal credentials: %w", err)
	}

	key, err := loadOrGenerateKey()
	if err != nil {
		return err
	}

	encrypted, err := encrypt(plaintext, key)
	if err != nil {
		return fmt.Errorf("encrypt credentials: %w", err)
	}

	cp, err := credentialsPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(cp), 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(cp, encrypted, 0o600); err != nil {
		return fmt.Errorf("write credentials: %w", err)
	}
	return nil
}
