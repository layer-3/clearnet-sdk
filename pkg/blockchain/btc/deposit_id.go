package btc

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/layer-3/clearnet-sdk/pkg/blockchain/internal/depositid"
)

// DepositID returns the Bitcoin deposit ID custody signs for output vout of
// transaction txID: "<txid>:<vout>". txID is expected as lowercase hex, as
// Bitcoin Core reports it; its case is kept as given. MintReceipts are signed
// over the result, so it must stay byte-identical to custody's; shared vectors
// in testdata/deposit_id_vectors.json at the repository root.
func DepositID(txID string, vout uint32) string {
	return txID + ":" + strconv.FormatUint(uint64(vout), 10)
}

// parseDepositID returns the output index of depositID, which must be exactly
// DepositID(txHash, vout) for some vout.
func parseDepositID(txHash, depositID string) (uint32, error) {
	if vout, ok := depositid.ParseIndex(depositID, txHash, 32); ok {
		return uint32(vout), nil
	}
	if !strings.HasPrefix(depositID, txHash+":") {
		return 0, fmt.Errorf("btc: deposit ID %q is not <txid>:<vout> for txid %s", depositID, txHash)
	}
	return 0, fmt.Errorf("btc: deposit ID %q has an invalid output index", depositID)
}

// PaysDepositScript reports whether an output of valueSats with scriptPubKey
// pays depositScript with a positive value. An empty depositScript matches
// nothing. Attribution is separate: the transaction must also carry exactly
// one valid marker (marker.ScanOutputs over every output script).
func PaysDepositScript(valueSats int64, scriptPubKey, depositScript []byte) bool {
	return valueSats > 0 && len(depositScript) > 0 && bytes.Equal(scriptPubKey, depositScript)
}
