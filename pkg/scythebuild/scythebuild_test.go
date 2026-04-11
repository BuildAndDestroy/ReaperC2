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
