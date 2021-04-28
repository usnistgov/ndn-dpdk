package tgconsumer

//go:generate go run ../../mk/enumgen/ -guard=NDNDPDK_TGCONSUMER_ENUM_H -out=../../csrc/tgconsumer/enum.h .

import (
	"errors"
	"fmt"

	"github.com/pkg/math"
	"github.com/usnistgov/ndn-dpdk/core/nnduration"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

const (
	// MaxPatterns is maximum number of traffic patterns.
	MaxPatterns = 128

	// MaxSumWeight is maximum sum of weights among traffic patterns.
	MaxSumWeight = 8192

	// TokenPatternBits is the bitwidth of pattern ID in PIT token.
	TokenPatternBits = 8

	// TokenRunBits in the bitwidth of run number in PIT token.
	TokenRunBits = 8

	// TokenTimeBits is the number of timestamp in PIT token.
	TokenTimeBits = 48

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

	Prefix           ndn.Name                `json:"prefix"`
	CanBePrefix      bool                    `json:"canBePrefix,omitempty"`
	MustBeFresh      bool                    `json:"mustBeFresh,omitempty"`
	InterestLifetime nnduration.Milliseconds `json:"interestLifetime,omitempty"`
	HopLimit         ndn.HopLimit            `json:"hopLimit,omitempty"`

	// If non-zero, request cached Data. This must appear after a pattern without SeqNumOffset.
	// The client derives sequence number by subtracting SeqNumOffset from the previous pattern's
	// sequence number. Sufficient CS capacity is necessary for Data to actually come from CS.
	SeqNumOffset int `json:"seqNumOffset,omitempty"`
}

func (pattern *Pattern) applyDefaults() {
	pattern.Weight = math.MaxInt(1, pattern.Weight)
}

func (pattern *Pattern) initInterestTemplate(tpl *ndni.InterestTemplate) {
	a := []interface{}{pattern.Prefix}
	if pattern.CanBePrefix {
		a = append(a, ndn.CanBePrefixFlag)
	}
	if pattern.MustBeFresh {
		a = append(a, ndn.MustBeFreshFlag)
	}
	if lifetime := pattern.InterestLifetime.Duration(); lifetime != 0 {
		a = append(a, lifetime)
	}
	if pattern.HopLimit != 0 {
		a = append(a, pattern.HopLimit)
	}
	tpl.Init(a...)
}
