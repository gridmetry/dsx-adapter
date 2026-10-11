// Config loading for the lab bridge/scenario binaries. Values come from
// bounded environment variables and small local files mounted read-only; no
// value is ever logged, and no sibling-repo file is read at runtime.
package lab

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// MaxSecretFileBytes bounds any single secret/config file this process
// will read (keys, tokens, credential JSON).
const MaxSecretFileBytes = 8 * 1024

// ReadBoundedFile reads path, rejecting anything larger than
// MaxSecretFileBytes, and trims a single trailing newline if present.
func ReadBoundedFile(path string) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.Size() > MaxSecretFileBytes {
		return nil, fmt.Errorf("lab: %s exceeds %d bytes", path, MaxSecretFileBytes)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return []byte(strings.TrimSuffix(string(data), "\n")), nil
}

// BridgeConfig configures the MQTT bridge/HTTP service binary.
type BridgeConfig struct {
	MQTTBrokerURL   string
	MQTTClientID    string
	MQTTUsername    string
	MQTTPassword    string
	MQTTTopic       string
	FeedTag         string
	MaxWatts        float64
	MaxEventAge     time.Duration
	MaxClockSkew    time.Duration
	SignerKeyID     string
	SignerSource    string
	PublicKeyPath   string
	HTTPAddr        string
	SSEToken        string
	CORSAllowOrigin string
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getenvFloat(key string, def float64) (float64, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, fmt.Errorf("lab: invalid %s: %w", key, err)
	}
	return f, nil
}

func getenvDuration(key string, def time.Duration) (time.Duration, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("lab: invalid %s: %w", key, err)
	}
	return d, nil
}

// LoadBridgeConfig reads bridge configuration from environment variables
// and the mounted credential/key/token files they reference.
func LoadBridgeConfig() (BridgeConfig, error) {
	cfg := BridgeConfig{
		MQTTBrokerURL:   getenv("DSX_MQTT_BROKER_URL", "tcp://broker:1883"),
		MQTTClientID:    getenv("DSX_MQTT_CLIENT_ID", "dsx-bridge"),
		MQTTTopic:       getenv("DSX_MQTT_TOPIC", "grid/v1/isv/gridmetry/loadtarget/set"),
		FeedTag:         getenv("DSX_FEED_TAG", "main-a"),
		SignerKeyID:     getenv("DSX_SIGNER_KID", ""),
		SignerSource:    getenv("DSX_SIGNER_SOURCE", "//grid/v1/isv/gridmetry"),
		PublicKeyPath:   getenv("DSX_SIGNING_PUBLIC_KEY_PATH", "/secrets/signing-public.pem"),
		HTTPAddr:        getenv("DSX_HTTP_ADDR", "0.0.0.0:8090"),
		CORSAllowOrigin: getenv("DSX_CORS_ALLOW_ORIGIN", ""),
	}
	if cfg.SignerKeyID == "" {
		return cfg, errors.New("lab: DSX_SIGNER_KID is required")
	}

	var err error
	cfg.MaxWatts, err = getenvFloat("DSX_MAX_WATTS", 1_000_000)
	if err != nil {
		return cfg, err
	}
	cfg.MaxEventAge, err = getenvDuration("DSX_MAX_EVENT_AGE", 5*time.Minute)
	if err != nil {
		return cfg, err
	}
	cfg.MaxClockSkew, err = getenvDuration("DSX_MAX_CLOCK_SKEW", 30*time.Second)
	if err != nil {
		return cfg, err
	}

	credPath := getenv("DSX_MQTT_CREDENTIALS_PATH", "/secrets/mqtt-bridge-credentials.json")
	user, pass, err := loadMQTTCredentials(credPath)
	if err != nil {
		return cfg, err
	}
	cfg.MQTTUsername, cfg.MQTTPassword = user, pass

	tokenPath := getenv("DSX_SSE_TOKEN_PATH", "/secrets/sse-token")
	tokenBytes, err := ReadBoundedFile(tokenPath)
	if err != nil {
		return cfg, fmt.Errorf("lab: reading SSE token: %w", err)
	}
	cfg.SSEToken = string(tokenBytes)

	return cfg, nil
}
