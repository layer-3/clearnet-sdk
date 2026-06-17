package sol

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/gagliardetto/solana-go"
	addresslookuptable "github.com/gagliardetto/solana-go/programs/address-lookup-table"
	"github.com/gagliardetto/solana-go/rpc"

	"github.com/layer-3/clearnet-sdk/pkg/blockchain/sol/custody"
	"github.com/layer-3/clearnet-sdk/pkg/sign"
)

// signAndSend builds a transaction over the instructions, signs it with the
// single fee-payer (the quorum's signatures ride inside the Ed25519 instruction,
// not the tx signers), and broadcasts it. When alt is non-zero it loads that
// Address Lookup Table and emits a v0 transaction, compressing the account list
// so large quorums (whose Ed25519 instruction is already large) still fit the
// 1232-byte packet limit; otherwise it emits a legacy transaction.
func signAndSend(ctx context.Context, client *rpc.Client, instructions []solana.Instruction, payerPub solana.PublicKey, payer sign.Signer, commitment rpc.CommitmentType, alt solana.PublicKey) (solana.Signature, error) {
	if commitment == "" {
		commitment = rpc.CommitmentFinalized
	}
	// The blockhash and the preflight simulation use the same commitment as the
	// caller's reads — a mismatch (e.g. a finalized blockhash while funds were
	// only just confirmed) evaluates the tx against stale state and misses the
	// credit / can't find the fresh blockhash.
	bh, err := client.GetLatestBlockhash(ctx, commitment)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("sol: latest blockhash: %w", err)
	}
	opts := []solana.TransactionOption{solana.TransactionPayer(payerPub)}
	if !alt.IsZero() {
		state, err := addresslookuptable.GetAddressLookupTable(ctx, client, alt)
		if err != nil {
			return solana.Signature{}, fmt.Errorf("sol: load ALT: %w", err)
		}
		opts = append(opts, solana.TransactionAddressTables(map[solana.PublicKey]solana.PublicKeySlice{
			alt: state.Addresses,
		}))
	}
	tx, err := solana.NewTransaction(instructions, bh.Value.Blockhash, opts...)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("sol: build tx: %w", err)
	}
	msg, err := tx.Message.MarshalBinary()
	if err != nil {
		return solana.Signature{}, fmt.Errorf("sol: marshal message: %w", err)
	}
	sigBytes, err := payer.Sign(ctx, msg)
	if err != nil {
		return solana.Signature{}, fmt.Errorf("sol: sign tx: %w", err)
	}
	var sig solana.Signature
	copy(sig[:], sigBytes)
	tx.Signatures = []solana.Signature{sig}
	if _, err := client.SendTransactionWithOpts(ctx, tx, rpc.TransactionOpts{PreflightCommitment: commitment}); err != nil {
		return solana.Signature{}, fmt.Errorf("sol: send tx: %w", err)
	}
	return sig, nil
}

// solanaPub maps a sign.Signer's ed25519 public key to a Solana pubkey.
func solanaPub(s sign.Signer) (solana.PublicKey, error) {
	pub := s.PublicKey()
	if s.Algorithm() != sign.AlgEd25519 || len(pub) != 32 {
		return solana.PublicKey{}, fmt.Errorf("sol: signer must be ed25519 with a 32-byte key, got %s/%d", s.Algorithm(), len(pub))
	}
	return solana.PublicKeyFromBytes(pub), nil
}

// ed25519ProgramID is Solana's native Ed25519 signature-verification precompile.
var ed25519ProgramID = solana.MustPublicKeyFromBase58("Ed25519SigVerify111111111111111111111111111")

// PDA seed prefixes — must match the Anchor program (programs/custody/src).
var (
	seedConfig         = []byte("config")
	seedVault          = []byte("vault")
	seedWithdrawal     = []byte("withdrawal")
	seedEventAuthority = []byte("__event_authority")
)

// ConfigPDA / VaultPDA / WithdrawalPDA / eventAuthorityPDA derive the program's
// deterministic accounts.
func ConfigPDA(programID solana.PublicKey) solana.PublicKey {
	pk, _, _ := solana.FindProgramAddress([][]byte{seedConfig}, programID)
	return pk
}

func VaultPDA(programID solana.PublicKey) solana.PublicKey {
	pk, _, _ := solana.FindProgramAddress([][]byte{seedVault}, programID)
	return pk
}

func WithdrawalPDA(programID solana.PublicKey, withdrawalID [32]byte) solana.PublicKey {
	pk, _, _ := solana.FindProgramAddress([][]byte{seedWithdrawal, withdrawalID[:]}, programID)
	return pk
}

func eventAuthorityPDA(programID solana.PublicKey) solana.PublicKey {
	pk, _, _ := solana.FindProgramAddress([][]byte{seedEventAuthority}, programID)
	return pk
}

