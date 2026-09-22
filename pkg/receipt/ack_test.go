package receipt

import (
	"errors"
	"testing"

	p2pproto "github.com/layer-3/clearnet-sdk/pkg/p2p/protocol"
)

func TestReceiptVerificationAckCode(t *testing.T) {
	tests := []struct {
		code ReceiptVerificationCode
		want p2pproto.ReceiptAckCode
	}{
		{ReceiptVerificationMalformed, p2pproto.ReceiptAckCorrupt},
		{ReceiptVerificationInvalidSignatures, p2pproto.ReceiptAckCorrupt},
		{ReceiptVerificationSignerStateUnavailable, p2pproto.ReceiptAckSignerStateUnavailable},
		{ReceiptVerificationStaleEpoch, p2pproto.ReceiptAckStaleEpoch},
		{ReceiptVerificationFutureEpoch, p2pproto.ReceiptAckFutureEpoch},
		{ReceiptVerificationIssuerUnavailable, p2pproto.ReceiptAckTemporaryFailure},
		{ReceiptVerificationUnknownWithdrawal, p2pproto.ReceiptAckTemporaryFailure},
		{ReceiptVerificationIssuerStateInvalid, p2pproto.ReceiptAckTemporaryFailure},
	}
	for _, tc := range tests {
		err := &ReceiptVerificationError{Code: tc.code}
		if got := ReceiptVerificationAckCode(err); got != tc.want {
			t.Fatalf("%s maps to %s, want %s", tc.code, got, tc.want)
		}
	}
	if got := ReceiptVerificationAckCode(errors.New("plain")); got != p2pproto.ReceiptAckTemporaryFailure {
		t.Fatalf("plain error maps to %s, want temporary_failure", got)
	}
	if len(receiptVerificationCodes) != len(receiptVerificationAckMappings) {
		t.Fatalf("canonical verification code list has %d entries, want %d mappings", len(receiptVerificationCodes), len(receiptVerificationAckMappings))
	}
	for _, code := range receiptVerificationCodes {
		if _, ok := receiptVerificationAckMappings[code]; !ok {
			t.Fatalf("canonical verification code %s has no ACK mapping", code)
		}
	}
	if _, ok := receiptVerificationAckMappings[ReceiptVerificationCode("future_unlisted_code")]; ok {
		t.Fatal("unknown verification code unexpectedly has an explicit mapping")
	}
}
