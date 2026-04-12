package adminpanel

import "testing"

func TestResolveRemoteUploadPathForStaging_UnchangedWhenFilePath(t *testing.T) {
	got := ResolveRemoteUploadPathForStaging("/tmp/out.png", "orig.png")
	if got != "/tmp/out.png" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveRemoteUploadPathForStaging_DirUnix(t *testing.T) {
	got := ResolveRemoteUploadPathForStaging("/var/www/uploads/", "photo.PNG")
	if got != "/var/www/uploads/photo.PNG" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveRemoteUploadPathForStaging_DirWindows(t *testing.T) {
	got := ResolveRemoteUploadPathForStaging(`C:\Users\Public\`, `C:\fake\image.png`)
	if got != `C:\Users\Public\image.png` {
		t.Fatalf("got %q", got)
	}
}

func TestResolveRemoteUploadPathForStaging_RootOnly(t *testing.T) {
	got := ResolveRemoteUploadPathForStaging("/", "a.bin")
	if got != "/a.bin" {
		t.Fatalf("got %q", got)
	}
}
