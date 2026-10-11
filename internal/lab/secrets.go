package lab

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
)

// GenerateToken returns a URL-safe random token with nBytes of entropy
// before encoding. The alphabet (base64url) contains no quote, backslash,
// or whitespace characters, so it is safe to embed directly in generated
// JSON and NATS config values without additional escaping.
func GenerateToken(nBytes int) (string, error) {
	buf := make([]byte, nBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("lab: generating random token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// WriteOwnedFile writes data to path with perm, then chowns it to
// uid:gid. Chown requires the calling process to be running as root (the
// init container only); non-root callers should omit the chown step by
// not using this helper.
func WriteOwnedFile(path string, data []byte, perm os.FileMode, uid, gid int) error {
	if err := os.WriteFile(path, data, perm); err != nil {
		return fmt.Errorf("lab: writing %s: %w", path, err)
	}
	if err := os.Chown(path, uid, gid); err != nil {
		return fmt.Errorf("lab: chowning %s: %w", path, err)
	}
	return nil
}

// EnsureOwnedDir creates dir (and parents) with perm, then chowns it.
func EnsureOwnedDir(dir string, perm os.FileMode, uid, gid int) error {
	if err := os.MkdirAll(dir, perm); err != nil {
		return fmt.Errorf("lab: creating %s: %w", dir, err)
	}
	if err := os.Chmod(dir, perm); err != nil {
		return fmt.Errorf("lab: chmod %s: %w", dir, err)
	}
	if err := os.Chown(dir, uid, gid); err != nil {
		return fmt.Errorf("lab: chowning %s: %w", dir, err)
	}
	return nil
}
