package xrpl

import (
	"encoding/hex"
	"strings"

	"github.com/Peersyst/xrpl-go/xrpl/transaction"
)

// DepositID returns the XRPL deposit ID custody signs: the transaction hash
// lower-cased. MintReceipts are signed over it, so it must stay byte-identical
// to custody's; shared vectors in testdata/deposit_id_vectors.json at the
// repository root.
func DepositID(txHash string) string {
	return strings.ToLower(txHash)
}

// IsVaultDepositPayment reports whether tx is a Payment to vaultAddress from
// another account. Addresses compare exactly. Validation, the result code and
// the memo are checked separately.
func IsVaultDepositPayment(tx transaction.FlatTransaction, vaultAddress string) bool {
	if txType, _ := tx["TransactionType"].(string); txType != "Payment" {
		return false
	}
	if src, _ := tx["Account"].(string); src == vaultAddress {
		return false
	}
	dst, _ := tx["Destination"].(string)
	return dst == vaultAddress
}

// ParseDepositMemo returns the account and ADR-015 reference of the first
// ynet-account memo in memos (the decoded Memos field) whose MemoData is
// exactly a 20-byte account followed by a 32-byte reference. A zero account
// is returned as parsed; it never names a real account.
func ParseDepositMemo(memos any) (account [20]byte, reference [32]byte, ok bool) {
	entries, isArray := memos.([]any)
	if !isArray {
		return account, reference, false
	}
	for _, entry := range entries {
		wrapper, isMap := entry.(map[string]any)
		if !isMap {
			continue
		}
		memo, isMap := wrapper["Memo"].(map[string]any)
		if !isMap {
			continue
		}
		typeHex, _ := memo["MemoType"].(string)
		memoType, err := hex.DecodeString(typeHex)
		if err != nil || string(memoType) != accountMemoType {
			continue
		}
		dataHex, _ := memo["MemoData"].(string)
		data, err := hex.DecodeString(dataHex)
		if err != nil || len(data) != len(account)+len(reference) {
			continue
		}
		copy(account[:], data[:20])
		copy(reference[:], data[20:])
		return account, reference, true
	}
	return account, reference, false
}
