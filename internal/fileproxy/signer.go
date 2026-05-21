package fileproxy

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"
)

type Signer struct {
	baseURL string
	secret  []byte
	ttl     time.Duration
}

type Config struct {
	BaseURL string
	Secret  string
	TTL     time.Duration
}

type tokenPayload struct {
	Key string `json:"key"`
	Exp int64  `json:"exp"`
}

func NewSigner(cfg Config) (*Signer, error) {
	baseURL := strings.TrimSpace(cfg.BaseURL)
	secret := strings.TrimSpace(cfg.Secret)

	if baseURL == "" {
		return nil, fmt.Errorf("file proxy base url is required")
	}

	if secret == "" {
		return nil, fmt.Errorf("file proxy secret is required")
	}

	if len(secret) < 32 {
		return nil, fmt.Errorf("file proxy secret must be at least 32 characters")
	}

	if cfg.TTL <= 0 {
		return nil, fmt.Errorf("file proxy ttl must be greater than zero")
	}

	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse file proxy base url: %w", err)
	}

	if parsedURL.Scheme != "https" && parsedURL.Scheme != "http" {
		return nil, fmt.Errorf("file proxy base url must include http or https scheme")
	}

	if parsedURL.Host == "" {
		return nil, fmt.Errorf("file proxy base url must include host")
	}

	baseURL = strings.TrimRight(baseURL, "/")

	return &Signer{
		baseURL: baseURL,
		secret:  []byte(secret),
		ttl:     cfg.TTL,
	}, nil
}

func (s *Signer) SignedFileURL(storageKey string) (string, error) {
	storageKey = strings.TrimSpace(storageKey)
	if storageKey == "" {
		return "", fmt.Errorf("storage key is required")
	}

	if strings.HasPrefix(storageKey, "/") || strings.Contains(storageKey, "..") {
		return "", fmt.Errorf("invalid storage key")
	}

	payload := tokenPayload{
		Key: storageKey,
		Exp: time.Now().UTC().Add(s.ttl).Unix(),
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal file token payload: %w", err)
	}

	token := base64.RawURLEncoding.EncodeToString(payloadJSON)
	signature := s.sign(token)

	fileURL, err := url.Parse(s.baseURL + "/file")
	if err != nil {
		return "", fmt.Errorf("parse file proxy url: %w", err)
	}

	query := fileURL.Query()
	query.Set("token", token)
	query.Set("signature", signature)
	fileURL.RawQuery = query.Encode()

	return fileURL.String(), nil
}

func (s *Signer) sign(value string) string {
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(value))

	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
