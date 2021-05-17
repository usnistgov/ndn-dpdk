package tgconsumer

//go:generate go run ../../mk/enumgen/ -guard=NDNDPDK_TGCONSUMER_ENUM_H -out=../../csrc/tgconsumer/enum.h .

import (
	"errors"
	"fmt"

	"github.com/pkg/math"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

const (
	// MaxPatterns is maximum number of traffic patterns.
	MaxPatterns = 128

	// MaxSumWeight is maximum sum of weights among traffic patterns.
	MaxSumWeight = 8192

	_ = "enumgen::Tgc"
)

// Error conditions.
var (
	ErrNoPattern         = errors.New("no pattern specified")
	ErrTooManyPatterns   = fmt.Errorf("cannot add more than %d patterns", MaxPatterns)
	ErrFirstSeqNumOffset = errors.New("first pattern cannot have SeqNumOffset")
	ErrTooManyWeights    = fmt.Errorf("sum of weight cannot exceed %d", MaxSumWeight)
)

// Pattern configures how the consumer generates a sequence of Interests.
type Pattern struct {
	Weight int `json:"weight,omitempty"` // weight of random choice, minimum/default is 1

	ndni.InterestTemplateConfig

	// If non-zero, request cached Data. This must appear after a pattern without SeqNumOffset.
	// The client derives sequence number by subtracting SeqNumOffset from the previous pattern's
	// sequence number. Sufficient CS capacity is necessary for Data to actually come from CS.
	SeqNumOffset int `json:"seqNumOffset,omitempty"`
}

func (pattern *Pattern) applyDefaults() {
	pattern.Weight = math.MaxInt(1, pattern.Weight)
}
