package appinit

import (
	"flag"
	"fmt"
	"io/ioutil"

	"github.com/ghodss/yaml"

	"ndn-dpdk/dpdk/eal"
	"ndn-dpdk/dpdk/pktmbuf"
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
// Embed this struct to add more sections.
type InitConfig struct {
	Mempool    pktmbuf.TemplateUpdates
	LCoreAlloc eal.LCoreAllocConfig
	Face       createface.Config
}

func (cfg InitConfig) Apply() {
	cfg.Mempool.Apply()
	eal.LCoreAlloc.Config = cfg.LCoreAlloc

	if e := cfg.Face.Apply(); e != nil {
		log.WithError(e).Warn("face init-config not applied")
	}
}
