// Package nvcf provides a bounded, read-only client for NVIDIA Cloud
// Functions (NVCF) discovery endpoints used by the dsx-access CLI.
package nvcf

import (
	"errors"
	"io"
	"os"
	"regexp"
	"strings"
)

// maxKeyFileBytes bounds how much of a credential file is ever read into
// memory, regardless of the file's actual size on disk.
const maxKeyFileBytes = 8192

const (
	minTokenLen = 10
	maxTokenLen = 4096
)

// validTokenCharset is the exact bearer-token character set this CLI
// accepts: alphanumeric plus '.', '_', '-'. Any other character (quotes,
// semicolons, shell metacharacters, control characters, etc.) is rejected
// before the value is ever used as a credential.
var validTokenCharset = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// LoadBearerToken reads a bearer token from a local file. It never returns
// the file's contents in error messages, and bounds how much data it reads
// so an oversized file cannot be loaded into memory.
func LoadBearerToken(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", errors.New("credential file not accessible")
	}
	defer f.Close()

	limited := io.LimitReader(f, maxKeyFileBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return "", errors.New("credential file not accessible")
	}
	if len(data) > maxKeyFileBytes {
		return "", errors.New("credential file too large")
	}

	token := strings.TrimSpace(string(data))
	if token == "" {
		return "", errors.New("credential file is empty")
	}
	if strings.ContainsAny(token, " \t\r\n") {
		return "", errors.New("credential value must be a single token")
	}
	if len(token) < minTokenLen || len(token) > maxTokenLen {
		return "", errors.New("credential value has an unexpected length")
	}
	if !validTokenCharset.MatchString(token) {
		return "", errors.New("credential value has unexpected characters")
	}
	return token, nil
}
