package lab

import (
	"encoding/json"
	"fmt"
)

// MQTTCredentials is the generated-at-init JSON shape for one MQTT user.
// Password is only ever written to a 0600 file in the shared secrets
// volume; it is never logged.
type MQTTCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func loadMQTTCredentials(path string) (username, password string, err error) {
	data, err := ReadBoundedFile(path)
	if err != nil {
		return "", "", fmt.Errorf("lab: reading MQTT credentials: %w", err)
	}
	var creds MQTTCredentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return "", "", fmt.Errorf("lab: parsing MQTT credentials: %w", err)
	}
	if creds.Username == "" || creds.Password == "" {
		return "", "", fmt.Errorf("lab: MQTT credentials file %s is incomplete", path)
	}
	return creds.Username, creds.Password, nil
}
