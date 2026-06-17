package evm

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/layer-3/clearnet-sdk/pkg/core"
)

// FraudAdapter wraps the Slasher binding to submit withdrawal fraud evidence.
// It implements core.FraudEvidenceSubmitter.
type FraudAdapter struct {
	client  *ethclient.Client
	slasher *Slasher
	auth    *bind.TransactOpts
}

var _ core.FraudEvidenceSubmitter = (*FraudAdapter)(nil)

// NewFraudAdapter binds the Slasher at slasherAddr over client with a
// transactor for the given key.
func NewFraudAdapter(ctx context.Context, client *ethclient.Client, slasherAddr common.Address, key *ecdsa.PrivateKey) (*FraudAdapter, error) {
	slasher, err := NewSlasher(slasherAddr, client)
	if err != nil {
		return nil, fmt.Errorf("load slasher: %w", err)
	}
	auth, err := newTransactor(ctx, client, key)
	if err != nil {
		return nil, err
	}
	return &FraudAdapter{client: client, slasher: slasher, auth: auth}, nil
}

// SubmitWithdrawalFraudEvidence submits byte-exact withdrawal fraud evidence to
// Slasher.sol and waits for the slashing transaction to mine.
func (a *FraudAdapter) SubmitWithdrawalFraudEvidence(ctx context.Context, evidence core.WithdrawalFraudEvidence) error {
	provenBalance := evidence.ProvenBalance
	if provenBalance == nil {
		provenBalance = new(big.Int)
	}
	smtBitmask := evidence.SMTBitmask
	if smtBitmask == nil {
		smtBitmask = new(big.Int)
	}
	tx, err := a.slasher.SubmitWithdrawalFraudEvidence(
		txOpts(a.auth, ctx),
		evidence.ChallengedObject,
		evidence.AnchorHeader,
		evidence.AnchorSignature,
		evidence.EntryIndex,
		evidence.SMTProof,
		smtBitmask,
		evidence.BalanceKey,
		provenBalance,
	)
	if err != nil {
		return fmt.Errorf("submit withdrawal fraud evidence: %w", err)
	}
	return waitMined(ctx, a.client, tx)
}
