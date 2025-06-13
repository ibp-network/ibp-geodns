package api

import (
	"encoding/json"
	"io"
)

func DecodeJSONBody(r io.Reader, target interface{}) error {
	decoder := json.NewDecoder(r)
	return decoder.Decode(target)
}
