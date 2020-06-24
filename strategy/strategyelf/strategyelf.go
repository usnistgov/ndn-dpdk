package strategyelf

//go:generate bash generate.sh

import (
	"fmt"
)

// Load a built-in strategy by short name.
func Load(shortname string) (elf []byte, e error) {
	return Asset(fmt.Sprintf("%s.o", shortname))
}
