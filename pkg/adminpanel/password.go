package adminpanel

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
)

const (
	argon2idStoredPrefix = "argon2id$"
	argon2Version        = 19
	saltLength           = 16
	argon2KeyLength      = 32
)

// Argon2id parameters (memory in KiB per golang.org/x/crypto/argon2).
type argon2Params struct {
	time    uint32
	memory  uint32 // KiB
	threads uint8
}

func argon2ParamsFromEnv() argon2Params {
	p := argon2Params{
		time:    3,
		memory:  65536, // 64 MiB
		threads: 4,
	}
	if v := os.Getenv("ADMIN_ARGON2_TIME"); v != "" {
		if n, err := strconv.ParseUint(v, 10, 32); err == nil && n >= 1 {
			p.time = uint32(n)
		}
	}
	if v := os.Getenv("ADMIN_ARGON2_MEMORY_KIB"); v != "" {
		if n, err := strconv.ParseUint(v, 10, 32); err == nil && n >= 8 {
			p.memory = uint32(n)
		}
	}
	if v := os.Getenv("ADMIN_ARGON2_THREADS"); v != "" {
		if n, err := strconv.ParseUint(v, 10, 8); err == nil && n >= 1 {
			p.threads = uint8(n)
		}
	}
	return p
}

// HashOperatorPassword returns a serialized Argon2id hash for storage in operators.password_hash.
func HashOperatorPassword(plain string) (string, error) {
	if plain == "" {
		return "", fmt.Errorf("empty password")
	}
	salt := make([]byte, saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	p := argon2ParamsFromEnv()
	key := argon2.IDKey([]byte(plain), salt, p.time, p.memory, p.threads, argon2KeyLength)
	return formatArgon2idStored(p, salt, key), nil
}

func formatArgon2idStored(p argon2Params, salt, key []byte) string {
	enc := base64.RawStdEncoding
	return fmt.Sprintf(
		"%sv=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2idStoredPrefix,
		argon2Version,
		p.memory,
		p.time,
		p.threads,
		enc.EncodeToString(salt),
		enc.EncodeToString(key),
	)
}

// VerifyOperatorPassword checks plain against a stored bcrypt or Argon2id hash.
func VerifyOperatorPassword(storedHash, plain string) bool {
	if storedHash == "" || plain == "" {
		return false
	}
	if strings.HasPrefix(storedHash, "$2a$") || strings.HasPrefix(storedHash, "$2b$") {
		err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(plain))
		return err == nil
	}
	if !strings.HasPrefix(storedHash, argon2idStoredPrefix) {
		log.Printf("admin: unknown password hash format")
		return false
	}
	ok, err := verifyArgon2idStored(storedHash, plain)
	if err != nil {
		log.Printf("admin: argon2 verify: %v", err)
		return false
	}
	return ok
}

func verifyArgon2idStored(storedHash, plain string) (bool, error) {
	// argon2id$v=19$m=65536,t=3,p=4$<salt_b64>$<key_b64>
	rest := strings.TrimPrefix(storedHash, argon2idStoredPrefix)
	parts := strings.Split(rest, "$")
	if len(parts) != 4 {
		return false, fmt.Errorf("invalid argon2id encoding (want 4 segments after prefix)")
	}
	var v int
	if _, err := fmt.Sscanf(parts[0], "v=%d", &v); err != nil || v != argon2Version {
		return false, fmt.Errorf("unsupported argon2 version")
	}
	var mem uint32
	var itime uint32
	var threads uint
	if _, err := fmt.Sscanf(parts[1], "m=%d,t=%d,p=%d", &mem, &itime, &threads); err != nil {
		return false, fmt.Errorf("parse params: %w", err)
	}
	enc := base64.RawStdEncoding
	salt, err := enc.DecodeString(parts[2])
	if err != nil {
		return false, fmt.Errorf("salt: %w", err)
	}
	want, err := enc.DecodeString(parts[3])
	if err != nil {
		return false, fmt.Errorf("hash: %w", err)
	}
	got := argon2.IDKey([]byte(plain), salt, itime, mem, uint8(threads), uint32(len(want)))
	if len(got) != len(want) {
		return false, nil
	}
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}
