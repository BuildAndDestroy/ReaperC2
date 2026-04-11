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
	"strings"
)

// HTTPOptions mirrors Scythe Http CLI flags (see ./Scythe Http -h).
// Timeout is the HTTP client timeout (e.g. "30s"), independent of ReaperC2 heartbeat interval.
type HTTPOptions struct {
	Method        string
	Timeout       string // e.g. "5s", "2m"; default "30s"
	Body          string
	Directories   string // comma-separated; default /heartbeat/<clientID>,/heartbeat
	Headers       string // comma-separated key:value; default uses client id + secret
	Proxy         string
	SkipTLSVerify bool
}

// DefaultHTTPOptions returns Method GET and Timeout "30s".
func DefaultHTTPOptions() HTTPOptions {
	return HTTPOptions{Method: "GET", Timeout: "30s"}
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
	dirs := strings.TrimSpace(o.Directories)
	if dirs == "" {
		dirs = fmt.Sprintf("/heartbeat/%s,/heartbeat", clientID)
	}
	headers := strings.TrimSpace(o.Headers)
	if headers == "" {
		headers = fmt.Sprintf("Content-Type:application/json,X-Client-Id:%s,X-API-Secret:%s", clientID, secret)
	}

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

// BuildEmbeddedBinary runs go build in the Scythe tree with EmbeddedArgv set to tokens.
func BuildEmbeddedBinary(ctx context.Context, tokens []string) ([]byte, error) {
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
	outPath := filepath.Join(tmpDir, "Scythe.embedded")

	ldflags := "-X github.com/BuildAndDestroy/Scythe/pkg/userinput.EmbeddedArgv=" + b64
	cmd := exec.CommandContext(ctx, "go", "build", "-trimpath", "-ldflags", ldflags, "-o", outPath, "./cmd")
	cmd.Dir = root
	cmd.Env = os.Environ()
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("go build in %s: %w\n%s", root, err, strings.TrimSpace(stderr.String()))
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
