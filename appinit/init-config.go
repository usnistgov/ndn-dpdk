package appinit

import (
	"flag"
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface/createface"
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

// Declare YAML config flag.
// The flag can either be a YAML document, or "@" followed by a filename.
// value should be pointer to a struct containing config sections.
func DeclareConfigFlag(flags *flag.FlagSet, value interface{}, name, usage string) {
	flags.Var(&yamlFlagValue{value}, name, usage)
}

// Declare 'initcfg' flag.
func DeclareInitConfigFlag(flags *flag.FlagSet, value interface{}) {
	DeclareConfigFlag(flags, value, "initcfg", "initialization config object")
}

// Config sections defined by appinit package.
// To add more sections, embed with `yaml:",inline"` tag.
type InitConfig struct {
	Mempool    MempoolsCapacityConfig
	LCoreAlloc dpdk.LCoreAllocConfig
	Face       createface.Config
}

func (initCfg InitConfig) Apply() {
	initCfg.Mempool.Apply()

	dpdk.LCoreAlloc.Config = initCfg.LCoreAlloc

	if e := EnableCreateFace(initCfg.Face); e != nil {
		log.WithError(e).Fatal("face init error")
	}
}
