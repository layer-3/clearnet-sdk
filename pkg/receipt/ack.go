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
	if code, ok := receiptVerificationAckMappings[verificationErr.Code]; ok {
		return code
	}
	return p2pproto.ReceiptAckTemporaryFailure
}

var receiptVerificationAckMappings = map[ReceiptVerificationCode]p2pproto.ReceiptAckCode{
	ReceiptVerificationMalformed:              p2pproto.ReceiptAckCorrupt,
	ReceiptVerificationInvalidSignatures:      p2pproto.ReceiptAckCorrupt,
	ReceiptVerificationSignerStateUnavailable: p2pproto.ReceiptAckSignerStateUnavailable,
	ReceiptVerificationStaleEpoch:             p2pproto.ReceiptAckStaleEpoch,
	ReceiptVerificationFutureEpoch:            p2pproto.ReceiptAckFutureEpoch,
	ReceiptVerificationIssuerUnavailable:      p2pproto.ReceiptAckTemporaryFailure,
	// TODO(clearnet): revisit permanent rejection once a resolver can prove
	// that an unknown withdrawal can never exist, rather than being behind.
	ReceiptVerificationUnknownWithdrawal:  p2pproto.ReceiptAckTemporaryFailure,
	ReceiptVerificationIssuerStateInvalid: p2pproto.ReceiptAckTemporaryFailure,
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
