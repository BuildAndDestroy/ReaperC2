package adminpanel

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"ReaperC2/pkg/dbconnections"
)

const (
	cookieName     = "reaperc2_admin_session"
	cookieMaxAge   = 86400 * 7 // 7 days (browser hint; real expiry is server-side)
	sessionIDBytes = 32
)

func sessionTTL() time.Duration {
	h := getEnvDefault("ADMIN_SESSION_TTL_HOURS", "168")
	n, err := strconv.Atoi(h)
	if err != nil || n < 1 {
		n = 168
	}
	return time.Duration(n) * time.Hour
}

func beaconPublicBaseURL() string {
	return strings.TrimRight(getEnvDefault("BEACON_PUBLIC_BASE_URL", "http://127.0.0.1:8080"), "/")
}

func adminCookieSecure() bool {
	return strings.EqualFold(os.Getenv("ADMIN_COOKIE_SECURE"), "true")
}

func getEnvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// BootstrapFirstOperator creates an operator when the DB has none and env is set.
func BootstrapFirstOperator() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	n, err := dbconnections.CountOperators(ctx)
	if err != nil || n > 0 {
		return
	}
	user := os.Getenv("ADMIN_BOOTSTRAP_USERNAME")
	pass := os.Getenv("ADMIN_BOOTSTRAP_PASSWORD")
	if user == "" || pass == "" {
		log.Println("admin: no operators in database; set ADMIN_BOOTSTRAP_USERNAME and ADMIN_BOOTSTRAP_PASSWORD to create the first account, or insert into MongoDB operators collection.")
		return
	}
	hash, err := HashOperatorPassword(pass)
	if err != nil {
		log.Printf("admin: bootstrap password hash: %v", err)
		return
	}
	err = dbconnections.InsertOperator(ctx, dbconnections.Operator{
		Username:     user,
		PasswordHash: hash,
		Role:         dbconnections.RoleAdmin,
	})
	if err != nil {
		log.Printf("admin: bootstrap insert operator: %v", err)
		return
	}
	log.Printf("admin: created bootstrap operator %q", user)
}

func newSessionToken() (string, error) {
	b := make([]byte, sessionIDBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
