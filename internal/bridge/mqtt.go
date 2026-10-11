package bridge

import (
	"errors"
	"log"
	"sync/atomic"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// MQTTConfig configures the bridge's single subscription.
type MQTTConfig struct {
	BrokerURL      string
	ClientID       string
	Username       string
	Password       string
	Topic          string
	ConnectTimeout time.Duration
}

// MQTTSubscriber owns the paho client, subscribing once connected and
// re-subscribing reliably on every reconnect. Readiness (for /readyz)
// reflects only "currently subscribed", never a cached historical state.
type MQTTSubscriber struct {
	cfg      MQTTConfig
	pipeline *Pipeline
	client   mqtt.Client
	ready    atomic.Bool
}

// NewMQTTSubscriber builds (but does not connect) a subscriber that feeds
// pipeline.Enqueue from the MQTT message callback.
func NewMQTTSubscriber(cfg MQTTConfig, pipeline *Pipeline) *MQTTSubscriber {
	s := &MQTTSubscriber{cfg: cfg, pipeline: pipeline}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(cfg.BrokerURL)
	opts.SetClientID(cfg.ClientID)
	opts.SetUsername(cfg.Username)
	opts.SetPassword(cfg.Password)
	opts.SetProtocolVersion(4) // MQTT 3.1.1
	opts.SetCleanSession(false)
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(2 * time.Second)
	opts.SetOnConnectHandler(s.onConnect)
	opts.SetConnectionLostHandler(s.onConnectionLost)
	opts.SetKeepAlive(30 * time.Second)
	// Bound Paho's own per-attempt dial timeout and reconnect backoff
	// ceiling: the library's defaults (30s connect, 10m max backoff) can
	// leave a short-lived lab/test run waiting far longer than any bounded
	// test timeout for a reconnect after a broker restart.
	opts.SetConnectTimeout(5 * time.Second)
	opts.SetMaxReconnectInterval(2 * time.Second)

	s.client = mqtt.NewClient(opts)
	return s
}

// Connect blocks (bounded by cfg.ConnectTimeout) until the initial
// connection attempt completes, reporting its error if any. Subsequent
// reconnects are handled automatically by the paho client plus
// onConnect's resubscribe.
func (s *MQTTSubscriber) Connect() error {
	timeout := s.cfg.ConnectTimeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	token := s.client.Connect()
	if !token.WaitTimeout(timeout) {
		return errors.New("bridge: timed out connecting to MQTT broker")
	}
	return token.Error()
}

// Ready reports whether the subscription is currently confirmed active.
// It is false before the first SUBACK and immediately after a connection
// loss, so /readyz never claims readiness the broker cannot currently back.
func (s *MQTTSubscriber) Ready() bool {
	return s.ready.Load()
}

// Close disconnects, allowing up to quiesceMillis for in-flight work.
func (s *MQTTSubscriber) Close(quiesceMillis uint) {
	s.ready.Store(false)
	s.client.Disconnect(quiesceMillis)
}

// subscribeClient is the narrow slice of mqtt.Client's surface onConnect
// needs (just Subscribe), so the reconnect/readiness decision is
// unit-testable with a fake client, without a broker.
type subscribeClient interface {
	Subscribe(topic string, qos byte, callback mqtt.MessageHandler) mqtt.Token
}

// subscribeResult is the narrow slice of *mqtt.SubscribeToken's surface
// read after a SUBACK. The concrete paho type has unexported fields and no
// exported constructor, so tests fake this interface instead.
type subscribeResult interface {
	mqtt.Token
	Result() map[string]byte
}

func (s *MQTTSubscriber) onConnectionLost(_ mqtt.Client, err error) {
	s.ready.Store(false)
	log.Printf("bridge: MQTT connection lost: %v", err)
}

func (s *MQTTSubscriber) onConnect(client mqtt.Client) {
	s.handleConnect(client)
}

// handleConnect subscribes and only marks the subscriber ready once the
// broker's SUBACK confirms exactly the requested QoS 1 for the topic: a
// denied (0x80), lowered, or missing grant leaves readiness false, as does
// a subscribe error or a timed-out wait. It runs on every (re)connect, so a
// broker restart drops readiness until a fresh SUBACK is observed.
func (s *MQTTSubscriber) handleConnect(client subscribeClient) {
	s.ready.Store(false)
	token := client.Subscribe(s.cfg.Topic, 1, s.onMessage)
	if !token.WaitTimeout(10 * time.Second) {
		log.Printf("bridge: timed out subscribing to %s", s.cfg.Topic)
		return
	}
	if err := token.Error(); err != nil {
		log.Printf("bridge: subscribe error for %s: %v", s.cfg.Topic, err)
		return
	}
	if sr, ok := token.(subscribeResult); ok {
		granted, exists := sr.Result()[s.cfg.Topic]
		if !exists || granted != 1 {
			log.Printf("bridge: broker did not grant QoS 1 for %s (got %v)", s.cfg.Topic, granted)
			// Do not mark ready: a lower/missing granted QoS means our
			// QoS 1 delivery assumption does not hold on this broker.
			return
		}
	}
	s.ready.Store(true)
	log.Printf("bridge: subscribed to %s", s.cfg.Topic)
}

func (s *MQTTSubscriber) onMessage(_ mqtt.Client, msg mqtt.Message) {
	s.pipeline.Enqueue(InboundMessage{
		Topic:    msg.Topic(),
		Payload:  msg.Payload(),
		Received: time.Now(),
	})
}
