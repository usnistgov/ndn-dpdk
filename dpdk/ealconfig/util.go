package ealconfig

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/kballard/go-shellquote"
)

func shellSplit(field, flags string) (args []string, e error) {
	args, e = shellquote.Split(flags)
	if e != nil {
		return nil, fmt.Errorf("%s: %w", field, e)
	}
	return args, nil
}

type commaSeparated []string

func (cs *commaSeparated) AppendInt(n ...int) {
	for _, v := range n {
		*cs = append(*cs, strconv.Itoa(v))
	}
}

func (cs *commaSeparated) AppendString(s ...string) {
	*cs = append(*cs, s...)
}

func (cs commaSeparated) String() string {
	return strings.Join([]string(cs), ",")
}
