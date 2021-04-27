package l3

// Transport represents a communicate channel to send and receive TLV packets.
type Transport interface {
	// Rx returns a channel to receive incoming TLV elements.
	// This function always returns the same channel.
	// This channel is closed when the transport is closed.
	Rx() <-chan []byte

	// Tx returns a channel to send outgoing TLV elements.
	// This function always returns the same channel.
	// Closing this channel causes the transport to close.
	Tx() chan<- []byte

	// State returns current state.
	State() TransportState

	// OnStateChange registers a callback to be invoked when State() changes.
	// Returns a function to cancel the callback registration.
	OnStateChange(cb func(st TransportState)) (cancel func())
}

// TransportQueueConfig defaults.
const (
	DefaultTransportRxQueueSize = 64
	DefaultTransportTxQueueSize = 64
)

// TransportQueueConfig contains Transport queue configuration.
type TransportQueueConfig struct {
	// RxQueueSize is the Go channel buffer size of RX channel.
	// The default is DefaultTransportRxQueueSize.
	RxQueueSize int `json:"rxQueueSize,omitempty"`

	// TxQueueSize is the Go channel buffer size of TX channel.
	// The default is DefaultTransportTxQueueSize.
	TxQueueSize int `json:"txQueueSize,omitempty"`
}

// ApplyTransportQueueConfigDefaults sets empty values to defaults.
func (cfg *TransportQueueConfig) ApplyTransportQueueConfigDefaults() {
	if cfg.RxQueueSize <= 0 {
		cfg.RxQueueSize = DefaultTransportRxQueueSize
	}
	if cfg.TxQueueSize <= 0 {
		cfg.TxQueueSize = DefaultTransportTxQueueSize
	}
}

// TransportState indicates up/down state of a transport.
type TransportState int

const (
	// TransportUp indicates the transport is operational.
	TransportUp TransportState = iota

	// TransportDown indicates the transport is nonoperational.
	TransportDown

	// TransportClosed indicates the transport has been closed.
	// It cannot be restarted.
	TransportClosed
)
