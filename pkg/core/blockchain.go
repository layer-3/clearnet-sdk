package core

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/layer-3/clearnet-sdk/pkg/decimal"
)

// TxRef identifies a submitted L1 transaction across chains: Hash is the
// canonical 32-byte tx hash / id; Raw is an optional chain-native reference
// (e.g. a hex txid) for chains where the string form is primary.
type TxRef struct {
	Hash [32]byte
	Raw  string
}

// Chain-agnostic adapter interfaces for L1 interactions, split by concern: the
// custody vault (deposit/withdraw), the node registry, token balances, the
// fraud adjudicator, and the testnet faucet. Each per-chain implementation
// (e.g. pkg/blockchain/evm) provides a separate adapter struct per role rather
// than one monolith, so the centralized registry/faucet paths stay decoupled
// from the vault money path.

// WithdrawalFraudEvidence is the frozen two-object Slasher evidence shape
// (protocol/security.md §11): challenged signed withdrawal Block bytes plus
// the immediate pre-withdrawal signed header and one balance SMT proof.
type WithdrawalFraudEvidence struct {
	ChallengedObject []byte
	AnchorHeader     []byte
	AnchorSignature  []byte
	EntryIndex       uint64
	SMTProof         [][32]byte
	SMTBitmask       *big.Int
	BalanceKey       [32]byte
	ProvenBalance    *big.Int
}

// FraudEvidenceSubmitter provides write access to the on-chain Slasher.
type FraudEvidenceSubmitter interface {
	SubmitWithdrawalFraudEvidence(ctx context.Context, evidence WithdrawalFraudEvidence) error
}

// DepositStatus is the tri-state result of VerifyDeposit, distinguishing a
// deposit that settled, one that is still in flight, and one that never landed
// (or was dropped/reorged out).
type DepositStatus int

const (
	// DepositAbsent: no matching deposit transaction is on chain or in the
	// mempool (never broadcast, dropped, reorged out, or reverted/failed).
	DepositAbsent DepositStatus = iota
	// DepositPending: the deposit transaction is observed but not yet final —
	// in the mempool, or with fewer than the requested confirmations.
	DepositPending
	// DepositConfirmed: the deposit transaction is final to the requested depth.
	DepositConfirmed
)

func (s DepositStatus) String() string {
	switch s {
	case DepositAbsent:
		return "absent"
	case DepositPending:
		return "pending"
	case DepositConfirmed:
		return "confirmed"
	default:
		return "unknown"
	}
}

// VaultDepositor moves funds into the L1 vault. The implementation owns the
// depositor's signing identity (a sign.Signer supplied at construction) and
// executes the deposit on its chain: a contract call (EVM), a funding tx to a
// derived address (BTC), or a tagged Payment (XRPL). It expects only the asset,
// amount, and crediting clearnet account.
type VaultDepositor interface {
	SubmitDeposit(ctx context.Context, asset string, amount decimal.Decimal, account string) (TxRef, error)
	// VerifyDeposit reports whether the deposit identified by ref (a TxRef
	// returned by SubmitDeposit) is present and final on chain — a pure read for
	// replay/audit. minConf is the confirmation depth required for
	// DepositConfirmed; chains with no numeric depth (Solana) map it onto a
	// commitment level instead.
	VerifyDeposit(ctx context.Context, ref TxRef, minConf uint64) (DepositStatus, error)
}

// VaultWithdrawalFinalizer turns an authorized withdrawal into an on-chain
// release, as a sequence each custody node runs over a caller-orchestrated
// quorum. The implementation owns the node's signer and the chain-specific
// authorization (vault address, signer set, fee policy, ticket source) supplied
// at construction.
//
//   - Pack returns the canonical bytes to be signed for this withdrawal.
//   - Validate re-derives the trust-bound shape from the op and asserts the
//     packed bytes match — the defense against a Byzantine packer; every node
//     runs it before Sign.
//   - Sign produces this node's signature over the packed bytes.
//   - Submit merges the packed bytes with the collected quorum signatures into
//     a submittable artifact and broadcasts it. It filters the signatures
//     against the live on-chain signer set and is idempotent against a
//     withdrawal a peer has already executed.
//   - VerifyExecution reads canonical chain state to answer "already executed?"
//     for the retry/finalize loop.
type VaultWithdrawalFinalizer interface {
	Pack(ctx context.Context, op *WithdrawalOp, withdrawalID [32]byte) ([]byte, error)
	Validate(ctx context.Context, packed []byte, op *WithdrawalOp, withdrawalID [32]byte) error
	Sign(ctx context.Context, packed []byte) ([]byte, error)
	Submit(ctx context.Context, packed []byte, signatures [][]byte) (TxRef, error)
	VerifyExecution(ctx context.Context, withdrawalID [32]byte) (txHash [32]byte, executed bool, err error)
}

