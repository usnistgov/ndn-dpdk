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

type commaSeparatedNumbers []string

func (csn *commaSeparatedNumbers) Append(n int) {
	*csn = append(*csn, strconv.Itoa(n))
}

func (csn commaSeparatedNumbers) String() string {
	return strings.Join([]string(csn), ",")
}
