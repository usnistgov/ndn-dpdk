package appinit

import (
	"flag"
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

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
	return fmt.Sprintf("%v", v.Value)
}

// Declare 'initcfg' flag.
// The flag can either be a YAML document, or "@" followed by a filename.
// value should be pointer to a struct containing config sections.
func DeclareInitConfigFlag(flags *flag.FlagSet, value interface{}) {
	flags.Var(&yamlFlagValue{value}, "initcfg", "initialization config object")
}

// Config sections defined by appinit package.
// To add more sections, embed with `yaml:",inline"` tag.
type InitConfig struct {
	Mempool           MempoolsCapacityConfig
	FaceQueueCapacity FaceQueueCapacityConfig
}