// SignerRotationFinalizer rotates the vault's authorized signer set. It is the
// same build→sign→merge→submit→verify shape as VaultWithdrawalFinalizer, but the
// signed payload commits to the new signer set + threshold + the chain's local
// replay token (EVM signerNonce, XRPL account sequence, Solana program nonce)
// rather than a withdrawal. Signature collection (mesh) and submitter selection
// stay with the caller; the implementation owns the node's signer and the
// chain-specific authorization supplied at construction.
//
// newSigners is the chain-native encoding of the incoming set — EVM/XRPL
// addresses, BTC 33-byte compressed pubkeys (hex), Solana ed25519 pubkeys (hex)
// — matching custody's RotationRequest. newThreshold is the new k-of-n quorum.
//
// In-place chains (EVM, XRPL, Solana) mutate on-chain signer state at a fixed
// vault address. BTC has no in-place form: its P2WSH vault address is a function
// of the signer set, so rotation is a sweep of every old-vault UTXO into the
// newly-derived vault. The BTC implementation hides that behind the same
// interface via a vault store supplied at construction (it pivots to the new
// vault on confirmation); to callers all four chains rotate identically.
//
//   - Pack returns the canonical bytes to be signed for this rotation.
//   - Validate re-derives the trust-bound shape and asserts the packed bytes
//     match — the Byzantine-packer defense; every node runs it before Sign.
//   - Sign produces this node's signature over the packed bytes.
//   - Submit merges the collected signatures against the live (outgoing) signer
//     set and broadcasts the rotation. Idempotent against an already-applied
//     rotation.
//   - VerifyRotation reads canonical chain state to answer "is the set now the
//     requested one?" — binary (done or not), the signal each node uses to close
//     the dual-sign window and drop the outgoing key.
type SignerRotationFinalizer interface {
	Pack(ctx context.Context, newSigners []string, newThreshold int) ([]byte, error)
	Validate(ctx context.Context, packed []byte, newSigners []string, newThreshold int) error
	Sign(ctx context.Context, packed []byte) ([]byte, error)
	Submit(ctx context.Context, packed []byte, signatures [][]byte) (TxRef, error)
	VerifyRotation(ctx context.Context, newSigners []string, newThreshold int) (txHash [32]byte, done bool, err error)
}

// RegistryReader provides read access to the L1 node registry.
//
// Naming follows the on-chain `IRegistry` surface:
//   - `TotalNodes` / `GetNodes` enumerate active + unbonding; active-only
//     callers use `ActiveCount` plus `GetActiveNodes` (which filters
//     `deactivatedAt == 0` on the Go side).
//   - `FloorPrice` is the activation-time floor at the next cardinality.
//   - `GetNodeId(tokenId)` resolves an NFT to the currently-locked nodeId.
type RegistryReader interface {
	GetNodeByID(ctx context.Context, nodeID [32]byte) (*Slot, error)
	GetNodes(ctx context.Context, offset, limit *big.Int) ([]*Slot, error)
	TotalNodes(ctx context.Context) (*big.Int, error)
	GetActiveNodes(ctx context.Context, offset, limit *big.Int) ([]*Slot, error)
	ActiveCount(ctx context.Context) (uint32, error)
	FloorPrice(ctx context.Context) (*big.Int, error)
	GetNodeId(ctx context.Context, tokenId uint32) ([32]byte, error)
	UnbondingPeriod(ctx context.Context) (uint64, error)
}

// RegistryWriter provides write access to the L1 node registry.
//
// `Lock` is a high-level onboarding helper: it calls `Registry.register`,
// which mints a NodeID NFT directly into Registry escrow and locks collateral.
// The `popSignature` parameter is accepted for source-compat but ignored on
// chain (ADR-008 2026-05-08: PoP verification moved to off-chain tooling).
type RegistryWriter interface {
	Lock(ctx context.Context, blsPubkeyG1 [2]*big.Int, blsPubkeyG2 [4]*big.Int, popSignature [2]*big.Int, maxPrice *big.Int) (uint32, error)
	Unlock(ctx context.Context, tokenId uint32) error
	Release(ctx context.Context, tokenId uint32) error
	Fund(ctx context.Context, tokenId uint32, amount *big.Int) error
}

// TokenReader provides read access to ERC-20 token balances. token is the
// token contract address (hex); account is the holder address (hex).
type TokenReader interface {
	BalanceOf(ctx context.Context, token string, account string) (*big.Int, error)
}

// FaucetReader provides read access to the testnet faucet's parameters.
type FaucetReader interface {
	DripAmount(ctx context.Context) (*big.Int, error)
	Cooldown(ctx context.Context) (*big.Int, error)
	Owner(ctx context.Context) (common.Address, error)
	LastDrip(ctx context.Context, addr common.Address) (*big.Int, error)
}

// FaucetWriter provides write access to the testnet faucet drip.
type FaucetWriter interface {
	Drip(ctx context.Context) error
	DripTo(ctx context.Context, recipient common.Address) error
}
