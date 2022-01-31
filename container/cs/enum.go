package cs

//go:generate go run ../../mk/enumgen/ -guard=NDNDPDK_CS_ENUM_H -out=../../csrc/pcct/cs-enum.h .

// Defaults and limits.
const (
	MaxIndirects = 4

	EvictBulk = 64

	_ = "enumgen::Cs"
)

// EntryKind identifies a CS entry kind.
type EntryKind int

// EntryKind values.
const (
	EntryNone EntryKind = iota
	EntryMemory
	EntryDisk
	EntryIndirect

	_ = "enumgen:CsEntryKind:Cs"
)

// ListID identifies a list in the CS.
type ListID int

// ListID values.
const (
	ListDirect ListID = iota
	ListDirectT1
	ListDirectB1
	ListDirectT2
	ListDirectB2
	ListDirectDel
	ListIndirect

	ListDirectNew = 0

	_ = "enumgen:CsListID:Csl:List"
)