// VaultLookupAddresses returns the invariant accounts of the execute
// instruction — the ones that recur across every withdrawal and are therefore
// eligible to populate an Address Lookup Table, letting large quorums fit a v0
// transaction (set the table via WithdrawalFinalizer's Config.AddressLookupTable).
// A zero mint returns the native-SOL set; a non-zero mint adds the token program
// and the vault's associated token account. Per-withdrawal accounts (the
// recipient, its token account, the Withdrawal PDA, the fee payer) vary each call
// and are intentionally excluded.
//
// Build the lookup table from this set so it stays in lockstep with the
// instruction's account layout (both live here, in the SDK).
func VaultLookupAddresses(programID, mint solana.PublicKey) []solana.PublicKey {
	addrs := []solana.PublicKey{
		ConfigPDA(programID),
		VaultPDA(programID),
		eventAuthorityPDA(programID),
		solana.SystemProgramID,
		solana.SysVarInstructionsPubkey,
	}
	if !mint.IsZero() {
		addrs = append(addrs, solana.TokenProgramID)
		if vaultATA, _, err := solana.FindAssociatedTokenAddress(VaultPDA(programID), mint); err == nil {
			addrs = append(addrs, vaultATA)
		}
	}
	return addrs
}

// BuildEd25519Instruction frames the quorum's signatures for the native
// Ed25519SigVerify precompile, all offsets self-referencing (instruction index
// 0xFFFF) so the verified data cannot be smuggled from another instruction.
// pubkeys/sigs are parallel (32-byte / 64-byte); message is the 32-byte digest.
func BuildEd25519Instruction(pubkeys, sigs [][]byte, message []byte) (solana.Instruction, error) {
	n := len(pubkeys)
	if n == 0 || n != len(sigs) {
		return nil, fmt.Errorf("sol: ed25519 needs matching non-empty pubkeys/sigs (%d/%d)", n, len(sigs))
	}
	if len(message) != 32 {
		return nil, fmt.Errorf("sol: ed25519 message must be 32 bytes, got %d", len(message))
	}
	const (
		header     = 2
		offsetSize = 14
		selfRef    = uint16(0xFFFF)
	)
	msgOffset := header + n*offsetSize
	pubkeysStart := msgOffset + 32
	sigsStart := pubkeysStart + n*32

	data := make([]byte, sigsStart+n*64)
	data[0] = byte(n)
	data[1] = 0
	put16 := func(at int, v uint16) { binary.LittleEndian.PutUint16(data[at:], v) }
	copy(data[msgOffset:], message)
	for i := 0; i < n; i++ {
		if len(pubkeys[i]) != 32 {
			return nil, fmt.Errorf("sol: pubkey %d not 32 bytes", i)
		}
		if len(sigs[i]) != 64 {
			return nil, fmt.Errorf("sol: signature %d not 64 bytes", i)
		}
		pkOff := pubkeysStart + i*32
		sigOff := sigsStart + i*64
		copy(data[pkOff:], pubkeys[i])
		copy(data[sigOff:], sigs[i])

		base := header + i*offsetSize
		put16(base+0, uint16(sigOff))
		put16(base+2, selfRef)
		put16(base+4, uint16(pkOff))
		put16(base+6, selfRef)
		put16(base+8, uint16(msgOffset))
		put16(base+10, 32)
		put16(base+12, selfRef)
	}
	return solana.NewInstruction(ed25519ProgramID, nil, data), nil
}

// fetchConfig reads the on-chain Config account (the live signer set + quorum)
// at the given commitment.
func fetchConfig(ctx context.Context, client *rpc.Client, programID solana.PublicKey, commitment rpc.CommitmentType) (*custody.Config, error) {
	info, err := client.GetAccountInfoWithOpts(ctx, ConfigPDA(programID), &rpc.GetAccountInfoOpts{Commitment: commitment})
	if err != nil {
		return nil, fmt.Errorf("sol: read config: %w", err)
	}
	if info == nil || info.Value == nil {
		return nil, fmt.Errorf("sol: config account not found (program not initialized?)")
	}
	return custody.ParseAccount_Config(info.Value.Data.GetBinary())
}

// resolveMint maps an asset string to its mint; the zero pubkey is native SOL.
func resolveMint(l1Asset string) (solana.PublicKey, error) {
	switch l1Asset {
	case "", "native", "SOL", "sol":
		return solana.PublicKey{}, nil
	default:
		mint, err := solana.PublicKeyFromBase58(l1Asset)
		if err != nil {
			return solana.PublicKey{}, fmt.Errorf("sol: l1_asset %q is not a base58 mint: %w", l1Asset, err)
		}
		return mint, nil
	}
}
