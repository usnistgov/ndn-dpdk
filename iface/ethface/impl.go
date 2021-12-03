package ethface

// RxImplKind identifies port RX implementation.
type RxImplKind string

// RxImplKind values.
const (
	RxImplMemif RxImplKind = "RxMemif"
	RxImplFlow  RxImplKind = "RxFlow"
	RxImplTable RxImplKind = "RxTable"
)

// RX/TX setup implementation.
type rxImpl interface {
	Kind() RxImplKind
	Init(port *Port) error
	Start(face *ethFace) error
	Stop(face *ethFace) error
	Close(port *Port) error
}
