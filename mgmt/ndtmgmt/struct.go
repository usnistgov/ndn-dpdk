package ndtmgmt

type UpdateArgs struct {
	Instructions []UpdateInstn
}

type UpdateInstn struct {
	Hash  uint64
	Value uint8
}
