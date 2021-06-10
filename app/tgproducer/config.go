package tgproducer

//go:generate go run ../../mk/enumgen/ -guard=NDNDPDK_TGPRODUCER_ENUM_H -out=../../csrc/tgproducer/enum.h .

import (
	"errors"
	"fmt"

	"github.com/pkg/math"
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

// Pattern configures how the producer replies to Interests under a name prefix.
type Pattern struct {
	Prefix  ndn.Name `json:"prefix"`
	Replies []Reply  `json:"replies"` // if empty, reply with Data

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
