package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	passwordSaltLength = 16
	passwordKeyLength  = 32
)

type PasswordParams struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
}

func DefaultPasswordParams() PasswordParams {
	return PasswordParams{
		Memory:      64 * 1024,
		Iterations:  3,
		Parallelism: 2,
	}
}

func HashPassword(password string) (string, error) {
	if len(password) < 8 {
		return "", errors.New("password must be at least 8 characters")
	}

	params := DefaultPasswordParams()

	salt := make([]byte, passwordSaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}

	hash := argon2.IDKey(
		[]byte(password),
		salt,
		params.Iterations,
		params.Memory,
		params.Parallelism,
		passwordKeyLength,
	)

	encodedSalt := base64.RawStdEncoding.EncodeToString(salt)
	encodedHash := base64.RawStdEncoding.EncodeToString(hash)

	encodedPassword := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		params.Memory,
		params.Iterations,
		params.Parallelism,
		encodedSalt,
		encodedHash,
	)

	return encodedPassword, nil
}

func VerifyPassword(password string, encodedPassword string) (bool, error) {
	params, salt, expectedHash, err := decodeHash(encodedPassword)
	if err != nil {
		return false, err
	}

	actualHash := argon2.IDKey(
		[]byte(password),
		salt,
		params.Iterations,
		params.Memory,
		params.Parallelism,
		uint32(len(expectedHash)),
	)

	if subtle.ConstantTimeCompare(actualHash, expectedHash) == 1 {
		return true, nil
	}

	return false, nil
}

func decodeHash(encodedPassword string) (PasswordParams, []byte, []byte, error) {
	parts := strings.Split(encodedPassword, "$")
	if len(parts) != 6 {
		return PasswordParams{}, nil, nil, errors.New("invalid password hash format")
	}

	if parts[1] != "argon2id" {
		return PasswordParams{}, nil, nil, errors.New("unsupported password hash algorithm")
	}

	versionPart := parts[2]
	if !strings.HasPrefix(versionPart, "v=") {
		return PasswordParams{}, nil, nil, errors.New("invalid password hash version")
	}

	version, err := strconv.Atoi(strings.TrimPrefix(versionPart, "v="))
	if err != nil {
		return PasswordParams{}, nil, nil, fmt.Errorf("parse password hash version: %w", err)
	}

	if version != argon2.Version {
		return PasswordParams{}, nil, nil, errors.New("unsupported argon2 version")
	}

	paramParts := strings.Split(parts[3], ",")
	if len(paramParts) != 3 {
		return PasswordParams{}, nil, nil, errors.New("invalid password hash params")
	}

	memory, err := parseUint32Param(paramParts[0], "m")
	if err != nil {
		return PasswordParams{}, nil, nil, err
	}

	iterations, err := parseUint32Param(paramParts[1], "t")
	if err != nil {
		return PasswordParams{}, nil, nil, err
	}

	parallelismUint32, err := parseUint32Param(paramParts[2], "p")
	if err != nil {
		return PasswordParams{}, nil, nil, err
	}

	if parallelismUint32 > 255 {
		return PasswordParams{}, nil, nil, errors.New("invalid password hash parallelism")
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return PasswordParams{}, nil, nil, fmt.Errorf("decode password salt: %w", err)
	}

	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return PasswordParams{}, nil, nil, fmt.Errorf("decode password hash: %w", err)
	}

	params := PasswordParams{
		Memory:      memory,
		Iterations:  iterations,
		Parallelism: uint8(parallelismUint32),
	}

	return params, salt, hash, nil
}

func parseUint32Param(part string, name string) (uint32, error) {
	prefix := name + "="

	if !strings.HasPrefix(part, prefix) {
		return 0, fmt.Errorf("missing password hash param %s", name)
	}

	value, err := strconv.ParseUint(strings.TrimPrefix(part, prefix), 10, 32)
	if err != nil {
		return 0, fmt.Errorf("parse password hash param %s: %w", name, err)
	}

	return uint32(value), nil
}
