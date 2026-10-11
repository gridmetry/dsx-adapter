package lab

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadBridgeConfigReadsEnvAndFiles(t *testing.T) {
	dir := t.TempDir()
	credsPath := filepath.Join(dir, "mqtt-bridge-credentials.json")
	if err := os.WriteFile(credsPath, []byte(`{"username":"bridge","password":"pw"}`), 0o600); err != nil {
		t.Fatalf("WriteFile creds: %v", err)
	}
	tokenPath := filepath.Join(dir, "sse-token")
	if err := os.WriteFile(tokenPath, []byte("sse-tok-value\n"), 0o600); err != nil {
		t.Fatalf("WriteFile token: %v", err)
	}

	t.Setenv("DSX_SIGNER_KID", "lab-signer-1")
	t.Setenv("DSX_MQTT_CREDENTIALS_PATH", credsPath)
	t.Setenv("DSX_SSE_TOKEN_PATH", tokenPath)

	cfg, err := LoadBridgeConfig()
	if err != nil {
		t.Fatalf("LoadBridgeConfig: %v", err)
	}
	if cfg.MQTTUsername != "bridge" || cfg.MQTTPassword != "pw" {
		t.Fatalf("unexpected MQTT credentials: %+v", cfg)
	}
	if cfg.SSEToken != "sse-tok-value" {
		t.Fatalf("unexpected SSE token: %q", cfg.SSEToken)
	}
	if cfg.SignerKeyID != "lab-signer-1" {
		t.Fatalf("unexpected signer kid: %q", cfg.SignerKeyID)
	}
}

func TestLoadBridgeConfigRequiresSignerKid(t *testing.T) {
	t.Setenv("DSX_SIGNER_KID", "")
	if _, err := LoadBridgeConfig(); err == nil {
		t.Fatal("expected error when DSX_SIGNER_KID is unset")
	}
}
