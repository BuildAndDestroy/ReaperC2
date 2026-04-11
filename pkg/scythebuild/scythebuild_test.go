package scythebuild

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveScytheSourceDir_SCYTHE_SRC(t *testing.T) {
	dir := t.TempDir()
	scythe := filepath.Join(dir, "third_party", "Scythe")
	if err := os.MkdirAll(scythe, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scythe, "go.mod"), []byte("module x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SCYTHE_SRC", scythe)
	t.Setenv("REAPERC2_ROOT", "")

	got, err := ResolveScytheSourceDir()
	if err != nil {
		t.Fatal(err)
	}
	if got != scythe {
		t.Fatalf("got %q want %q", got, scythe)
	}
}

func TestMergeScytheHTTPHeaders(t *testing.T) {
	const cid = "68902a9a-e40f-4f15-a101-995800fa39b9"
	const sec = "secrethex"
	baseWant := "Content-Type:application/json,X-Client-Id:" + cid + ",X-API-Secret:" + sec
	if g := MergeScytheHTTPHeaders(cid, sec, ""); g != baseWant {
		t.Fatalf("empty user: got %q want %q", g, baseWant)
	}
	extra := `User-Agent:Mozilla/5.0`
	if g := MergeScytheHTTPHeaders(cid, sec, extra); g != baseWant+","+extra {
		t.Fatalf("with extra: got %q", g)
	}
	full := baseWant + ",User-Agent:x"
	if g := MergeScytheHTTPHeaders(cid, sec, full); g != full {
		t.Fatalf("legacy full headers: got %q want %q", g, full)
	}
}

func TestMergeScytheHTTPDirectories(t *testing.T) {
	const cid = "68902a9a-e40f-4f15-a101-995800fa39b9"
	baseWant := "/heartbeat/" + cid + ",/heartbeat"
	if g := MergeScytheHTTPDirectories(cid, ""); g != baseWant {
		t.Fatalf("empty user: got %q want %q", g, baseWant)
	}
	if g := MergeScytheHTTPDirectories(cid, "/shit"); g != baseWant+",/shit" {
		t.Fatalf("with extra: got %q", g)
	}
	if g := MergeScytheHTTPDirectories(cid, baseWant); g != baseWant {
		t.Fatalf("legacy full dirs: got %q", g)
	}
}

func TestNormalizeEmbedTarget(t *testing.T) {
	goos, goarch, err := NormalizeEmbedTarget("linux", "arm64")
	if err != nil {
		t.Fatal(err)
	}
	if goos != "linux" || goarch != "arm64" {
		t.Fatalf("got %s/%s", goos, goarch)
	}
	_, _, err = NormalizeEmbedTarget("plan9", "amd64")
	if err == nil {
		t.Fatal("expected error for invalid GOOS")
	}
	_, _, err = NormalizeEmbedTarget("linux", "ppc64")
	if err == nil {
		t.Fatal("expected error for invalid GOARCH")
	}
}

func TestSuggestedAttachmentFilename(t *testing.T) {
	if g, w := SuggestedAttachmentFilename("abc", "windows", "amd64"), "Scythe-embedded-abc-windows-amd64.exe"; g != w {
		t.Fatalf("got %q want %q", g, w)
	}
	if g, w := SuggestedAttachmentFilename("abc", "linux", "arm64"), "Scythe-embedded-abc-linux-arm64.bin"; g != w {
		t.Fatalf("got %q want %q", g, w)
	}
}

func TestResolveScytheSourceDir_REAPERC2_ROOT(t *testing.T) {
	dir := t.TempDir()
	scythe := filepath.Join(dir, "third_party", "Scythe")
	if err := os.MkdirAll(scythe, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scythe, "go.mod"), []byte("module x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SCYTHE_SRC", "")
	t.Setenv("REAPERC2_ROOT", dir)

	got, err := ResolveScytheSourceDir()
	if err != nil {
		t.Fatal(err)
	}
	if got != scythe {
		t.Fatalf("got %q want %q", got, scythe)
	}
}
