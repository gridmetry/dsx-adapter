package main

import (
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
)

// captureOutput swaps os.Stdout/os.Stderr for the duration of fn so tests
// can assert on exactly what dsx-lab printed without touching the real
// process streams.
func captureOutput(t *testing.T, fn func() int) (stdout, stderr string, code int) {
	t.Helper()
	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	errR, errW, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	oldOut, oldErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = outW, errW
	code = fn()
	outW.Close()
	errW.Close()
	os.Stdout, os.Stderr = oldOut, oldErr
	outBytes, _ := io.ReadAll(outR)
	errBytes, _ := io.ReadAll(errR)
	return string(outBytes), string(errBytes), code
}

// setInitEnv points dsx-lab init at an isolated temp directory and an
// owner uid/gid the test process can actually chown to, so the test never
// needs real privilege elevation or a real secrets path.
func setInitEnv(t *testing.T, secretsDir string) {
	t.Helper()
	t.Setenv("DSX_SECRETS_DIR", secretsDir)
	t.Setenv("DSX_OWNER_UID", strconv.Itoa(os.Getuid()))
	t.Setenv("DSX_OWNER_GID", strconv.Itoa(os.Getgid()))
	t.Setenv("DSX_SIGNER_KID", "test-signer-1")
	t.Setenv("DSX_MQTT_TOPIC", "grid/v1/isv/gridmetry/loadtarget/set")
	t.Setenv("DSX_PUBLISHER_USER", "gridmetry")
	t.Setenv("DSX_BRIDGE_USER", "bridge")
	t.Setenv("DSX_MQTT_LISTEN_ADDR", "0.0.0.0:1883")
	t.Setenv("DSX_HTTP_MONITOR_ADDR", "0.0.0.0:8222")
	t.Setenv("DSX_JETSTREAM_DIR", "/data")
}

func TestRunInitFailsClosedOnPartialSecrets(t *testing.T) {
	dir := t.TempDir()
	setInitEnv(t, dir)
	// Simulate an interrupted prior run: signing-private.pem exists but
	// every other required file is missing.
	if err := os.WriteFile(filepath.Join(dir, "signing-private.pem"), []byte("stale-key-bytes"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	stdout, stderr, code := captureOutput(t, func() int { return runInit(nil) })
	if code == 0 {
		t.Fatalf("expected a nonzero exit code for a partial secrets directory, got 0 (stdout=%q stderr=%q)", stdout, stderr)
	}
	for _, name := range requiredSecretFiles {
		if name == "signing-private.pem" {
			continue
		}
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			t.Fatalf("expected %s not to be generated when failing closed on a partial directory", name)
		}
	}
	got, err := os.ReadFile(filepath.Join(dir, "signing-private.pem"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "stale-key-bytes" {
		t.Fatal("expected the pre-existing partial file to be left untouched")
	}
}

func TestRunInitCompleteSecretsIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	setInitEnv(t, dir)

	if _, _, code := captureOutput(t, func() int { return runInit(nil) }); code != 0 {
		t.Fatalf("expected first init run to succeed, got exit code %d", code)
	}

	before := map[string][]byte{}
	for _, name := range requiredSecretFiles {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("reading %s after first run: %v", name, err)
		}
		before[name] = data
	}

	if _, _, code := captureOutput(t, func() int { return runInit(nil) }); code != 0 {
		t.Fatalf("expected a second init run over complete secrets to succeed, got exit code %d", code)
	}

	for _, name := range requiredSecretFiles {
		after, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("reading %s after second run: %v", name, err)
		}
		if string(after) != string(before[name]) {
			t.Fatalf("expected %s to be left unchanged by a second init run over complete secrets", name)
		}
	}
}

func TestRunInitValidatesBeforeWritingAnyFile(t *testing.T) {
	dir := t.TempDir()
	setInitEnv(t, dir)
	// A topic containing characters outside the generated-config charset
	// makes nats.conf rendering fail; this must be detected before any
	// secret file is written.
	t.Setenv("DSX_MQTT_TOPIC", `grid/v1/"; accounts: { EVIL: {} }`)

	if _, _, code := captureOutput(t, func() int { return runInit(nil) }); code == 0 {
		t.Fatal("expected init to fail when the rendered nats.conf would be invalid")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected no files to be written when validation fails before publishing, found %v", entries)
	}
}

func TestRunInitSetsOwnerAndPermissions(t *testing.T) {
	dir := t.TempDir()
	setInitEnv(t, dir)

	if _, _, code := captureOutput(t, func() int { return runInit(nil) }); code != 0 {
		t.Fatalf("expected init to succeed, got exit code %d", code)
	}

	dirInfo, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("Stat dir: %v", err)
	}
	if dirInfo.Mode().Perm() != 0o700 {
		t.Fatalf("expected secrets dir mode 0700, got %v", dirInfo.Mode().Perm())
	}

	wantPerm := map[string]os.FileMode{
		"signing-private.pem":             0o600,
		"signing-public.pem":              0o640,
		"signer-kid":                      0o640,
		"mqtt-publisher-credentials.json": 0o600,
		"mqtt-bridge-credentials.json":    0o600,
		"sse-token":                       0o600,
		"nats.conf":                       0o640,
	}
	for name, perm := range wantPerm {
		info, err := os.Stat(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("Stat %s: %v", name, err)
		}
		if info.Mode().Perm() != perm {
			t.Fatalf("expected %s mode %v, got %v", name, perm, info.Mode().Perm())
		}
		if st, ok := info.Sys().(*syscall.Stat_t); ok {
			if int(st.Uid) != os.Getuid() || int(st.Gid) != os.Getgid() {
				t.Fatalf("expected %s owned by %d:%d, got %d:%d", name, os.Getuid(), os.Getgid(), st.Uid, st.Gid)
			}
		}
	}
}

func TestRunInitNeverPrintsGeneratedSecretValues(t *testing.T) {
	dir := t.TempDir()
	setInitEnv(t, dir)

	stdout, stderr, code := captureOutput(t, func() int { return runInit(nil) })
	if code != 0 {
		t.Fatalf("expected init to succeed, got exit code %d", code)
	}

	for _, name := range []string{
		"signing-private.pem",
		"mqtt-publisher-credentials.json",
		"mqtt-bridge-credentials.json",
		"sse-token",
		"nats.conf",
	} {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("reading %s: %v", name, err)
		}
		content := string(data)
		if content == "" {
			continue
		}
		if strings.Contains(stdout, content) || strings.Contains(stderr, content) {
			t.Fatalf("expected %s contents never printed to stdout/stderr", name)
		}
	}
}
