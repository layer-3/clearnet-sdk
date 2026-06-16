package evm

import (
	"bytes"
	"context"
	"fmt"
	"sort"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// fetchLiveQuorum reads the vault's current authorized signer set and threshold.
// The outgoing/current quorum is what authorizes both execute and updateSigners,
// so withdrawal and rotation both size and filter against it.
func fetchLiveQuorum(ctx context.Context, custody *Custody) ([]common.Address, int, error) {
	signers, err := custody.Signers(&bind.CallOpts{Context: ctx})
	if err != nil {
		return nil, 0, fmt.Errorf("read signers: %w", err)
	}
	thr, err := custody.Threshold(&bind.CallOpts{Context: ctx})
	if err != nil {
		return nil, 0, fmt.Errorf("read threshold: %w", err)
	}
	if !thr.IsInt64() || thr.Int64() <= 0 || thr.Int64() > int64(len(signers)) {
		return nil, 0, fmt.Errorf("on-chain threshold %s out of range for %d signers", thr, len(signers))
	}
	return signers, int(thr.Int64()), nil
}

// mergeQuorumSigs filters the collected signatures over digest against the live
// signer set, drops duplicates and unauthorized recoveries, trims to the live
// threshold, orders by signer address (Custody.sol requires ascending, no
// duplicates), and shifts V to {27,28}. It returns the contract-ready signature
// list. Both Custody.execute (withdrawal) and Custody.updateSigners (rotation)
// share this verification shape, differing only in the digest.
func mergeQuorumSigs(digest common.Hash, signatures [][]byte, liveSigners []common.Address, liveThreshold int) ([][]byte, error) {
	authorized := make(map[common.Address]struct{}, len(liveSigners))
	for _, a := range liveSigners {
		authorized[a] = struct{}{}
	}

	type sigAddr struct {
		sig  []byte
		addr common.Address
	}
	kept := make([]sigAddr, 0, len(signatures))
	seen := make(map[common.Address]struct{})
	for _, s := range signatures {
		if len(s) != 65 {
			return nil, fmt.Errorf("signature has wrong length %d", len(s))
		}
		pub, err := crypto.SigToPub(digest[:], s)
		if err != nil {
			return nil, fmt.Errorf("recover signer: %w", err)
		}
		addr := crypto.PubkeyToAddress(*pub)
		if _, ok := authorized[addr]; !ok {
			continue // not in the live signer set
		}
		if _, dup := seen[addr]; dup {
			continue
		}
		seen[addr] = struct{}{}
		kept = append(kept, sigAddr{sig: s, addr: addr})
	}
	if len(kept) < liveThreshold {
		return nil, fmt.Errorf("only %d of %d authorized signatures", len(kept), liveThreshold)
	}
	// Custody.sol's _verifySignatures stops at `threshold` and rejects extras.
	kept = kept[:liveThreshold]
	// Ascending uint160 order == bytes order over [20]byte.
	sort.Slice(kept, func(i, j int) bool { return bytes.Compare(kept[i].addr[:], kept[j].addr[:]) < 0 })

	sigs := make([][]byte, len(kept))
	for i, k := range kept {
		cp := make([]byte, 65)
		copy(cp, k.sig)
		if cp[64] < 27 {
			cp[64] += 27 // shift V {0,1} -> {27,28} at the contract boundary
		}
		sigs[i] = cp
	}
	return sigs, nil
}
