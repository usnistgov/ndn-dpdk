package pdump

//go:generate go run ../../mk/enumgen/ -guard=NDNDPDK_PDUMP_ENUM_H -out=../../csrc/pdump/enum.h .

const (
	// MaxNames is the maximum number of name filters.
	MaxNames = 4

	// WriterBurstSize is the burst size in the writer.
	WriterBurstSize = 64

	// NgTypeSHB is PCAPNG section header block type.
	NgTypeSHB = 0x0A0D0D0A

	// NgTypeIDB is PCAPNG interface description block type.
	NgTypeIDB = 0x00000001

	// NgTypeIDB is PCAPNG enhanced packet block type.
	NgTypeEPB = 0x00000006

	_ = "enumgen::Pdump"
)

// Limits and defaults.
const (
	MinFileSize     = 1 << 16
	DefaultFileSize = 1 << 24
)
