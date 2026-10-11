package schema

import (
	"strings"
	"testing"
)

func validEnvelopeJSON(data string) string {
	return `{
		"specversion": "1.0.2",
		"id": "b3b5b9d0-7c1a-4f4e-9a2b-0f1a2b3c4d5e",
		"source": "//grid/v1/isv/gridmetry",
		"type": "grid.loadtarget.set.v1",
		"time": "2026-10-11T00:00:00Z",
		"datacontenttype": "application/json",
		"subject": "loadtarget",
		"traceparent": "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
		"data": ` + data + `
	}`
}

func TestEnvelopeSchemaAcceptsSignedString(t *testing.T) {
	instance, err := ToInstance([]byte(validEnvelopeJSON(`"eyJhbGciOiJFUzI1NiJ9.eyJmb28iOiJiYXIifQ.c2ln"`)))
	if err != nil {
		t.Fatalf("ToInstance: %v", err)
	}
	if err := EnvelopeSchema.Validate(instance); err != nil {
		t.Fatalf("expected valid envelope, got: %v", err)
	}
}

func TestEnvelopeSchemaRejectsMissingRequiredField(t *testing.T) {
	missing := `{
		"specversion": "1.0.2",
		"id": "b3b5b9d0-7c1a-4f4e-9a2b-0f1a2b3c4d5e",
		"source": "//grid/v1/isv/gridmetry",
		"type": "grid.loadtarget.set.v1",
		"datacontenttype": "application/json",
		"subject": "loadtarget",
		"traceparent": "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
		"data": "eyJhbGciOiJFUzI1NiJ9.eyJmb28iOiJiYXIifQ.c2ln"
	}`
	instance, err := ToInstance([]byte(missing))
	if err != nil {
		t.Fatalf("ToInstance: %v", err)
	}
	if err := EnvelopeSchema.Validate(instance); err == nil {
		t.Fatal("expected validation error for missing time field")
	}
}

func TestEnvelopeSchemaRejectsWrongSpecVersion(t *testing.T) {
	bad := strings.Replace(validEnvelopeJSON(`"eyJhbGciOiJFUzI1NiJ9.eyJmb28iOiJiYXIifQ.c2ln"`), `"1.0.2"`, `"1.0"`, 1)
	instance, err := ToInstance([]byte(bad))
	if err != nil {
		t.Fatalf("ToInstance: %v", err)
	}
	if err := EnvelopeSchema.Validate(instance); err == nil {
		t.Fatal("expected validation error for wrong specversion")
	}
}

func TestPayloadSchemaRequiresNonEmptyTargets(t *testing.T) {
	instance, err := ToInstance([]byte(`{"targets": []}`))
	if err != nil {
		t.Fatalf("ToInstance: %v", err)
	}
	if err := PayloadSchema.Validate(instance); err == nil {
		t.Fatal("expected validation error for empty targets")
	}
}

func TestPayloadSchemaAcceptsNullReleaseConstraint(t *testing.T) {
	instance, err := ToInstance([]byte(`{"targets":[{"interval":{},"load_constraint":null}]}`))
	if err != nil {
		t.Fatalf("ToInstance: %v", err)
	}
	if err := PayloadSchema.Validate(instance); err != nil {
		t.Fatalf("expected valid payload with null release, got: %v", err)
	}
}

func TestPayloadSchemaRejectsMissingLoadConstraint(t *testing.T) {
	instance, err := ToInstance([]byte(`{"targets":[{"interval":{}}]}`))
	if err != nil {
		t.Fatalf("ToInstance: %v", err)
	}
	if err := PayloadSchema.Validate(instance); err == nil {
		t.Fatal("expected validation error for missing load_constraint")
	}
}
