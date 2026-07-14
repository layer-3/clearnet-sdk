package bls

import (
	"testing"

	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/decimal"
)

func TestComputeVaultWithdrawalID_Deterministic(t *testing.T) {
	in := withdrawalIDInputFixture()
	a := computeWithdrawalIDFromFixture(in)
	b := computeWithdrawalIDFromFixture(in)
	if a != b {
		t.Fatalf("withdrawal ID not deterministic: %x != %x", a, b)
	}
}

func TestComputeVaultWithdrawalID_FieldSensitivity(t *testing.T) {
	base := withdrawalIDInputFixture()
	baseID := computeWithdrawalIDFromFixture(base)

	cases := []struct {
		name string
		mut  func(*withdrawalIDInput)
	}{
		{"AccountID", func(in *withdrawalIDInput) { in.accountID[0] ^= 0xff }},
		{"BlockHash", func(in *withdrawalIDInput) { in.blockHash[0] ^= 0xff }},
		{"EntryIndex", func(in *withdrawalIDInput) { in.entryIndex++ }},
		{"AssetURI", func(in *withdrawalIDInput) {
			in.assetURI = "yellow://ynet/asset/custody/evm/1/0xa0b8000000000000000000000000000000000002"
		}},
		{"Amount", func(in *withdrawalIDInput) { in.amount = decimal.NewFromInt(2) }},
		{"Recipient", func(in *withdrawalIDInput) { in.recipient = "rRecipient2" }},
		{"Nonce", func(in *withdrawalIDInput) { in.nonce++ }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			next := base
			tc.mut(&next)
			if got := computeWithdrawalIDFromFixture(next); got == baseID {
				t.Fatalf("%s mutation did not change withdrawal ID", tc.name)
			}
		})
	}
}

func TestComputeVaultWithdrawalID_NoVariableFieldBoundaryCollision(t *testing.T) {
	inA := withdrawalIDInputFixture()
	inA.assetURI = "yellow://ynet/asset/custody/a"
	inA.recipient = "bc"

	inB := withdrawalIDInputFixture()
	inB.assetURI = "yellow://ynet/asset/custody/ab"
	inB.recipient = "c"

	a := computeWithdrawalIDFromFixture(inA)
	b := computeWithdrawalIDFromFixture(inB)
	if a == b {
		t.Fatalf("withdrawal ID collided across assetURI/recipient boundary: %x", a)
	}
}

type withdrawalIDInput struct {
	accountID  [32]byte
	blockHash  [32]byte
	entryIndex uint64
	assetURI   core.AssetURI
	amount     decimal.Decimal
	recipient  string
	nonce      uint64
}

func withdrawalIDInputFixture() withdrawalIDInput {
	return withdrawalIDInput{
		accountID:  [32]byte{0x11},
		blockHash:  [32]byte{0x22},
		entryIndex: 7,
		assetURI:   "yellow://ynet/asset/custody/evm/1/0xa0b8000000000000000000000000000000000001",
		amount:     decimal.NewFromInt(1),
		recipient:  "rRecipient",
		nonce:      42,
	}
}

func computeWithdrawalIDFromFixture(in withdrawalIDInput) [32]byte {
	return ComputeVaultWithdrawalID(
		in.accountID,
		in.blockHash,
		in.entryIndex,
		in.assetURI,
		in.amount,
		in.recipient,
		in.nonce,
	)
}
