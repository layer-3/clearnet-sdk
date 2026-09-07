package receipt

import (
	"errors"

	p2pproto "github.com/layer-3/clearnet-sdk/pkg/p2p/protocol"
)

func ReceiptVerificationAckCode(err error) p2pproto.ReceiptAckCode {
	var verificationErr *ReceiptVerificationError
	if !errors.As(err, &verificationErr) {
		return p2pproto.ReceiptAckTemporaryFailure
	}
	switch verificationErr.Code {
	case ReceiptVerificationMalformed, ReceiptVerificationInvalidSignatures, ReceiptVerificationIssuerStateInvalid:
		return p2pproto.ReceiptAckCorrupt
	case ReceiptVerificationSignerStateUnavailable:
		return p2pproto.ReceiptAckSignerStateUnavailable
	case ReceiptVerificationStaleEpoch:
		return p2pproto.ReceiptAckStaleEpoch
	case ReceiptVerificationFutureEpoch:
		return p2pproto.ReceiptAckFutureEpoch
	case ReceiptVerificationIssuerUnavailable:
		return p2pproto.ReceiptAckTemporaryFailure
	case ReceiptVerificationUnknownWithdrawal:
		return p2pproto.ReceiptAckRejected
	default:
		return p2pproto.ReceiptAckTemporaryFailure
	}
}
