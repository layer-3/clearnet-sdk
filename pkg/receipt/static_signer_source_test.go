package receipt

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestNewStaticSignerSourceRejectsZeroEpoch(t *testing.T) {
	_, err := NewStaticSignerSource(0, []common.Address{common.HexToAddress("0x0000000000000000000000000000000000000001")}, 1)
	if err == nil {
		t.Fatal("zero signer epoch unexpectedly accepted")
	}
}
