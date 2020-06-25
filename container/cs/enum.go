package cs

//go:generate go run ../../mk/enumgen/ -guard=NDN_DPDK_PCCT_CS_ENUM_H -out=../../csrc/pcct/cs-enum.h .

// ListID identifies a list in the CS.
type ListID int

// ListID values.
const (
	CslMd ListID = iota
	CslMdT1
	CslMdB1
	CslMdT2
	CslMdB2
	CslMdDel
	CslMi

	_ = "enumgen:CsListId"
)
