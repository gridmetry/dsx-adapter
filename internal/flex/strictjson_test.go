package flex

import "testing"

type simple struct {
	A string `json:"a"`
	B int    `json:"b"`
}

func TestDecodeStrictAcceptsUnknownFields(t *testing.T) {
	var s simple
	if err := DecodeStrict([]byte(`{"a":"x","b":1,"future_field":"ignored"}`), &s); err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if s.A != "x" || s.B != 1 {
		t.Fatalf("unexpected decode result: %+v", s)
	}
}

func TestDecodeStrictRejectsTopLevelDuplicateKey(t *testing.T) {
	var s simple
	err := DecodeStrict([]byte(`{"a":"x","a":"y","b":1}`), &s)
	if err == nil {
		t.Fatal("expected duplicate key rejection")
	}
}

func TestDecodeStrictRejectsNestedDuplicateKey(t *testing.T) {
	var out map[string]any
	err := DecodeStrict([]byte(`{"outer":{"x":1,"x":2}}`), &out)
	if err == nil {
		t.Fatal("expected nested duplicate key rejection")
	}
}

func TestDecodeStrictRejectsDuplicateKeyInArrayElement(t *testing.T) {
	var out map[string]any
	err := DecodeStrict([]byte(`{"targets":[{"a":1,"a":2}]}`), &out)
	if err == nil {
		t.Fatal("expected duplicate key rejection inside array element")
	}
}
