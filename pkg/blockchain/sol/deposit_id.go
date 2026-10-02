package sol

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"

	"github.com/layer-3/clearnet-sdk/pkg/blockchain/internal/depositid"
)

// DepositID returns the Solana deposit ID custody signs: "0x" + lowercase hex
// of sha256(signature), ":", and index in decimal. index is the Deposited
// event's ProgramEvents Index; it is 0 for a transaction SubmitDeposit builds.
// MintReceipts are signed over the result, so it must stay byte-identical to
// custody's; shared vectors in testdata/deposit_id_vectors.json at the
// repository root.
func DepositID(sig solana.Signature, index uint64) string {
	return depositIDPrefix(sig) + ":" + strconv.FormatUint(index, 10)
}

func depositIDPrefix(sig solana.Signature) string {
	h := sha256.Sum256(sig[:])
	return "0x" + hex.EncodeToString(h[:])
}

// parseDepositID returns the event index of depositID, which must be exactly
// DepositID(sig, index) for some index.
func parseDepositID(sig solana.Signature, depositID string) (uint64, error) {
	if index, ok := depositid.ParseIndex(depositID, depositIDPrefix(sig), 64); ok {
		return index, nil
	}
	if !strings.Contains(depositID, ":") {
		return 0, fmt.Errorf("sol: deposit ID %q is not 0x<signature hash>:<index>", depositID)
	}
	return 0, fmt.Errorf("sol: deposit ID %q does not match signature %s", depositID, sig)
}

// depositEventAt reports whether the event at index is a Deposited event.
func depositEventAt(tx *solana.Transaction, meta *rpc.TransactionMeta, programID solana.PublicKey, index uint64) bool {
	events, _ := ProgramEvents(tx, meta, programID)
	return index < uint64(len(events)) && events[index].Kind == EventDeposited
}
