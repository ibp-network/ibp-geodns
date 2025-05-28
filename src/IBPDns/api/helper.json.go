package api

import (
	"encoding/json"
	"io"
)

// DecodeJSONBody decodes JSON from an io.Reader into a target interface
func DecodeJSONBody(r io.Reader, target interface{}) error {
	decoder := json.NewDecoder(r)
	return decoder.Decode(target)
}
