package bridge

import (
	"encoding/json"

	"gridmetry/dsx-adapter/internal/flex"
)

func marshalEvent(event *flex.NormalizedEvent) ([]byte, error) {
	return json.Marshal(event)
}
