package tgproducer

//go:generate go run ../../mk/enumgen/ -guard=NDNDPDK_TGPRODUCER_ENUM_H -out=../../csrc/tgproducer/enum.h .

import (
	"errors"
	"fmt"

	"github.com/pkg/math"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

const (
	// MaxPatterns is maximum number of traffic patterns.
	MaxPatterns = 128

	// MaxReplies is maximum number of replies per pattern.
	MaxReplies = 8

	// MaxSumWeight is maximum sum of weights among replies.
	MaxSumWeight = 256

	_ = "enumgen::Tgp"
)

// Error conditions.
var (
	ErrNoPattern       = errors.New("no pattern specified")
	ErrTooManyPatterns = fmt.Errorf("cannot add more than %d patterns", MaxPatterns)
	ErrPrefixTooLong   = fmt.Errorf("prefix cannot exceed %d octets", ndni.NameMaxLength)
	ErrTooManyReplies  = fmt.Errorf("cannot add more than %d replies", MaxReplies)
	ErrTooManyWeights  = fmt.Errorf("sum of weight cannot exceed %d", MaxSumWeight)
)

// Config describes producer configuration.
type Config struct {
	NThreads int                  `json:"nThreads,omitempty"` // number of threads, minimum/default is 1
	RxQueue  iface.PktQueueConfig `json:"rxQueue,omitempty"`
	Patterns []Pattern            `json:"patterns"`

	nDataGen int
}

func (cfg *Config) validateWithDefaults() error {
	cfg.NThreads = math.MaxInt(1, cfg.NThreads)
	cfg.RxQueue.DisableCoDel = true

	if len(cfg.Patterns) == 0 {
		return ErrNoPattern
	}
	if len(cfg.Patterns) > MaxPatterns {
		return ErrTooManyPatterns
	}
	patterns := []Pattern{}
	nDataGen := 0
	for _, pattern := range cfg.Patterns {
		sumWeight, nData := pattern.applyDefaults()
		if sumWeight > MaxSumWeight {
			return ErrTooManyWeights
		}
		nDataGen += nData
		if len(pattern.prefixV) > ndni.NameMaxLength {
			return ErrPrefixTooLong
		}
		patterns = append(patterns, pattern)
	}

	cfg.Patterns, cfg.nDataGen = patterns, nDataGen
	return nil
}

// Pattern configures how the producer replies to Interests under a name prefix.
type Pattern struct {
	Prefix  ndn.Name `json:"prefix"`
	Replies []Reply  `json:"replies"` // if empty, reply with Data FreshnessPeriod=1

	prefixV []byte
}

func (pattern *Pattern) applyDefaults() (sumWeight, nDataGen int) {
	pattern.prefixV, _ = pattern.Prefix.MarshalBinary()
	if len(pattern.Replies) == 0 {
		pattern.Replies = []Reply{
			{
				DataGenConfig: ndni.DataGenConfig{
					FreshnessPeriod: 1,
				},
			},
		}
	}

	for i := range pattern.Replies {
		reply := &pattern.Replies[i]
		reply.Weight = math.MaxInt(1, reply.Weight)
		sumWeight += reply.Weight
		if reply.Kind() == ReplyData {
			nDataGen++
		}
	}
	return
}

// ReplyKind indicates reply packet type.
type ReplyKind int

// ReplyKind values.
const (
	ReplyData ReplyKind = iota
	ReplyNack
	ReplyTimeout

	_ = "enumgen:TgpReplyKind:TgpReply:Reply"
)

// Reply configures how the producer replies to the Interest.
type Reply struct {
	Weight int `json:"weight,omitempty"` // weight of random choice, minimum/default is 1

	ndni.DataGenConfig
	Nack    uint8 `json:"nack,omitempty"`    // if not NackNone, reply with Nack instead of Data
	Timeout bool  `json:"timeout,omitempty"` // if true, drop the Interest instead of sending Data
}

// Kind returns ReplyKind.
func (reply Reply) Kind() ReplyKind {
	switch {
	case reply.Timeout:
		return ReplyTimeout
	case reply.Nack != an.NackNone:
		return ReplyNack
	default:
		return ReplyData
	}
}
