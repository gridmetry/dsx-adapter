// Package bridge wires an MQTT 3.1.1 QoS 1 subscription to the flex codec
// and the transport hub. Paho connection/subscription glue is exercised by
// the Compose-backed integration path; the bounded ingress worker queue and
// processing pipeline here are unit-testable without a broker.
package bridge

import (
	"log"
	"time"

	"gridmetry/dsx-adapter/internal/flex"
	"gridmetry/dsx-adapter/internal/lab"
)

// InboundMessage is one raw MQTT PUBLISH payload queued for processing.
type InboundMessage struct {
	Topic    string
	Payload  []byte
	Received time.Time
}

// Publisher is the subset of transport.Hub the pipeline needs.
type Publisher interface {
	Publish(data []byte)
}

// Pipeline decodes/validates inbound MQTT payloads off a bounded queue and
// publishes accepted normalized events. It never blocks the MQTT client
// callback: Enqueue is non-blocking and drops (counting, not panicking)
// when the queue is full, which is reported honestly rather than claimed
// as durable delivery.
type Pipeline struct {
	Codec *flex.Codec
	Hub   Publisher
	Stats *lab.Stats
	queue chan InboundMessage
	nowFn func() time.Time
}

// NewPipeline creates a pipeline with a bounded inbound queue of the given
// capacity and a single worker goroutine. Start must be called to begin
// processing.
func NewPipeline(codec *flex.Codec, hub Publisher, stats *lab.Stats, queueCapacity int) *Pipeline {
	if queueCapacity < 1 {
		queueCapacity = 1
	}
	return &Pipeline{
		Codec: codec,
		Hub:   hub,
		Stats: stats,
		queue: make(chan InboundMessage, queueCapacity),
		nowFn: time.Now,
	}
}

// Enqueue is called from the MQTT message callback. It never blocks: if the
// bounded queue is full, the message is dropped and counted rather than
// stalling the MQTT client's receive loop.
func (p *Pipeline) Enqueue(msg InboundMessage) (queued bool) {
	select {
	case p.queue <- msg:
		return true
	default:
		if p.Stats != nil {
			p.Stats.IncQueueDropped()
		}
		log.Printf("bridge: inbound queue full, dropping message from topic %s", msg.Topic)
		return false
	}
}

// Run processes queued messages until ctx/stop is signaled via Close
// (closing done). It is intended to run in its own goroutine.
func (p *Pipeline) Run(done <-chan struct{}) {
	for {
		select {
		case <-done:
			return
		case msg, ok := <-p.queue:
			if !ok {
				return
			}
			p.process(msg)
		}
	}
}

func (p *Pipeline) process(msg InboundMessage) {
	now := p.nowFn()
	event, err := p.Codec.Decode(msg.Payload, now)
	if err != nil {
		if err == flex.ErrDuplicate {
			if p.Stats != nil {
				p.Stats.IncDuplicate()
			}
			return
		}
		if p.Stats != nil {
			p.Stats.IncRejected()
		}
		log.Printf("bridge: rejected command from topic %s: %v", msg.Topic, err)
		return
	}
	if p.Stats != nil {
		p.Stats.IncAccepted()
	}
	// Re-check end_time with a fresh clock read immediately before hub
	// publish: the bounded queue may have delayed processing past
	// end_time even though the command was fresh at ingress/decode time.
	// A nil EndTime never expires (an unbounded release/constraint).
	if event.EndTime != nil && !event.EndTime.After(p.nowFn()) {
		if p.Stats != nil {
			p.Stats.IncExpired()
		}
		log.Printf("bridge: dropping accepted command %s from topic %s: end_time %s already passed before hub publish", event.ID, msg.Topic, event.EndTime.Format(time.RFC3339))
		return
	}
	data, err := marshalEvent(event)
	if err != nil {
		log.Printf("bridge: failed to marshal normalized event: %v", err)
		return
	}
	p.Hub.Publish(data)
}
