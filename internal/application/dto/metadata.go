package dto

import "encoding/json"

type JSONMetadata json.RawMessage

func (m JSONMetadata) MarshalJSON() ([]byte, error) {
	if len(m) == 0 || string(m) == "null" {
		return []byte("{}"), nil
	}
	return m, nil
}
