package tgconsumer

//go:generate go run ../../mk/enumgen/ -guard=NDNDPDK_TGCONSUMER_ENUM_H -out=../../csrc/tgconsumer/enum.h .

import (
	"errors"
	"fmt"
	"time"

	"github.com/usnistgov/ndn-dpdk/core/nnduration"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"github.com/zyedidia/generic"
)

const (
	// MaxPatterns is maximum number of traffic patterns.
	MaxPatterns = 128

	// MaxSumWeight is maximum sum of weights among traffic patterns.
	MaxSumWeight = 8192

	// DigestLowWatermark is the number of remaining Data packets in the crypto device before enqueuing a new burst.
	DigestLowWatermark = 16

	// DigestBurstSize is the number of Data packets to enqueue into crypto device.
	DigestBurstSize = 64

	_ = "enumgen::Tgc"
)

const defaultInterval = 1 * time.Millisecond

// Config describes consumer configuration.
type Config struct {
	RxQueue iface.PktQueueConfig `json:"rxQueue,omitempty"`

	// Interval defines average Interest interval.
	// TX thread transmits Interests in bursts, so the specified interval will be converted to
	// a burst interval with equivalent traffic amount.
	// Default is 1ms.
	Interval nnduration.Nanoseconds `json:"interval"`

	// Patterns defines traffic patterns.
	// It must contain between 1 and MaxPatterns entries.
	Patterns []Pattern `json:"patterns"`

	nWeights, nDigestPatterns int
}

// Validate applies defaults and validates the configuration.
func (cfg *Config) Validate() error {
	cfg.RxQueue.DisableCoDel = true

	if len(cfg.Patterns) == 0 {
		return errors.New("no pattern specified")
	}
	if len(cfg.Patterns) > MaxPatterns {
		return fmt.Errorf("cannot add more than %d patterns", MaxPatterns)
	}

	patterns := []Pattern{}
	nWeights, nDigestPatterns := 0, 0
	for i, pattern := range cfg.Patterns {
		pattern.applyDefaults()
		patterns = append(patterns, pattern)
		nWeights += pattern.Weight
		if pattern.Digest != nil {
			nDigestPatterns++
		}
		if pattern.SeqNumOffset != 0 {
			if pattern.Digest != nil {
				return errors.New("pattern cannot have both Digest and SeqNumOffset")
			}
			if i == 0 {
				return errors.New("first pattern cannot have SeqNumOffset")
			}
		}
	}
	if nWeights > MaxSumWeight {
		return fmt.Errorf("sum of weight cannot exceed %d", MaxSumWeight)
	}
	cfg.Patterns, cfg.nWeights, cfg.nDigestPatterns = patterns, nWeights, nDigestPatterns
	return nil
}

// Pattern configures how the consumer generates a sequence of Interests.
type Pattern struct {
	// Weight of random choice, minimum/default is 1.
	Weight int `json:"weight,omitempty"`

	ndni.InterestTemplateConfig

	// If specified, append implicit digest to Interest name.
	// For Data to satisfy Interests, the producer pattern must reply with the same DataGenConfig.
	Digest *ndni.DataGenConfig `json:"digest,omitempty"`

	// If non-zero, request cached Data. This must appear after a pattern without SeqNumOffset.
	// The consumer derives sequence number by subtracting SeqNumOffset from the previous pattern's
	// sequence number. Sufficient CS capacity is necessary for Data to actually come from CS.
	SeqNumOffset int `json:"seqNumOffset,omitempty"`
}

func (pattern *Pattern) applyDefaults() {
	pattern.Weight = generic.Max(1, pattern.Weight)
}
