// Package marker is Yellow-Custody-specific, deposit-marker format v1.
//
// It implements the OP_RETURN attribution marker: the payload
// "YNET" || version || eth_addr [|| reference] that a depositing transaction
// carries so the vault can credit an account without a per-account address.
// The format is normative for anyone building a Yellow BTC deposit; the JSON
// golden vectors under testdata/ are the language-neutral statement of it and
// are the artifact a non-Go implementation should conform to.
//
// Scope boundary: this package parses and builds markers only, and knows nothing
// about the rest of the deposit transaction.
package marker
