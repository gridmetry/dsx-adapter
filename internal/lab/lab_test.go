package lab

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestGenerateTokenLengthAndCharset(t *testing.T) {
	tok, err := GenerateToken(32)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	decoded, err := base64.RawURLEncoding.DecodeString(tok)
	if err != nil {
		t.Fatalf("token is not valid base64url: %v", err)
	}
	if len(decoded) != 32 {
		t.Fatalf("expected 32 bytes of entropy, got %d", len(decoded))
	}
}

func TestGenerateTokenIsNotConstant(t *testing.T) {
	a, _ := GenerateToken(16)
	b, _ := GenerateToken(16)
	if a == b {
		t.Fatal("expected distinct random tokens")
	}
}

func TestWriteOwnedFileSetsPermissionsAndOwner(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "secret")
	if err := WriteOwnedFile(path, []byte("x"), 0o600, os.Getuid(), os.Getgid()); err != nil {
		t.Fatalf("WriteOwnedFile: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("expected mode 0600, got %v", info.Mode().Perm())
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if ok && int(st.Uid) != os.Getuid() {
		t.Fatalf("expected owner uid %d, got %d", os.Getuid(), st.Uid)
	}
}

func TestReadBoundedFileRejectsOversized(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "big")
	big := make([]byte, MaxSecretFileBytes+1)
	if err := os.WriteFile(path, big, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := ReadBoundedFile(path); err == nil {
		t.Fatal("expected rejection of oversized secret file")
	}
}

func TestReadBoundedFileTrimsTrailingNewline(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tok")
	if err := os.WriteFile(path, []byte("abc123\n"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	got, err := ReadBoundedFile(path)
	if err != nil {
		t.Fatalf("ReadBoundedFile: %v", err)
	}
	if string(got) != "abc123" {
		t.Fatalf("expected trimmed value, got %q", got)
	}
}

func TestMQTTTopicToNATSSubject(t *testing.T) {
	got := MQTTTopicToNATSSubject("grid/v1/isv/gridmetry/loadtarget/set")
	want := "grid.v1.isv.gridmetry.loadtarget.set"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func validNATSParams() NATSConfigParams {
	return NATSConfigParams{
		MQTTListenAddr:  "0.0.0.0:1883",
		HTTPMonitorAddr: "0.0.0.0:8222",
		JetStreamDir:    "/data",
		PublisherUser:   "gridmetry",
		PublisherPass:   "abcDEF123_-",
		BridgeUser:      "bridge",
		BridgePass:      "xyzXYZ789_-",
		CommandSubject:  "grid.v1.isv.gridmetry.loadtarget.set",
	}
}

func TestRenderNATSConfigIncludesExpectedStanzas(t *testing.T) {
	conf, err := RenderNATSConfig(validNATSParams())
	if err != nil {
		t.Fatalf("RenderNATSConfig: %v", err)
	}
	for _, want := range []string{
		`store_dir: "/data"`,
		`listen: "0.0.0.0:1883"`,
		`user: "gridmetry"`,
		`user: "bridge"`,
		`allow: ["grid.v1.isv.gridmetry.loadtarget.set"]`,
		"jetstream: enabled",
	} {
		if !strings.Contains(conf, want) {
			t.Fatalf("expected generated config to contain %q, got:\n%s", want, conf)
		}
	}
}

func TestRenderNATSConfigRejectsUnsafeCharacters(t *testing.T) {
	p := validNATSParams()
	p.PublisherPass = `abc" } accounts: { EVIL: {}`
	if _, err := RenderNATSConfig(p); err == nil {
		t.Fatal("expected rejection of unsafe characters in generated config value")
	}
}

func TestLoadMQTTCredentialsRejectsIncomplete(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "creds.json")
	if err := os.WriteFile(path, []byte(`{"username":"bridge"}`), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, _, err := loadMQTTCredentials(path); err == nil {
		t.Fatal("expected rejection of incomplete credentials file")
	}
}

func TestLoadMQTTCredentialsReadsUsernamePassword(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "creds.json")
	if err := os.WriteFile(path, []byte(`{"username":"bridge","password":"secret"}`), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	user, pass, err := loadMQTTCredentials(path)
	if err != nil {
		t.Fatalf("loadMQTTCredentials: %v", err)
	}
	if user != "bridge" || pass != "secret" {
		t.Fatalf("got (%q, %q)", user, pass)
	}
}
