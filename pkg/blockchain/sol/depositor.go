package sol

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"

	"github.com/layer-3/clearnet-sdk/pkg/blockchain/sol/custody"
	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/decimal"
	"github.com/layer-3/clearnet-sdk/pkg/sign"
)

// Depositor moves funds into the custody vault on Solana, signed by the
// depositor's own ed25519 key. It implements core.VaultDepositor. Native SOL
// and SPL tokens are both supported. The deposit credits the 20-byte clearnet
// account encoded in `account` (hex).
type Depositor struct {
	client       *rpc.Client
	programID    solana.PublicKey
	vaultPDA     solana.PublicKey
	eventAuth    solana.PublicKey
	signer       sign.Signer
	depositorPub solana.PublicKey
	commitment   rpc.CommitmentType
}

var _ core.VaultDepositor = (*Depositor)(nil)

// NewDepositor builds the Solana depositor over the JSON-RPC at rpcURL. signer
// is the depositor's ed25519 key (it pays + funds). commitment is the level the
// deposit tx's blockhash + preflight use; empty → CommitmentFinalized (see the
// NOTE on the withdrawal finalizer's Config.Commitment for the test tradeoff).
func NewDepositor(rpcURL string, programID solana.PublicKey, signer sign.Signer, commitment rpc.CommitmentType) (*Depositor, error) {
	pub, err := solanaPub(signer)
	if err != nil {
		return nil, err
	}
	if commitment == "" {
		commitment = rpc.CommitmentFinalized
	}
	return &Depositor{
		client:       rpc.New(rpcURL),
		programID:    programID,
		vaultPDA:     VaultPDA(programID),
		eventAuth:    eventAuthorityPDA(programID),
		signer:       signer,
		depositorPub: pub,
		commitment:   commitment,
	}, nil
}

// DepositorAddress returns the depositor's Solana address.
func (d *Depositor) DepositorAddress() string { return d.depositorPub.String() }

// SubmitDeposit transfers `amount` of `asset` into the vault, crediting clearnet
// dest.Account (20-byte hex) with the optional ADR-015 dest.Ref sub-account
// reference. asset is "" / "SOL" for native or a base58 mint.
func (d *Depositor) SubmitDeposit(ctx context.Context, asset string, amount decimal.Decimal, dest core.DepositDestination) (core.TxRef, error) {
	acct, err := parseClearnetAccount(dest.Account)
	if err != nil {
		return core.TxRef{}, err
	}
	amt := amount.BigInt()
	if !amt.IsUint64() || amt.Sign() <= 0 {
		return core.TxRef{}, fmt.Errorf("sol: amount %s not a positive uint64", amount.String())
	}
	lamports := amt.Uint64()
	mint, err := resolveMint(asset)
	if err != nil {
		return core.TxRef{}, err
	}

	var ix solana.Instruction
	if mint.IsZero() {
		ix, err = custody.NewDepositSolInstruction(
			acct, dest.Ref, lamports,
			d.depositorPub, d.vaultPDA, solana.SystemProgramID, d.eventAuth, d.programID,
		)
	} else {
		depositorATA, _, e := solana.FindAssociatedTokenAddress(d.depositorPub, mint)
		if e != nil {
			return core.TxRef{}, fmt.Errorf("sol: depositor ATA: %w", e)
		}
		vaultATA, _, e := solana.FindAssociatedTokenAddress(d.vaultPDA, mint)
		if e != nil {
			return core.TxRef{}, fmt.Errorf("sol: vault ATA: %w", e)
		}
		ix, err = custody.NewDepositSplInstruction(
			acct, dest.Ref, lamports,
			d.depositorPub, mint, depositorATA, d.vaultPDA, vaultATA,
			solana.TokenProgramID, solana.SPLAssociatedTokenAccountProgramID, d.eventAuth, d.programID,
		)
	}
	if err != nil {
		return core.TxRef{}, fmt.Errorf("sol: build deposit ix: %w", err)
	}

	sig, err := signAndSend(ctx, d.client, []solana.Instruction{ix}, d.depositorPub, d.signer, d.commitment, solana.PublicKey{})
	if err != nil {
		return core.TxRef{}, err
	}
	return txRef(sig), nil
}

// VerifyDeposit reports the on-chain status of the deposit tx in ref (matched by
// signature, ref.Raw). minConf maps onto Solana's commitment ladder, which has
// no numeric depth: minConf 0 accepts the optimistic "confirmed" level (~1-2
// slots), while minConf >= 1 requires "finalized" (irreversible). A failed tx
// reads as DepositAbsent (it credited nothing).
func (d *Depositor) VerifyDeposit(ctx context.Context, ref core.TxRef, minConf uint64) (core.DepositStatus, error) {
	sig, err := solana.SignatureFromBase58(ref.Raw)
	if err != nil {
		return core.DepositAbsent, fmt.Errorf("sol: bad signature %q: %w", ref.Raw, err)
	}
	out, err := d.client.GetSignatureStatuses(ctx, true, sig)
	if err != nil {
		if errors.Is(err, rpc.ErrNotFound) {
			return core.DepositAbsent, nil
		}
		return core.DepositAbsent, fmt.Errorf("sol: signature status: %w", err)
	}
	if len(out.Value) == 0 || out.Value[0] == nil {
		return core.DepositAbsent, nil
	}
	st := out.Value[0]
	if st.Err != nil {
		return core.DepositAbsent, nil
	}
	switch st.ConfirmationStatus {
	case rpc.ConfirmationStatusFinalized:
		return core.DepositConfirmed, nil
	case rpc.ConfirmationStatusConfirmed:
		if minConf == 0 {
			return core.DepositConfirmed, nil
		}
		return core.DepositPending, nil
	default:
		return core.DepositPending, nil
	}
}

// parseClearnetAccount decodes a 20-byte clearnet account address from hex
// (optionally a yellow://.../user/<hex> URI's last segment).
func parseClearnetAccount(account string) ([20]byte, error) {
	seg := account
	if i := strings.LastIndex(seg, "/"); i >= 0 {
		seg = seg[i+1:]
	}
	seg = strings.TrimPrefix(strings.ToLower(seg), "0x")
	b, err := hex.DecodeString(seg)
	if err != nil || len(b) != 20 {
		return [20]byte{}, fmt.Errorf("sol: account %q must be a 20-byte hex address (len=%d): %v", account, len(b), err)
	}
	var out [20]byte
	copy(out[:], b)
	return out, nil
}

// txRef builds a core.TxRef from a Solana signature: Hash = sha256(sig) (the
// 32-byte receipt form clearnet uses), Raw = the base58 signature.
func txRef(sig solana.Signature) core.TxRef {
	h := sha256.Sum256(sig[:])
	return core.TxRef{Hash: h, Raw: sig.String()}
}
