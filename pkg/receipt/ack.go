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
	case ReceiptVerificationMalformed, ReceiptVerificationInvalidSignatures:
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
		// TODO(clearnet): revisit permanent rejection once a resolver can prove
		// that an unknown withdrawal can never exist, rather than being behind.
		return p2pproto.ReceiptAckTemporaryFailure
	case ReceiptVerificationIssuerStateInvalid:
		return p2pproto.ReceiptAckTemporaryFailure
	default:
		return p2pproto.ReceiptAckTemporaryFailure
	}
}

var receiptVerificationCodes = []ReceiptVerificationCode{
	ReceiptVerificationMalformed,
	ReceiptVerificationSignerStateUnavailable,
	ReceiptVerificationStaleEpoch,
	ReceiptVerificationFutureEpoch,
	ReceiptVerificationInvalidSignatures,
	ReceiptVerificationIssuerUnavailable,
	ReceiptVerificationUnknownWithdrawal,
	ReceiptVerificationIssuerStateInvalid,
}
