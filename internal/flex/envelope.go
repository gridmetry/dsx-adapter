package flex

import (
	"errors"
	"regexp"
	"time"
)

// SpecVersion is the DSX-pinned CloudEvents specversion. The standard
// CloudEvents SDK spec version is "1.0"; DSX requires "1.0.2" and this
// profile does not claim generic CloudEvents SDK conformance.
const SpecVersion = "1.0.2"

// LoadTargetSetType is the only event type this lab profile accepts.
const LoadTargetSetType = "grid.loadtarget.set.v1"

// MaxRawEnvelopeBytes bounds an entire inbound wire message.
const MaxRawEnvelopeBytes = 64 * 1024

// MaxDecodedPayloadBytes bounds the JWS-verified plaintext payload.
const MaxDecodedPayloadBytes = 8 * 1024

var (
	sourcePattern      = regexp.MustCompile(`^//grid/v1/(isv|dsx-flex-agent|infra)/[^/]+$`)
	traceparentPattern = regexp.MustCompile(`^[0-9a-f]{2}-[0-9a-f]{32}-[0-9a-f]{16}-[0-9a-f]{2}$`)
	uuidV4Pattern      = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
)

// CloudEvent is the pinned DSX wire envelope for grid.loadtarget.set.v1 in
// MQTT 3.1.1 structured content mode. Data is always the JWS compact
// serialization string; this lab profile rejects plaintext data objects, so
// the field is typed as string rather than the schema's permissive oneOf.
type CloudEvent struct {
	SpecVersion     string `json:"specversion"`
	ID              string `json:"id"`
	Source          string `json:"source"`
	Type            string `json:"type"`
	Time            string `json:"time"`
	DataContentType string `json:"datacontenttype"`
	Subject         string `json:"subject"`
	Traceparent     string `json:"traceparent"`
	Tracestate      string `json:"tracestate,omitempty"`
	CorrelationID   string `json:"correlationid,omitempty"`
	Data            string `json:"data"`
}

// ValidateEnvelope applies manual checks beyond what the JSON Schema
// enforces: strict UUIDv4 (the schema only asserts format "uuid", any
// version), a nonzero W3C traceparent trace-id/parent-id, and the expected
// subject for this message type.
func ValidateEnvelope(ev *CloudEvent) error {
	if ev.SpecVersion != SpecVersion {
		return errors.New("flex: unsupported specversion")
	}
	if !uuidV4Pattern.MatchString(ev.ID) {
		return errors.New("flex: id must be a UUIDv4")
	}
	if !sourcePattern.MatchString(ev.Source) {
		return errors.New("flex: source does not match //grid/v1/<role>/<identifier>")
	}
	if ev.Type != LoadTargetSetType {
		return errors.New("flex: unsupported event type")
	}
	if _, err := time.Parse(time.RFC3339, ev.Time); err != nil {
		return errors.New("flex: time must be RFC3339")
	}
	if ev.DataContentType != "application/json" {
		return errors.New("flex: datacontenttype must be application/json")
	}
	if ev.Subject != "loadtarget" {
		return errors.New("flex: unsupported subject for loadtarget command")
	}
	if !validTraceparent(ev.Traceparent) {
		return errors.New("flex: invalid or zero traceparent")
	}
	if ev.CorrelationID != "" && !uuidV4Pattern.MatchString(ev.CorrelationID) {
		return errors.New("flex: correlationid must be a UUIDv4")
	}
	if ev.Data == "" {
		return errors.New("flex: data must be a signed JWS compact string")
	}
	return nil
}

// validTraceparent enforces the W3C requirement that trace-id and parent-id
// are not all-zero, beyond the schema's hex-pattern-only assertion.
func validTraceparent(tp string) bool {
	if !traceparentPattern.MatchString(tp) {
		return false
	}
	traceID := tp[3:35]
	parentID := tp[36:52]
	return traceID != "00000000000000000000000000000000" && parentID != "0000000000000000"
}
