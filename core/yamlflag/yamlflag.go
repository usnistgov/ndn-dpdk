// Package yamlflag provides a command line flag that accepts a YAML document.
package yamlflag

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"reflect"

	"github.com/ghodss/yaml"
)

// New creates a flag.Value that recognizes a YAML document.
//
// The YAML document can be specified directly on the command line:
//   --flag="Key: value"
// Or it can be read from a file, when the flag value starts with '@':
//   --flag=@file.yaml
//
// value must be a pointer to a struct containing config sections.
// Panics if value is not a pointer.
func New(value interface{}) flag.Getter {
	if val := reflect.ValueOf(value); val.Kind() != reflect.Ptr {
		panic(val.Kind())
	}
	return &yamlFlagValue{value}
}

type yamlFlagValue struct {
	Value interface{}
}

func (v *yamlFlagValue) Get() interface{} {
	return v.Value
}

func (v *yamlFlagValue) Set(s string) error {
	if len(s) >= 1 && s[0] == '@' {
		file, e := ioutil.ReadFile(s[1:])
		if e != nil {
			return e
		}
		return yaml.Unmarshal(file, v.Value)
	}
	return yaml.Unmarshal([]byte(s), v.Value)
}

func (v *yamlFlagValue) String() string {
	j, _ := json.Marshal(v.Value)
	return string(j)
}
