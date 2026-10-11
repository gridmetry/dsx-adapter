// Package schema compiles JSON Schemas from a build-contained, pinned copy of
// the DSX Flex AsyncAPI contract. It never fetches schemas at runtime and
// never depends on the sibling dsx-exchange checkout being present.
//
// Source: ../dsx-exchange/schemas/asyncapi/dsx-flex/dsx-flex.yaml at commit
// 4e8b71f0584913a5f88fa728dc9fe724337aa97b (original file copied verbatim
// into dsx-flex.yaml alongside this file, including its original
// Copyright/SPDX header, which is not modified or supplemented here).
package schema

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"gopkg.in/yaml.v3"
)

//go:embed dsx-flex.yaml
var contractYAML []byte

const resourceURL = "dsx-flex-contract.yaml"

// EnvelopeSchema validates a raw CloudEvents envelope against
// components/schemas/LoadTargetSetEvent.
var EnvelopeSchema *jsonschema.Schema

// PayloadSchema validates a decoded (post-JWS-verification) load-target
// payload against components/schemas/SetLoadTargetData.
var PayloadSchema *jsonschema.Schema

func init() {
	doc, err := yamlToJSONInstance(contractYAML)
	if err != nil {
		panic(fmt.Sprintf("schema: decoding embedded contract: %v", err))
	}
	c := jsonschema.NewCompiler()
	c.AssertFormat()
	if err := c.AddResource(resourceURL, doc); err != nil {
		panic(fmt.Sprintf("schema: adding embedded contract resource: %v", err))
	}
	EnvelopeSchema, err = c.Compile(resourceURL + "#/components/schemas/LoadTargetSetEvent")
	if err != nil {
		panic(fmt.Sprintf("schema: compiling LoadTargetSetEvent: %v", err))
	}
	PayloadSchema, err = c.Compile(resourceURL + "#/components/schemas/SetLoadTargetData")
	if err != nil {
		panic(fmt.Sprintf("schema: compiling SetLoadTargetData: %v", err))
	}
}

// yamlToJSONInstance decodes YAML into Go-native values, then re-encodes and
// re-decodes through encoding/json (with json.Number) so the resulting tree
// matches the canonical representation jsonschema/v6 expects from
// jsonschema.UnmarshalJSON, regardless of how yaml.v3 represented numbers.
func yamlToJSONInstance(raw []byte) (any, error) {
	var v any
	if err := yaml.Unmarshal(raw, &v); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("normalizing to JSON: %w", err)
	}
	return jsonschema.UnmarshalJSON(bytes.NewReader(jsonBytes))
}

// ToInstance converts raw JSON bytes into the canonical instance
// representation expected by jsonschema/v6's Schema.Validate.
func ToInstance(jsonBytes []byte) (any, error) {
	return jsonschema.UnmarshalJSON(bytes.NewReader(jsonBytes))
}
