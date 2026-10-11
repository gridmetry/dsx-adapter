package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"gridmetry/dsx-adapter/internal/flex"
	"gridmetry/dsx-adapter/internal/lab"
)

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envIntOr(key string, def int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	return strconv.Atoi(v)
}

// requiredSecretFiles lists every file a complete init run leaves behind.
// signing-private.pem is last: generateSecrets writes it only after every
// other value has been generated, validated, and written, so its presence
// is the completion sentinel checked below.
var requiredSecretFiles = []string{
	"signing-public.pem",
	"signer-kid",
	"mqtt-publisher-credentials.json",
	"mqtt-bridge-credentials.json",
	"sse-token",
	"nats.conf",
	"signing-private.pem",
}

// secretsPresence stats each required secret file without reading its
// contents, returning how many of them exist. A partial result (neither
// zero nor the full set) means a previous run was interrupted or the
// directory was tampered with; callers must fail closed rather than
// regenerate over it.
func secretsPresence(secretsDir string) (present int, err error) {
	for _, name := range requiredSecretFiles {
		info, statErr := os.Stat(filepath.Join(secretsDir, name))
		if statErr != nil {
			if errors.Is(statErr, os.ErrNotExist) {
				continue
			}
			return present, fmt.Errorf("checking %s: %w", name, statErr)
		}
		if !info.Mode().IsRegular() || info.Size() == 0 || info.Size() > lab.MaxSecretFileBytes {
			return present, fmt.Errorf("%s failed validation", name)
		}
		present++
	}
	return present, nil
}

// runInit generates isolated local secrets/config into the shared secrets
// volume: an ES256 signing keypair, MQTT credentials for the lab
// publisher and the bridge, an SSE bearer token, and a derived nats.conf.
// It is idempotent: if every required secret file is already present, it
// leaves all trust material untouched rather than rotating it under a
// running broker/bridge. If only some of them are present it refuses to
// touch the directory at all, since silently filling in the rest or
// regenerating everything could overwrite still-valid partial secret
// material from an interrupted run.
func runInit(args []string) int {
	fs := flag.NewFlagSet("dsx-lab init", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}

	secretsDir := envOr("DSX_SECRETS_DIR", "/secrets")
	kid := envOr("DSX_SIGNER_KID", "gridmetry-lab-signer-1")
	topic := envOr("DSX_MQTT_TOPIC", "grid/v1/isv/gridmetry/loadtarget/set")
	publisherUser := envOr("DSX_PUBLISHER_USER", "gridmetry")
	bridgeUser := envOr("DSX_BRIDGE_USER", "bridge")
	mqttListenAddr := envOr("DSX_MQTT_LISTEN_ADDR", "0.0.0.0:1883")
	httpMonitorAddr := envOr("DSX_HTTP_MONITOR_ADDR", "0.0.0.0:8222")
	jetstreamDir := envOr("DSX_JETSTREAM_DIR", "/data")
	uid, err := envIntOr("DSX_OWNER_UID", 10001)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dsx-lab init: invalid DSX_OWNER_UID: %v\n", err)
		return 2
	}
	gid, err := envIntOr("DSX_OWNER_GID", 10001)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dsx-lab init: invalid DSX_OWNER_GID: %v\n", err)
		return 2
	}

	present, err := secretsPresence(secretsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dsx-lab init: checking existing state: %v\n", err)
		return 1
	}
	switch present {
	case len(requiredSecretFiles):
		fmt.Println("dsx-lab init: secrets already present, leaving existing trust material unchanged")
		return 0
	case 0:
		// fall through to generation below
	default:
		fmt.Fprintln(os.Stderr, "dsx-lab init: secrets directory has an incomplete file set from a previous run; refusing to regenerate or overwrite it")
		return 1
	}

	if err := lab.EnsureOwnedDir(secretsDir, 0o700, uid, gid); err != nil {
		fmt.Fprintf(os.Stderr, "dsx-lab init: %v\n", err)
		return 1
	}

	if err := generateSecrets(secretsDir, kid, topic, publisherUser, bridgeUser,
		mqttListenAddr, httpMonitorAddr, jetstreamDir, uid, gid); err != nil {
		fmt.Fprintf(os.Stderr, "dsx-lab init: %v\n", err)
		return 1
	}

	fmt.Println("dsx-lab init: generated signing key, MQTT credentials, SSE token, and nats.conf")
	return 0
}

func generateSecrets(secretsDir, kid, topic, publisherUser, bridgeUser,
	mqttListenAddr, httpMonitorAddr, jetstreamDir string, uid, gid int) error {
	key, err := flex.GenerateSigningKey()
	if err != nil {
		return fmt.Errorf("generating signing key: %w", err)
	}
	privPEM, err := flex.EncodePrivateKeyPEM(key)
	if err != nil {
		return err
	}
	pubPEM, err := flex.EncodePublicKeyPEM(&key.PublicKey)
	if err != nil {
		return err
	}

	publisherPass, err := lab.GenerateToken(24)
	if err != nil {
		return err
	}
	bridgePass, err := lab.GenerateToken(24)
	if err != nil {
		return err
	}
	sseToken, err := lab.GenerateToken(24)
	if err != nil {
		return err
	}

	publisherCreds, err := json.Marshal(lab.MQTTCredentials{Username: publisherUser, Password: publisherPass})
	if err != nil {
		return err
	}
	bridgeCreds, err := json.Marshal(lab.MQTTCredentials{Username: bridgeUser, Password: bridgePass})
	if err != nil {
		return err
	}

	subject := lab.MQTTTopicToNATSSubject(topic)
	natsConf, err := lab.RenderNATSConfig(lab.NATSConfigParams{
		MQTTListenAddr:  mqttListenAddr,
		HTTPMonitorAddr: httpMonitorAddr,
		JetStreamDir:    jetstreamDir,
		PublisherUser:   publisherUser,
		PublisherPass:   publisherPass,
		BridgeUser:      bridgeUser,
		BridgePass:      bridgePass,
		CommandSubject:  subject,
	})
	if err != nil {
		return fmt.Errorf("rendering nats.conf: %w", err)
	}

	// Everything above is generated and validated in memory; only now do
	// we touch disk, writing signing-private.pem (the completion
	// sentinel) last so a failure partway through never leaves it behind
	// without the rest of the set.
	if err := lab.WriteOwnedFile(filepath.Join(secretsDir, "signing-public.pem"), pubPEM, 0o640, uid, gid); err != nil {
		return err
	}
	if err := lab.WriteOwnedFile(filepath.Join(secretsDir, "signer-kid"), []byte(kid+"\n"), 0o640, uid, gid); err != nil {
		return err
	}
	if err := lab.WriteOwnedFile(filepath.Join(secretsDir, "mqtt-publisher-credentials.json"), publisherCreds, 0o600, uid, gid); err != nil {
		return err
	}
	if err := lab.WriteOwnedFile(filepath.Join(secretsDir, "mqtt-bridge-credentials.json"), bridgeCreds, 0o600, uid, gid); err != nil {
		return err
	}
	if err := lab.WriteOwnedFile(filepath.Join(secretsDir, "sse-token"), []byte(sseToken+"\n"), 0o600, uid, gid); err != nil {
		return err
	}
	if err := lab.WriteOwnedFile(filepath.Join(secretsDir, "nats.conf"), []byte(natsConf), 0o640, uid, gid); err != nil {
		return err
	}
	if err := lab.WriteOwnedFile(filepath.Join(secretsDir, "signing-private.pem"), privPEM, 0o600, uid, gid); err != nil {
		return err
	}

	return nil
}
