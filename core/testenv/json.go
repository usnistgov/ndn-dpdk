package testenv

import "encoding/json"

// FromJSON unmarshals from JSON string.
// Error causes panic.
func FromJSON(j string, ptr interface{}) {
	e := json.Unmarshal([]byte(j), ptr)
	if e != nil {
		panic(e)
	}
}

// ToJSON marshals a value as JSON string.
func ToJSON(v interface{}) string {
	j, e := json.Marshal(v)
	if e != nil {
		return "ERROR: " + e.Error()
	}
	return string(j)
}
