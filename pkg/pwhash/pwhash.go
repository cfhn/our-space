package pwhash

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"runtime"
	"strings"

	"golang.org/x/crypto/argon2"
)

var ErrSaltGeneration = errors.New("error generating salt")

var (
	defaultSaltLength         = 16
	defaultKeyLength   uint32 = 32
	defaultMemory      uint32 = 64 * 1024
	defaultIterations  uint32 = 1
	defaultParallelism        = uint8(runtime.NumCPU())
)

// Verify compares a plaintext password against a given hash.
// It rehashes the password if the parameters are lower than the standard.
func Verify(password, hash string) (string, bool) {
	parts := strings.Split(hash, "$")

	if len(parts) != 6 {
		return "", false
	}

	if parts[1] != "argon2id" {
		return "", false
	}

	var version int
	_, err := fmt.Sscanf(parts[2], "v=%d", &version)
	if err != nil {
		return "", false
	}

	if version != argon2.Version {
		return "", false
	}

	var (
		memory, iteration uint32
		parallelism       uint8
	)
	_, err = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iteration, &parallelism)
	if err != nil {
		return "", false
	}

	salt, err := base64.RawStdEncoding.Strict().DecodeString(parts[4])
	if err != nil {
		return "", false
	}

	key, err := base64.RawStdEncoding.Strict().DecodeString(parts[5])
	if err != nil {
		return "", false
	}

	keyLength := uint32(len(key))

	otherKey := argon2.IDKey([]byte(password), salt, iteration, memory, parallelism, keyLength)
	otherKeyLength := int32(len(key))

	if subtle.ConstantTimeEq(int32(keyLength), otherKeyLength) == 0 {
		return "", false
	}

	if subtle.ConstantTimeCompare(key, otherKey) != 1 {
		return "", false
	}

	if memory < defaultMemory ||
		iteration < defaultIterations ||
		parallelism < defaultParallelism ||
		len(salt) < defaultSaltLength ||
		keyLength < defaultKeyLength {
		newHash, err := Create(password)
		if err != nil {
			return "", true
		}

		return newHash, true
	}

	return "", true
}

// Create hashes a password. Use Verify for comparing.
func Create(password string) (string, error) {
	salt := make([]byte, defaultSaltLength)
	n, err := rand.Reader.Read(salt)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrSaltGeneration, err)
	}
	if n != defaultSaltLength {
		return "", fmt.Errorf("%w: short read (n=%d)", ErrSaltGeneration, n)
	}

	key := argon2.IDKey([]byte(password), salt, defaultIterations, defaultMemory, defaultParallelism, defaultKeyLength)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Key := base64.RawStdEncoding.EncodeToString(key)

	hash := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, defaultMemory, defaultIterations, defaultParallelism, b64Salt, b64Key)
	return hash, nil
}
