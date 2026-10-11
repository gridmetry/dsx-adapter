package bridge

import (
	"errors"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// fakeSubackToken is a minimal fake of the narrow subscribeResult surface
// (mqtt.Token plus Result()) that handleConnect reads after a SUBACK. The
// concrete *mqtt.SubscribeToken type has no exported constructor and
// unexported fields, so it cannot be fabricated directly in tests; this
// fake lets the reconnect/readiness decision be tested without a broker.
type fakeSubackToken struct {
	timedOut bool
	err      error
	result   map[string]byte
}

func (f *fakeSubackToken) Wait() bool                     { return !f.timedOut }
func (f *fakeSubackToken) WaitTimeout(time.Duration) bool { return !f.timedOut }
func (f *fakeSubackToken) Done() <-chan struct{} {
	ch := make(chan struct{})
	if !f.timedOut {
		close(ch)
	}
	return ch
}
func (f *fakeSubackToken) Error() error            { return f.err }
func (f *fakeSubackToken) Result() map[string]byte { return f.result }

// fakeSubscribeClient is a minimal fake of the narrow subscribeClient
// surface (just Subscribe) handleConnect needs from mqtt.Client.
type fakeSubscribeClient struct {
	token mqtt.Token
}

func (f *fakeSubscribeClient) Subscribe(topic string, qos byte, callback mqtt.MessageHandler) mqtt.Token {
	return f.token
}

func TestHandleConnectReadinessFromSUBACKResult(t *testing.T) {
	const topic = "grid/v1/isv/gridmetry/loadtarget/set"
	cases := []struct {
		name      string
		token     *fakeSubackToken
		wantReady bool
	}{
		{"granted QoS 1", &fakeSubackToken{result: map[string]byte{topic: 1}}, true},
		{"denied 0x80", &fakeSubackToken{result: map[string]byte{topic: 0x80}}, false},
		{"lowered to QoS 0", &fakeSubackToken{result: map[string]byte{topic: 0}}, false},
		{"missing topic in result", &fakeSubackToken{result: map[string]byte{}}, false},
		{"subscribe error", &fakeSubackToken{err: errors.New("boom"), result: map[string]byte{topic: 1}}, false},
		{"suback wait timed out", &fakeSubackToken{timedOut: true, result: map[string]byte{topic: 1}}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := &MQTTSubscriber{cfg: MQTTConfig{Topic: topic}}
			s.handleConnect(&fakeSubscribeClient{token: c.token})
			if got := s.Ready(); got != c.wantReady {
				t.Fatalf("Ready() = %v, want %v", got, c.wantReady)
			}
		})
	}
}

func TestReadinessDropsOnConnectionLossAndReturnsAfterResubscribe(t *testing.T) {
	const topic = "grid/v1/isv/gridmetry/loadtarget/set"
	s := &MQTTSubscriber{cfg: MQTTConfig{Topic: topic}}

	s.handleConnect(&fakeSubscribeClient{token: &fakeSubackToken{result: map[string]byte{topic: 1}}})
	if !s.Ready() {
		t.Fatal("expected ready after the first granted SUBACK")
	}

	s.onConnectionLost(nil, errors.New("connection lost"))
	if s.Ready() {
		t.Fatal("expected not ready immediately after a connection loss")
	}

	// A reconnect that fails to regain QoS 1 must stay not ready.
	s.handleConnect(&fakeSubscribeClient{token: &fakeSubackToken{result: map[string]byte{topic: 0}}})
	if s.Ready() {
		t.Fatal("expected not ready after a reconnect that only regained QoS 0")
	}

	s.handleConnect(&fakeSubscribeClient{token: &fakeSubackToken{result: map[string]byte{topic: 1}}})
	if !s.Ready() {
		t.Fatal("expected ready again after a reconnect that regained QoS 1")
	}
}

// TestNewMQTTSubscriberBoundsReconnectBackoff guards against Paho's default
// MaxReconnectInterval (10 minutes): without an explicit bound, a broker
// restart in a short-lived lab/test run can leave the client backing off
// far longer than any bounded test timeout waits, which looks identical to
// a permanently broken reconnect. ConnectTimeout bounds how long a single
// (re)connect attempt itself may block.
func TestNewMQTTSubscriberBoundsReconnectBackoff(t *testing.T) {
	s := NewMQTTSubscriber(MQTTConfig{BrokerURL: "tcp://127.0.0.1:1", Topic: "grid/v1/isv/gridmetry/loadtarget/set"}, nil)
	reader := s.client.OptionsReader()
	if got := reader.MaxReconnectInterval(); got != 2*time.Second {
		t.Fatalf("MaxReconnectInterval() = %v, want 2s", got)
	}
	if got := reader.ConnectTimeout(); got != 5*time.Second {
		t.Fatalf("ConnectTimeout() = %v, want 5s", got)
	}
}
