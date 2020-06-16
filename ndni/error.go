package ndni

//go:generate stringer -type=NdnError -output=error_string.go

// NdnError indicates an error condition in ndni package.
type NdnError int

// Known NdnError numbers.
const (
	NdnErrOK NdnError = iota
	NdnErrIncomplete
	NdnErrAllocError
	NdnErrLengthOverflow
	NdnErrBadType
	NdnErrBadNni
	NdnErrFragmented
	NdnErrUnknownCriticalLpHeader
	NdnErrFragIndexExceedFragCount
	NdnErrLpHasTrailer
	NdnErrBadLpSeqNum
	NdnErrBadPitToken
	NdnErrNameIsEmpty
	NdnErrNameTooLong
	NdnErrBadNameComponentType
	NdnErrNameHasComponentAfterDigest
	NdnErrBadDigestComponentLength
	NdnErrBadNonceLength
	NdnErrBadInterestLifetime
	NdnErrBadHopLimitLength
	NdnErrHopLimitZero
)

func (e NdnError) Error() string {
	return e.String()
}
