package adminpanel

import (
	"path/filepath"
	"strings"
)

// ResolveRemoteUploadPathForStaging builds the beacon path for a staged upload. If remote looks like a
// directory (trailing / or \\), the original staged filename (basename) is appended — same idea as
// copying a file into a folder. Otherwise remote is returned unchanged.
func ResolveRemoteUploadPathForStaging(remote, stagedOriginalFilename string) string {
	remote = strings.TrimSpace(remote)
	if remote == "" {
		return ""
	}
	if !strings.HasSuffix(remote, "/") && !strings.HasSuffix(remote, `\`) {
		return remote
	}
	fn := filepath.Base(strings.ReplaceAll(strings.TrimSpace(stagedOriginalFilename), `\`, `/`))
	if fn == "" || fn == "." {
		fn = "upload.bin"
	}
	base := strings.TrimRight(remote, `/\`)
	if base == "" {
		base = "/"
	}
	// Prefer backslash join when path used Windows-style separators.
	if strings.Contains(remote, `\`) {
		return base + `\` + fn
	}
	return filepath.Join(base, fn)
}
