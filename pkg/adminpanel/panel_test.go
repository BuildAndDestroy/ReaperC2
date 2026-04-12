package adminpanel

import (
	"os"
	"testing"
)

func TestResolveBeaconBaseURL_EmptyUsesEnv(t *testing.T) {
	_ = os.Setenv("BEACON_PUBLIC_BASE_URL", "http://env-default:9999")
	defer os.Unsetenv("BEACON_PUBLIC_BASE_URL")
	got, err := ResolveBeaconBaseURL("")
	if err != nil {
		t.Fatal(err)
	}
	if got != "http://env-default:9999" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveBeaconBaseURL_HostPort(t *testing.T) {
	got, err := ResolveBeaconBaseURL("10.0.0.5:8443")
	if err != nil {
		t.Fatal(err)
	}
	if got != "http://10.0.0.5:8443" {
		t.Fatalf("got %q want http://10.0.0.5:8443", got)
	}
}

func TestResolveBeaconBaseURL_FullHTTPS(t *testing.T) {
	got, err := ResolveBeaconBaseURL("https://c2.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://c2.example.com" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveBeaconBaseURL_StripsPath(t *testing.T) {
	got, err := ResolveBeaconBaseURL("http://192.168.1.1:8080/extra/path")
	if err != nil {
		t.Fatal(err)
	}
	if got != "http://192.168.1.1:8080" {
		t.Fatalf("got %q want origin only", got)
	}
}

func TestResolveBeaconBaseURL_RejectsFTP(t *testing.T) {
	_, err := ResolveBeaconBaseURL("ftp://1.2.3.4:21")
	if err == nil {
		t.Fatal("expected error")
	}
}
