package flex

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// DecodeStrict unmarshals data into v after rejecting duplicate JSON object
// field names at any nesting level. encoding/json's default decoder silently
// keeps the last value for a repeated key, which this contract's data field
// integrity (the outer envelope is unsigned) makes an ambiguity risk worth
// bounding explicitly, without rejecting legitimate unknown fields.
func DecodeStrict(data []byte, v any) error {
	if err := checkNoDuplicateKeys(data); err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func checkNoDuplicateKeys(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	if _, err := decodeCheckingDuplicates(dec); err != nil {
		return err
	}
	return nil
}

// decodeCheckingDuplicates walks one JSON value using the streaming
// tokenizer, returning an error if any object within it repeats a field
// name. It does not build or return a usable value; callers still decode
// with encoding/json.Unmarshal afterward.
func decodeCheckingDuplicates(dec *json.Decoder) (any, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	delim, ok := tok.(json.Delim)
	if !ok {
		return tok, nil
	}
	switch delim {
	case '{':
		seen := make(map[string]bool)
		for dec.More() {
			keyTok, err := dec.Token()
			if err != nil {
				return nil, err
			}
			key, ok := keyTok.(string)
			if !ok {
				return nil, fmt.Errorf("invalid object key token %v", keyTok)
			}
			if seen[key] {
				return nil, fmt.Errorf("duplicate JSON field %q", key)
			}
			seen[key] = true
			if _, err := decodeCheckingDuplicates(dec); err != nil {
				return nil, err
			}
		}
		if _, err := dec.Token(); err != nil && err != io.EOF {
			return nil, err
		}
		return nil, nil
	case '[':
		for dec.More() {
			if _, err := decodeCheckingDuplicates(dec); err != nil {
				return nil, err
			}
		}
		if _, err := dec.Token(); err != nil && err != io.EOF {
			return nil, err
		}
		return nil, nil
	default:
		return tok, nil
	}
}
