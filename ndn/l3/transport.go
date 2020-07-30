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
