package nvcf

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTempKeyFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "fake.key")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write temp fixture: %v", err)
	}
	return path
}

func TestLoadBearerToken_Valid(t *testing.T) {
	path := writeTempKeyFile(t, "fake-test-token-value-123\n")
	token, err := LoadBearerToken(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "fake-test-token-value-123" {
		t.Fatalf("expected trimmed token, got %q", token)
	}
}

func TestLoadBearerToken_MissingFile(t *testing.T) {
	_, err := LoadBearerToken(filepath.Join(t.TempDir(), "does-not-exist.key"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if strings.Contains(err.Error(), "does-not-exist") {
		t.Fatalf("error should not echo file path content: %v", err)
	}
}

func TestLoadBearerToken_Empty(t *testing.T) {
	path := writeTempKeyFile(t, "   \n\t  ")
	_, err := LoadBearerToken(path)
	if err == nil {
		t.Fatal("expected error for empty/whitespace-only credential")
	}
}

func TestLoadBearerToken_Oversized(t *testing.T) {
	path := writeTempKeyFile(t, strings.Repeat("a", maxKeyFileBytes+1000))
	_, err := LoadBearerToken(path)
	if err == nil {
		t.Fatal("expected error for oversized credential file")
	}
	if strings.Contains(err.Error(), strings.Repeat("a", 20)) {
		t.Fatalf("error must not echo file contents: %v", err)
	}
}

func TestLoadBearerToken_MultipleTokens(t *testing.T) {
	path := writeTempKeyFile(t, "fake-secret-one fake-secret-two")
	_, err := LoadBearerToken(path)
	if err == nil {
		t.Fatal("expected error when credential contains internal whitespace")
	}
}

func TestLoadBearerToken_BadCharacters(t *testing.T) {
	path := writeTempKeyFile(t, "fake-secret-\x01-value")
	_, err := LoadBearerToken(path)
	if err == nil {
		t.Fatal("expected error for non-printable characters")
	}
}

func TestLoadBearerToken_RejectsDisallowedPunctuation(t *testing.T) {
	for _, bad := range []string{
		`fake-secret-"quoted"`,
		"fake-secret-;rm -rf",
		"fake-secret-<script>",
		"fake-secret-|pipe",
		"fake-secret-$(cmd)",
		"fake-secret-`cmd`",
	} {
		path := writeTempKeyFile(t, bad)
		if _, err := LoadBearerToken(path); err == nil {
			t.Fatalf("expected error for disallowed punctuation in %q", bad)
		}
	}
}

func TestLoadBearerToken_AllowsBearerCharset(t *testing.T) {
	path := writeTempKeyFile(t, "fake.Token_value-123")
	token, err := LoadBearerToken(path)
	if err != nil {
		t.Fatalf("unexpected error for alnum/._- token: %v", err)
	}
	if token != "fake.Token_value-123" {
		t.Fatalf("expected trimmed token, got %q", token)
	}
}

func TestLoadBearerToken_ErrorsNeverEchoSecretContent(t *testing.T) {
	secret := "super-fake-leaked-credential-should-not-appear"
	path := writeTempKeyFile(t, secret+" "+secret)
	_, err := LoadBearerToken(path)
	if err == nil {
		t.Fatal("expected error for malformed multi-token credential")
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("error message leaked secret content: %v", err)
	}
}
