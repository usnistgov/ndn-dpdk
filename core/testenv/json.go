package testenv

import "encoding/json"

// FromJSON unmarshals from JSON string.
// Error causes panic.
func FromJSON[T any](j string) (value T) {
	e := json.Unmarshal([]byte(j), &value)
	if e != nil {
		panic(e)
	}
	return
}

// ToJSON marshals a value as JSON string.
func ToJSON(v any) string {
	j, e := json.Marshal(v)
	if e != nil {
		return "ERROR: " + e.Error()
	}
	return string(j)
}
