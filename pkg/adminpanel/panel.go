package adminpanel

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/url"
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

// ResolveBeaconBaseURL normalizes operator input for the beacon C2 base URL (scheme + host [:port] only).
// Empty input uses BEACON_PUBLIC_BASE_URL (see beaconPublicBaseURL). Accepts full URLs or host:port / FQDN / IP with optional port.
func ResolveBeaconBaseURL(input string) (string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return beaconPublicBaseURL(), nil
	}
	raw := input
	if !strings.Contains(raw, "://") {
		raw = "http://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid beacon base URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("beacon base URL must use http or https")
	}
	if strings.TrimSpace(u.Host) == "" {
		return "", fmt.Errorf("beacon base URL must include a host (FQDN or IP) and optional port")
	}
	origin := &url.URL{Scheme: u.Scheme, Host: u.Host}
	out := strings.TrimRight(origin.String(), "/")
	return out, nil
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
