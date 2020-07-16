package cs

//go:generate go run ../../mk/enumgen/ -guard=NDNDPDK_CS_ENUM_H -out=../../csrc/pcct/cs-enum.h .

// ListID identifies a list in the CS.
type ListID int

// ListID values.
const (
	ListMd ListID = iota
	ListMdT1
	ListMdB1
	ListMdT2
	ListMdB2
	ListMdDel
	ListMi

	_ = "enumgen:CsListID:Csl:List"
)
