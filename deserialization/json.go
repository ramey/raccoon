package deserialization

import "encoding/json"

// type JSONDeserializer struct{}

func DeserializeJSON(b []byte, i interface{}) error {
	return json.Unmarshal(b, i)
}
