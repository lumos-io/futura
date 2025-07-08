package metadata

import (
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// AttributeDirection specifies the value direction attribute.
type AttributeDirection int

const (
	_ AttributeDirection = iota
	AttributeDirectionReceive
	AttributeDirectionTransmit
)

// String returns the string representation of the AttributeDirection.
func (av AttributeDirection) String() string {
	switch av {
	case AttributeDirectionReceive:
		return "receive"
	case AttributeDirectionTransmit:
		return "transmit"
	}
	return ""
}

// AggregationTemporality defines how a metric aggregator reports aggregated values.
// It describes how those values relate to the time interval over which they are aggregated.
type AggregationTemporality int32

const (
	// AggregationTemporalityUnspecified is the default AggregationTemporality, it MUST NOT be used.
	AggregationTemporalityUnspecified = AggregationTemporality(1)
	// AggregationTemporalityDelta is a AggregationTemporality for a metric aggregator which reports changes since last report time.
	AggregationTemporalityDelta = AggregationTemporality(2)
	// AggregationTemporalityCumulative is a AggregationTemporality for a metric aggregator which reports changes since a fixed start time.
	AggregationTemporalityCumulative = AggregationTemporality(3)
)

// String returns the string representation of the AggregationTemporality.
func (at AggregationTemporality) String() string {
	switch at {
	case AggregationTemporalityUnspecified:
		return "Unspecified"
	case AggregationTemporalityDelta:
		return "Delta"
	case AggregationTemporalityCumulative:
		return "Cumulative"
	}
	return ""
}

func toProtoTime(t time.Time) *timestamppb.Timestamp {
	return timestamppb.New(t)
}
