package btc

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	btcecdsa "github.com/btcsuite/btcd/btcec/v2/ecdsa"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"

	"github.com/layer-3/clearnet-sdk/pkg/core"
)

const (
	// PreparedWithdrawalFormatVersion is the current stable binary format for a
	// prepared Bitcoin withdrawal.
	PreparedWithdrawalFormatVersion uint16 = 3
	// PreparedPolicyFormatVersion is the current validation-policy snapshot
	// format embedded in every prepared transaction.
	PreparedPolicyFormatVersion uint16 = 1

	preparedWithdrawalMagic       = "BTPW"
	preparedRotationMagic         = "BTPR"
	preparedConsolidationMagic    = "BTPC"
	maxPreparedCanonicalBytes     = 4 << 20
	maxPreparedBinaryBytes        = 16 << 20
	maxPreparedInputs             = 100_000
	maxPreparedScriptBytes        = 10_000
	preparedWithdrawalHashBytes   = sha256.Size
	preparedWithdrawalTxIDBytes   = chainhash.HashSize
	preparedWithdrawalHeaderBytes = len(preparedWithdrawalMagic) + 2
	broadcastLookupTimeout        = 5 * time.Second
)

// ErrCandidateRejected classifies deterministic peer-provided candidate or
// share invalidity. RPC, context, backend, and local prepared-state errors do
// not wrap this sentinel.
var ErrCandidateRejected = errors.New("btc: candidate rejected")

// PreparedPolicy is an immutable-by-copy snapshot of the mutable admission
// settings used by offline prepared-transaction validation.
type PreparedPolicy struct {
	FormatVersion     uint16
	ConfirmationDepth uint64
	FeeCapSatPerVByte int64
}

// WithdrawalAuthorization is the immutable application authorization for one
// prepared withdrawal. Every operation on a prepared withdrawal must present
// the same authorization; the prepared binary stores its domain-separated
// hash, not a caller-mutable operation pointer.
type WithdrawalAuthorization struct {
	Operation    *core.WithdrawalOp
	WithdrawalID [32]byte
	Deadline     int64
}

// RotationAuthorization binds a sweep to its operation marker and exact
// incoming vault policy. Signers are canonicalized by compressed public key.
type RotationAuthorization struct {
	OperationID [32]byte
	Signers     []string
	Threshold   int
}

// ConsolidationAuthorization binds a fold to its marker and destination vault.
// TargetVaultIdentity is the canonical base-vault address returned by
// ConsolidationFinalizer.TargetVaultIdentity.
type ConsolidationAuthorization struct {
	OperationID         [32]byte
	TargetVaultIdentity string
}

type preparedOperationKind uint8

const (
	preparedOperationWithdrawal  preparedOperationKind = 1
	preparedOperationRotation    preparedOperationKind = 2
	preparedOperationConsolidate preparedOperationKind = 3
)

// PreparedVault is the complete immutable verifier snapshot for the vault
// which authorized the transaction inputs.
type PreparedVault struct {
	PubKeys        [][]byte
	Threshold      int
	AllowedScripts [][]byte
	Fingerprint    [sha256.Size]byte
}

// PreparedNetwork identifies the Bitcoin consensus network used at prepare
// time. Both fields are checked before any later cryptographic operation.
type PreparedNetwork struct {
	Name        string
	GenesisHash string
}

// preparedTransaction is the shared offline signing snapshot. It remains
// unexported so callers cannot mix snapshots authorized for different
// operations.
type preparedTransaction struct {
	formatVersion uint16
	operationKind preparedOperationKind
	authorization [sha256.Size]byte
	network       PreparedNetwork
	vault         PreparedVault
	canonicalTx   []byte
	expectedTxID  string
	inputs        []PreparedInput
	policy        PreparedPolicy
	integrityHash [sha256.Size]byte
}

// PreparedWithdrawal is an opaque snapshot of a validated withdrawal and its
// previous outputs. MarshalBinary and UnmarshalBinary provide its stable
// persistence format.
type PreparedWithdrawal struct {
	preparedTransaction
	withdrawalPrepared struct{}
}

// PreparedRotation is an opaque snapshot of a validated vault rotation.
type PreparedRotation struct {
	preparedTransaction
	rotationPrepared struct{}
}

// PreparedConsolidation is an opaque snapshot of a validated consolidation.
type PreparedConsolidation struct {
	preparedTransaction
	consolidationPrepared struct{}
}

func withdrawalTransaction(prepared *PreparedWithdrawal) *preparedTransaction {
	if prepared == nil {
		return nil
	}
	return &prepared.preparedTransaction
}

func rotationTransaction(prepared *PreparedRotation) *preparedTransaction {
	if prepared == nil {
		return nil
	}
	return &prepared.preparedTransaction
}

func consolidationTransaction(prepared *PreparedConsolidation) *preparedTransaction {
	if prepared == nil {
		return nil
	}
	return &prepared.preparedTransaction
}

// PreparedInput is the complete previous-output record for one canonical
// transaction input. Records are in exact transaction input order.
type PreparedInput struct {
	PreviousTxID         string
	PreviousVout         uint32
	AmountSats           int64
	PreviousOutputScript []byte
	WitnessScript        []byte
	Confirmations        int64
}

// FormatVersion returns the prepared-withdrawal binary format version.
func (p *PreparedWithdrawal) FormatVersion() uint16 {
	if p == nil {
		return 0
	}
	return p.formatVersion
}

// CanonicalBytes returns a defensive copy of the canonical unsigned transaction.
func (p *PreparedWithdrawal) CanonicalBytes() []byte {
	if p == nil {
		return nil
	}
	return append([]byte(nil), p.canonicalTx...)
}

// ExpectedTxID returns the transaction ID committed by the canonical body.
func (p *PreparedWithdrawal) ExpectedTxID() string {
	if p == nil {
		return ""
	}
	return p.expectedTxID
}

// Inputs returns a deep defensive copy of the ordered previous-output records.
func (p *PreparedWithdrawal) Inputs() []PreparedInput {
	if p == nil {
		return nil
	}
	return clonePreparedInputs(p.inputs)
}

// IntegrityHash returns the hash covering the complete versioned snapshot.
func (p *PreparedWithdrawal) IntegrityHash() [sha256.Size]byte {
	if p == nil {
		return [sha256.Size]byte{}
	}
	return p.integrityHash
}

// PolicySnapshot returns the immutable validation policy embedded at prepare time.
func (p *PreparedWithdrawal) PolicySnapshot() PreparedPolicy {
	if p == nil {
		return PreparedPolicy{}
	}
	return p.policy
}

// AuthorizationHash returns the domain-separated authorization commitment.
func (p *PreparedWithdrawal) AuthorizationHash() [sha256.Size]byte {
	if p == nil {
		return [sha256.Size]byte{}
	}
	return p.authorization
}

// VaultFingerprint returns the authorizing vault snapshot fingerprint.
func (p *PreparedWithdrawal) VaultFingerprint() [sha256.Size]byte {
	if p == nil {
		return [sha256.Size]byte{}
	}
	return p.vault.Fingerprint
}

// CanonicalBytes returns a defensive copy of the prepared rotation body.
func (p *PreparedRotation) CanonicalBytes() []byte {
	if p == nil {
		return nil
	}
	return append([]byte(nil), p.canonicalTx...)
}

// ExpectedTxID returns the prepared rotation transaction ID.
func (p *PreparedRotation) ExpectedTxID() string {
	if p == nil {
		return ""
	}
	return p.expectedTxID
}

// Inputs returns a defensive copy of the prepared rotation prevouts.
func (p *PreparedRotation) Inputs() []PreparedInput {
	if p == nil {
		return nil
	}
	return clonePreparedInputs(p.inputs)
}

// PolicySnapshot returns the rotation policy embedded at prepare time.
func (p *PreparedRotation) PolicySnapshot() PreparedPolicy {
	if p == nil {
		return PreparedPolicy{}
	}
	return p.policy
}

// CanonicalBytes returns a defensive copy of the prepared consolidation body.
func (p *PreparedConsolidation) CanonicalBytes() []byte {
	if p == nil {
		return nil
	}
	return append([]byte(nil), p.canonicalTx...)
}

// ExpectedTxID returns the prepared consolidation transaction ID.
func (p *PreparedConsolidation) ExpectedTxID() string {
	if p == nil {
		return ""
	}
	return p.expectedTxID
}

// Inputs returns a defensive copy of the prepared consolidation prevouts.
func (p *PreparedConsolidation) Inputs() []PreparedInput {
	if p == nil {
		return nil
	}
	return clonePreparedInputs(p.inputs)
}

// PolicySnapshot returns the consolidation policy embedded at prepare time.
func (p *PreparedConsolidation) PolicySnapshot() PreparedPolicy {
	if p == nil {
		return PreparedPolicy{}
	}
	return p.policy
}

// MarshalBinary returns the stable versioned representation of p. It rejects a
// context whose fields no longer match its integrity hash.
func (p PreparedWithdrawal) MarshalBinary() ([]byte, error) {
	return marshalTypedPrepared(preparedWithdrawalMagic, &p.preparedTransaction)
}

// MarshalBinary returns the type-tagged prepared rotation context.
func (p PreparedRotation) MarshalBinary() ([]byte, error) {
	return marshalTypedPrepared(preparedRotationMagic, &p.preparedTransaction)
}

// MarshalBinary returns the type-tagged prepared consolidation context.
func (p PreparedConsolidation) MarshalBinary() ([]byte, error) {
	return marshalTypedPrepared(preparedConsolidationMagic, &p.preparedTransaction)
}

func marshalTypedPrepared(magic string, transaction *preparedTransaction) ([]byte, error) {
	wantKind, err := operationKindForMagic(magic)
	if err != nil {
		return nil, err
	}
	if transaction == nil || transaction.operationKind != wantKind {
		return nil, fmt.Errorf("btc prepared: operation type mismatch")
	}
	if magic != preparedWithdrawalMagic {
		body, err := marshalPreparedTransactionBody(transaction)
		if err != nil {
			return nil, err
		}
		hash := sha256.Sum256(body)
		if hash != transaction.integrityHash {
			return nil, fmt.Errorf("btc prepared: integrity hash mismatch")
		}
		encoded := append([]byte(magic), body...)
		return append(encoded, hash[:]...), nil
	}
	body, err := marshalPreparedTransactionBody(transaction)
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256(body)
	if hash != transaction.integrityHash {
		return nil, fmt.Errorf("btc prepared: integrity hash mismatch")
	}
	if _, _, err := validatePreparedLocal(transaction); err != nil {
		return nil, fmt.Errorf("btc prepared: %w", err)
	}
	encoded := make([]byte, 0, len(body)+len(hash))
	encoded = append(encoded, body...)
	encoded = append(encoded, hash[:]...)
	return encoded, nil
}

// UnmarshalBinary decodes and verifies the stable versioned representation of
// a prepared withdrawal.
func (p *PreparedWithdrawal) UnmarshalBinary(data []byte) error {
	if p == nil {
		return fmt.Errorf("btc prepared withdrawal: nil receiver")
	}
	transaction, err := unmarshalTypedPrepared(preparedWithdrawalMagic, data)
	if err != nil {
		return err
	}
	p.preparedTransaction = transaction
	return nil
}

// UnmarshalBinary decodes only a type-tagged prepared rotation context.
func (p *PreparedRotation) UnmarshalBinary(data []byte) error {
	if p == nil {
		return fmt.Errorf("btc prepared rotation: nil receiver")
	}
	transaction, err := unmarshalTypedPrepared(preparedRotationMagic, data)
	if err != nil {
		return err
	}
	p.preparedTransaction = transaction
	return nil
}

// UnmarshalBinary decodes only a type-tagged prepared consolidation context.
func (p *PreparedConsolidation) UnmarshalBinary(data []byte) error {
	if p == nil {
		return fmt.Errorf("btc prepared consolidation: nil receiver")
	}
	transaction, err := unmarshalTypedPrepared(preparedConsolidationMagic, data)
	if err != nil {
		return err
	}
	p.preparedTransaction = transaction
	return nil
}

func unmarshalTypedPrepared(magic string, data []byte) (preparedTransaction, error) {
	wantKind, err := operationKindForMagic(magic)
	if err != nil {
		return preparedTransaction{}, err
	}
	if magic != preparedWithdrawalMagic {
		if len(data) < len(magic)+preparedWithdrawalHeaderBytes+preparedWithdrawalHashBytes || string(data[:len(magic)]) != magic {
			return preparedTransaction{}, fmt.Errorf("btc prepared: invalid operation type")
		}
		data = data[len(magic):]
	}
	if len(data) > maxPreparedBinaryBytes {
		return preparedTransaction{}, fmt.Errorf("btc prepared: binary length %d exceeds limit %d", len(data), maxPreparedBinaryBytes)
	}
	if len(data) < preparedWithdrawalHeaderBytes+preparedWithdrawalHashBytes {
		return preparedTransaction{}, fmt.Errorf("btc prepared: truncated binary")
	}
	body := data[:len(data)-preparedWithdrawalHashBytes]
	wantHash := sha256.Sum256(body)
	var gotHash [sha256.Size]byte
	copy(gotHash[:], data[len(body):])
	if gotHash != wantHash {
		return preparedTransaction{}, fmt.Errorf("btc prepared: integrity hash mismatch")
	}

	r := bytes.NewReader(body)
	encodedMagic := make([]byte, len(preparedWithdrawalMagic))
	if _, err := io.ReadFull(r, encodedMagic); err != nil {
		return preparedTransaction{}, fmt.Errorf("btc prepared: read magic: %w", err)
	}
	if string(encodedMagic) != preparedWithdrawalMagic {
		return preparedTransaction{}, fmt.Errorf("btc prepared: invalid operation type")
	}
	version, err := readPreparedUint16(r)
	if err != nil {
		return preparedTransaction{}, err
	}
	if version != PreparedWithdrawalFormatVersion {
		return preparedTransaction{}, fmt.Errorf("btc prepared: unsupported format version %d", version)
	}
	kindByte, err := r.ReadByte()
	if err != nil {
		return preparedTransaction{}, fmt.Errorf("btc prepared: read operation kind: %w", err)
	}
	kind := preparedOperationKind(kindByte)
	if kind != wantKind {
		return preparedTransaction{}, fmt.Errorf("btc prepared: invalid operation type")
	}
	var authorization [sha256.Size]byte
	if _, err := io.ReadFull(r, authorization[:]); err != nil {
		return preparedTransaction{}, fmt.Errorf("btc prepared: read authorization hash: %w", err)
	}
	networkName, err := readPreparedString(r, 128, "network name")
	if err != nil {
		return preparedTransaction{}, err
	}
	genesisHash, err := readPreparedString(r, chainhash.MaxHashStringSize, "genesis hash")
	if err != nil {
		return preparedTransaction{}, err
	}
	vault, err := readPreparedVault(r)
	if err != nil {
		return preparedTransaction{}, err
	}
	policyVersion, err := readPreparedUint16(r)
	if err != nil {
		return preparedTransaction{}, err
	}
	confirmationDepth, err := readPreparedUint64(r)
	if err != nil {
		return preparedTransaction{}, err
	}
	feeCap, err := readPreparedInt64(r)
	if err != nil {
		return preparedTransaction{}, err
	}
	canonical, err := readPreparedBytes(r, maxPreparedCanonicalBytes, "canonical transaction")
	if err != nil {
		return preparedTransaction{}, err
	}
	expectedTxID, err := readPreparedTxID(r, "expected txid")
	if err != nil {
		return preparedTransaction{}, err
	}
	inputCount, err := readPreparedUint32(r)
	if err != nil {
		return preparedTransaction{}, err
	}
	if inputCount == 0 || inputCount > maxPreparedInputs {
		return preparedTransaction{}, fmt.Errorf("btc prepared: invalid input count %d", inputCount)
	}
	inputs := make([]PreparedInput, int(inputCount))
	for i := range inputs {
		input := &inputs[i]
		input.PreviousTxID, err = readPreparedTxID(r, fmt.Sprintf("input %d previous txid", i))
		if err != nil {
			return preparedTransaction{}, err
		}
		input.PreviousVout, err = readPreparedUint32(r)
		if err != nil {
			return preparedTransaction{}, err
		}
		input.AmountSats, err = readPreparedInt64(r)
		if err != nil {
			return preparedTransaction{}, err
		}
		input.PreviousOutputScript, err = readPreparedBytes(r, maxPreparedScriptBytes, fmt.Sprintf("input %d previous-output script", i))
		if err != nil {
			return preparedTransaction{}, err
		}
		input.WitnessScript, err = readPreparedBytes(r, maxPreparedScriptBytes, fmt.Sprintf("input %d witness script", i))
		if err != nil {
			return preparedTransaction{}, err
		}
		input.Confirmations, err = readPreparedInt64(r)
		if err != nil {
			return preparedTransaction{}, err
		}
	}
	if r.Len() != 0 {
		return preparedTransaction{}, fmt.Errorf("btc prepared: %d trailing body bytes", r.Len())
	}

	decoded := preparedTransaction{
		formatVersion: version,
		operationKind: kind,
		authorization: authorization,
		network:       PreparedNetwork{Name: networkName, GenesisHash: genesisHash},
		vault:         vault,
		canonicalTx:   canonical,
		expectedTxID:  expectedTxID,
		inputs:        inputs,
		policy: PreparedPolicy{
			FormatVersion:     policyVersion,
			ConfirmationDepth: confirmationDepth,
			FeeCapSatPerVByte: feeCap,
		},
		integrityHash: gotHash,
	}
	if _, _, err := validatePreparedLocal(&decoded); err != nil {
		return preparedTransaction{}, err
	}
	return decoded, nil
}

func operationKindForMagic(magic string) (preparedOperationKind, error) {
	switch magic {
	case preparedWithdrawalMagic:
		return preparedOperationWithdrawal, nil
	case preparedRotationMagic:
		return preparedOperationRotation, nil
	case preparedConsolidationMagic:
		return preparedOperationConsolidate, nil
	default:
		return 0, fmt.Errorf("btc prepared: unknown operation type")
	}
}

// Prepare performs live withdrawal validation once and captures the exact
// previous outputs required for all later offline operations. It calls
// GetTxOut with includeMempool=true exactly once for each canonical input.
func (f *WithdrawalFinalizer) Prepare(ctx context.Context, canonical []byte, auth WithdrawalAuthorization) (*PreparedWithdrawal, error) {
	recipient, amount, err := f.parseOp(ctx, auth.Operation)
	if err != nil {
		return nil, err
	}
	tx, err := deserializeUnsignedCanonical(canonical)
	if err != nil {
		return nil, candidateError("btc prepare", err)
	}
	if err := validateWithdrawalOutputs(f, tx, recipient, amount, auth.WithdrawalID); err != nil {
		return nil, candidateError("btc prepare", err)
	}
	authHash, err := hashWithdrawalAuthorization(auth)
	if err != nil {
		return nil, candidateError("btc prepare", err)
	}
	transaction, err := f.prepareTransaction(ctx, canonical, tx, preparedOperationWithdrawal, authHash)
	if err != nil {
		return nil, fmt.Errorf("btc prepare: %w", err)
	}
	if _, err := f.validatePreparedAgainst(transaction, recipient, amount, auth.WithdrawalID); err != nil {
		return nil, candidateError("btc prepare", err)
	}
	return &PreparedWithdrawal{preparedTransaction: *transaction}, nil
}

// prepareTransactionFromCanonical captures prevouts only after an
// operation-specific validator has authorized the outputs. It is intentionally
// unexported so SDK consumers cannot accidentally treat ownership/fee checks as
// withdrawal authorization.
func (f *WithdrawalFinalizer) prepareTransactionFromCanonical(ctx context.Context, canonical []byte, kind preparedOperationKind, authorization [sha256.Size]byte) (*preparedTransaction, error) {
	tx, err := deserializeUnsignedCanonical(canonical)
	if err != nil {
		return nil, candidateError("btc prepare transaction", err)
	}
	prepared, err := f.prepareTransaction(ctx, canonical, tx, kind, authorization)
	if err != nil {
		return nil, fmt.Errorf("btc prepare transaction: %w", err)
	}
	return prepared, nil
}

func (f *WithdrawalFinalizer) prepareTransaction(ctx context.Context, canonical []byte, tx *wire.MsgTx, kind preparedOperationKind, authorization [sha256.Size]byte) (*preparedTransaction, error) {
	inputs := make([]PreparedInput, len(tx.TxIn))
	seen := make(map[wire.OutPoint]struct{}, len(tx.TxIn))
	for i, txIn := range tx.TxIn {
		if _, duplicate := seen[txIn.PreviousOutPoint]; duplicate {
			return nil, candidateError("btc prepare transaction", fmt.Errorf("duplicate input outpoint %s", txIn.PreviousOutPoint))
		}
		seen[txIn.PreviousOutPoint] = struct{}{}
		out, err := f.rpc.GetTxOut(ctx, txIn.PreviousOutPoint.Hash.String(), txIn.PreviousOutPoint.Index, true)
		if err != nil {
			return nil, fmt.Errorf("gettxout input %d: %w", i, err)
		}
		if out == nil {
			return nil, candidateError("btc prepare transaction", fmt.Errorf("input %d spent or unknown", i))
		}
		witnessScript, ok := f.resolveScript(out.ScriptPubKey)
		if !ok {
			return nil, candidateError("btc prepare transaction", fmt.Errorf("input %d not a vault output", i))
		}
		previousOutputScript, err := hex.DecodeString(out.ScriptPubKey)
		if err != nil {
			return nil, fmt.Errorf("input %d bad scriptPubKey %q: %w", i, out.ScriptPubKey, err)
		}
		inputs[i] = PreparedInput{
			PreviousTxID:         txIn.PreviousOutPoint.Hash.String(),
			PreviousVout:         txIn.PreviousOutPoint.Index,
			AmountSats:           out.AmountSats,
			PreviousOutputScript: append([]byte(nil), previousOutputScript...),
			WitnessScript:        append([]byte(nil), witnessScript...),
			Confirmations:        out.Confirmations,
		}
	}

	prepared := &preparedTransaction{
		formatVersion: PreparedWithdrawalFormatVersion,
		operationKind: kind,
		authorization: authorization,
		network:       PreparedNetwork{Name: f.net.Name, GenesisHash: f.net.GenesisHash.String()},
		vault:         f.vaultSnapshot(),
		canonicalTx:   append([]byte(nil), canonical...),
		expectedTxID:  tx.TxHash().String(),
		inputs:        inputs,
		policy: PreparedPolicy{
			FormatVersion:     PreparedPolicyFormatVersion,
			ConfirmationDepth: f.cfg.ConfirmationDepth,
			FeeCapSatPerVByte: f.cfg.FeeCapSatPerVByte,
		},
	}
	if err := sealPreparedTransaction(prepared); err != nil {
		return nil, err
	}
	if _, _, err := validatePreparedLocal(prepared); err != nil {
		return nil, candidateError("btc prepare transaction", err)
	}
	return prepared, nil
}

// ValidatePrepared rechecks a prepared context and its authorization entirely
// offline. As with the legacy BTC Validate method, deadline is deliberately not
// a transaction field because Bitcoin has no consensus transaction expiry.
func (f *WithdrawalFinalizer) ValidatePrepared(prepared *PreparedWithdrawal, auth WithdrawalAuthorization) error {
	recipient, amount, err := f.parseOp(context.Background(), auth.Operation)
	if err != nil {
		return err
	}
	transaction := withdrawalTransaction(prepared)
	authHash, err := hashWithdrawalAuthorization(auth)
	if err != nil {
		return err
	}
	if err := validateAuthorizationHash(transaction, preparedOperationWithdrawal, authHash); err != nil {
		return err
	}
	verifier, err := f.offlineVerifier(transaction)
	if err != nil {
		return err
	}
	_, err = verifier.validatePreparedAgainst(transaction, recipient, amount, auth.WithdrawalID)
	return err
}

// SignPrepared signs every input using only the previous outputs embedded in
// prepared. It performs no RPC calls.
func (f *WithdrawalFinalizer) SignPrepared(ctx context.Context, prepared *PreparedWithdrawal, auth WithdrawalAuthorization) ([]byte, error) {
	if err := f.ValidatePrepared(prepared, auth); err != nil {
		return nil, err
	}
	verifier, err := f.offlineVerifier(withdrawalTransaction(prepared))
	if err != nil {
		return nil, err
	}
	return verifier.signPrepared(ctx, withdrawalTransaction(prepared))
}

func (f *WithdrawalFinalizer) signPrepared(ctx context.Context, prepared *preparedTransaction) ([]byte, error) {
	tx, prevFetcher, err := validatePreparedLocal(prepared)
	if err != nil {
		return nil, fmt.Errorf("btc sign prepared: %w", err)
	}
	sigs := make([]string, len(tx.TxIn))
	for i := range tx.TxIn {
		digest, err := SighashAll(tx, i, prepared.inputs[i].WitnessScript, prepared.inputs[i].AmountSats, prevFetcher)
		if err != nil {
			return nil, fmt.Errorf("btc sign prepared: sighash input %d: %w", i, err)
		}
		der, err := f.signer.Sign(ctx, digest)
		if err != nil {
			return nil, fmt.Errorf("btc sign prepared: input %d: %w", i, err)
		}
		if _, err := parseCanonicalLowSDER(der); err != nil {
			return nil, fmt.Errorf("btc sign prepared: input %d signature: %w", i, err)
		}
		sigs[i] = hex.EncodeToString(append(append([]byte(nil), der...), byte(txscript.SigHashAll)))
	}
	return json.Marshal(SigShare{PubKey: hex.EncodeToString(f.signerPub), Sigs: sigs})
}

// VerifySharePrepared verifies the share's strict shape, signer membership,
// canonical DER+SIGHASH_ALL encoding, and every per-input ECDSA signature.
func (f *WithdrawalFinalizer) VerifySharePrepared(prepared *PreparedWithdrawal, auth WithdrawalAuthorization, share []byte) error {
	if err := f.ValidatePrepared(prepared, auth); err != nil {
		return err
	}
	verifier, err := f.offlineVerifier(withdrawalTransaction(prepared))
	if err != nil {
		return err
	}
	return verifier.verifySharePrepared(withdrawalTransaction(prepared), share)
}

func (f *WithdrawalFinalizer) verifySharePrepared(prepared *preparedTransaction, share []byte) error {
	tx, prevFetcher, err := validatePreparedLocal(prepared)
	if err != nil {
		return fmt.Errorf("btc verify prepared share: %w", err)
	}
	_, err = f.verifyPreparedShare(tx, prevFetcher, prepared, share)
	if err != nil {
		return candidateError("btc verify prepared share", err)
	}
	return nil
}

// FinalizePrepared filters invalid, duplicate, and unknown shares, chooses the
// threshold lowest signers by redeem-script key order, and assembles the exact
// signed transaction without RPC access.
func (f *WithdrawalFinalizer) FinalizePrepared(prepared *PreparedWithdrawal, auth WithdrawalAuthorization, shares [][]byte) ([]byte, string, error) {
	if err := f.ValidatePrepared(prepared, auth); err != nil {
		return nil, "", err
	}
	verifier, err := f.offlineVerifier(withdrawalTransaction(prepared))
	if err != nil {
		return nil, "", err
	}
	return verifier.finalizePrepared(withdrawalTransaction(prepared), shares)
}

func (f *WithdrawalFinalizer) finalizePrepared(prepared *preparedTransaction, shares [][]byte) ([]byte, string, error) {
	tx, prevFetcher, err := validatePreparedLocal(prepared)
	if err != nil {
		return nil, "", fmt.Errorf("btc finalize prepared: %w", err)
	}

	bySigner := make(map[string]verifiedPreparedShare, len(shares))
	for _, raw := range shares {
		verified, err := f.verifyPreparedShare(tx, prevFetcher, prepared, raw)
		if err != nil {
			continue
		}
		current, exists := bySigner[verified.pubKey]
		if !exists || comparePreparedSignatures(verified.signatures, current.signatures) < 0 {
			bySigner[verified.pubKey] = verified
		}
	}
	valid := make([]verifiedPreparedShare, 0, len(bySigner))
	for _, share := range bySigner {
		valid = append(valid, share)
	}
	sort.Slice(valid, func(i, j int) bool { return valid[i].position < valid[j].position })
	if len(valid) < f.threshold {
		return nil, "", fmt.Errorf("btc finalize prepared: %d valid unique shares, need %d", len(valid), f.threshold)
	}
	valid = valid[:f.threshold]
	for inputIndex := range tx.TxIn {
		ordered := make([][]byte, f.threshold)
		for signerIndex := range valid {
			ordered[signerIndex] = valid[signerIndex].signatures[inputIndex]
		}
		tx.TxIn[inputIndex].Witness = AssembleWitness(prepared.inputs[inputIndex].WitnessScript, ordered)
	}
	raw, err := serializeTx(tx)
	if err != nil {
		return nil, "", fmt.Errorf("btc finalize prepared: %w", err)
	}
	if txid := tx.TxHash().String(); txid != prepared.expectedTxID {
		return nil, "", fmt.Errorf("btc finalize prepared: finalized txid %s != expected %s", txid, prepared.expectedTxID)
	}
	return raw, prepared.expectedTxID, nil
}

// VerifyFinalizedPrepared verifies that raw is the exact witness-bearing form of
// prepared and executes every input script against the persisted prevouts. It
// performs no RPC calls and is suitable for corruption checks during recovery.
func (f *WithdrawalFinalizer) VerifyFinalizedPrepared(prepared *PreparedWithdrawal, auth WithdrawalAuthorization, raw []byte) error {
	if err := f.ValidatePrepared(prepared, auth); err != nil {
		return err
	}
	verifier, err := f.offlineVerifier(withdrawalTransaction(prepared))
	if err != nil {
		return err
	}
	return verifier.verifyFinalizedPrepared(withdrawalTransaction(prepared), raw)
}

func (f *WithdrawalFinalizer) verifyFinalizedPrepared(prepared *preparedTransaction, raw []byte) error {
	unsigned, prevFetcher, err := validatePreparedLocal(prepared)
	if err != nil {
		return fmt.Errorf("btc verify finalized prepared: %w", err)
	}
	finalTx, err := deserializeExactTx(raw)
	if err != nil {
		return fmt.Errorf("btc verify finalized prepared: %w", err)
	}
	if finalTx.TxHash().String() != prepared.expectedTxID {
		return fmt.Errorf("btc verify finalized prepared: txid %s != expected %s", finalTx.TxHash(), prepared.expectedTxID)
	}
	var finalBody bytes.Buffer
	if err := finalTx.SerializeNoWitness(&finalBody); err != nil {
		return fmt.Errorf("btc verify finalized prepared: serialize body: %w", err)
	}
	var unsignedBody bytes.Buffer
	if err := unsigned.SerializeNoWitness(&unsignedBody); err != nil {
		return fmt.Errorf("btc verify finalized prepared: serialize canonical body: %w", err)
	}
	if !bytes.Equal(finalBody.Bytes(), unsignedBody.Bytes()) {
		return fmt.Errorf("btc verify finalized prepared: non-witness body differs from prepared canonical")
	}
	sigHashes := txscript.NewTxSigHashes(finalTx, prevFetcher)
	for i, input := range prepared.inputs {
		engine, err := txscript.NewEngine(input.PreviousOutputScript, finalTx, i, txscript.StandardVerifyFlags, nil, sigHashes, input.AmountSats, prevFetcher)
		if err != nil {
			return fmt.Errorf("btc verify finalized prepared: input %d engine: %w", i, err)
		}
		if err := engine.Execute(); err != nil {
			return fmt.Errorf("btc verify finalized prepared: input %d witness: %w", i, err)
		}
	}
	return nil
}

// BroadcastPrepared cryptographically verifies the exact finalized withdrawal
// immediately before broadcasting it.
func (f *WithdrawalFinalizer) BroadcastPrepared(ctx context.Context, prepared *PreparedWithdrawal, auth WithdrawalAuthorization, raw []byte) (string, error) {
	if err := f.VerifyFinalizedPrepared(prepared, auth, raw); err != nil {
		return "", err
	}
	return f.sendVerifiedPrepared(ctx, raw, prepared.expectedTxID)
}

func (f *WithdrawalFinalizer) broadcastPrepared(ctx context.Context, prepared *preparedTransaction, raw []byte) (string, error) {
	if err := f.verifyFinalizedPrepared(prepared, raw); err != nil {
		return "", fmt.Errorf("btc broadcast prepared: %w", err)
	}
	return f.sendVerifiedPrepared(ctx, raw, prepared.expectedTxID)
}

func (f *WithdrawalFinalizer) sendVerifiedPrepared(ctx context.Context, raw []byte, expectedTxID string) (string, error) {
	returnedTxID, err := f.rpc.SendRawTransaction(ctx, hex.EncodeToString(raw))
	if err == nil {
		if !equalCanonicalTxIDs(returnedTxID, expectedTxID) {
			return "", fmt.Errorf("btc broadcast prepared: sendrawtransaction returned txid %q, expected %s", returnedTxID, expectedTxID)
		}
		return expectedTxID, nil
	}
	lookupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), broadcastLookupTimeout)
	defer cancel()
	if exactTxIDExists(lookupCtx, f.rpc, expectedTxID) {
		return expectedTxID, nil
	}
	return "", fmt.Errorf("btc broadcast prepared: sendrawtransaction: %w", err)
}

type verifiedPreparedShare struct {
	pubKey     string
	position   int
	signatures [][]byte
}

func (f *WithdrawalFinalizer) verifyPreparedShare(tx *wire.MsgTx, prevFetcher txscript.PrevOutputFetcher, prepared *preparedTransaction, raw []byte) (verifiedPreparedShare, error) {
	var share SigShare
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&share); err != nil {
		return verifiedPreparedShare{}, fmt.Errorf("btc verify prepared share: decode: %w", err)
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return verifiedPreparedShare{}, fmt.Errorf("btc verify prepared share: decode: %w", err)
	}
	pubKeyBytes, err := hex.DecodeString(share.PubKey)
	if err != nil || len(pubKeyBytes) != 33 || share.PubKey != hex.EncodeToString(pubKeyBytes) {
		return verifiedPreparedShare{}, fmt.Errorf("btc verify prepared share: signer pubkey is not canonical compressed hex")
	}
	pubKey, err := btcec.ParsePubKey(pubKeyBytes)
	if err != nil || !bytes.Equal(pubKey.SerializeCompressed(), pubKeyBytes) {
		return verifiedPreparedShare{}, fmt.Errorf("btc verify prepared share: invalid compressed signer pubkey")
	}
	position, member := f.pubkeyPos[share.PubKey]
	if !member {
		return verifiedPreparedShare{}, fmt.Errorf("btc verify prepared share: unknown signer %s", share.PubKey)
	}
	if len(share.Sigs) != len(tx.TxIn) {
		return verifiedPreparedShare{}, fmt.Errorf("btc verify prepared share: share has %d signatures, transaction has %d inputs", len(share.Sigs), len(tx.TxIn))
	}

	verified := verifiedPreparedShare{pubKey: share.PubKey, position: position, signatures: make([][]byte, len(share.Sigs))}
	for i, encoded := range share.Sigs {
		sigBytes, err := hex.DecodeString(encoded)
		if err != nil || encoded != hex.EncodeToString(sigBytes) {
			return verifiedPreparedShare{}, fmt.Errorf("btc verify prepared share: input %d signature is not canonical hex", i)
		}
		if len(sigBytes) < 2 || sigBytes[len(sigBytes)-1] != byte(txscript.SigHashAll) {
			return verifiedPreparedShare{}, fmt.Errorf("btc verify prepared share: input %d signature missing SIGHASH_ALL", i)
		}
		der := sigBytes[:len(sigBytes)-1]
		sig, err := parseCanonicalLowSDER(der)
		if err != nil {
			return verifiedPreparedShare{}, fmt.Errorf("btc verify prepared share: input %d signature: %w", i, err)
		}
		digest, err := SighashAll(tx, i, prepared.inputs[i].WitnessScript, prepared.inputs[i].AmountSats, prevFetcher)
		if err != nil {
			return verifiedPreparedShare{}, fmt.Errorf("btc verify prepared share: input %d sighash: %w", i, err)
		}
		if !sig.Verify(digest, pubKey) {
			return verifiedPreparedShare{}, fmt.Errorf("btc verify prepared share: input %d signature is invalid", i)
		}
		verified.signatures[i] = append([]byte(nil), sigBytes...)
	}
	return verified, nil
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("multiple JSON values")
		}
		return err
	}
	return nil
}

func (f *WithdrawalFinalizer) validatePreparedAgainst(prepared *preparedTransaction, recipient btcutil.Address, amount int64, withdrawalID [32]byte) (*wire.MsgTx, error) {
	tx, _, err := validatePreparedLocal(prepared)
	if err != nil {
		return nil, fmt.Errorf("btc validate prepared: %w", err)
	}
	if err := validateWithdrawalOutputs(f, tx, recipient, amount, withdrawalID); err != nil {
		return nil, fmt.Errorf("btc validate prepared: %w", err)
	}
	return tx, nil
}

func validatePreparedLocal(prepared *preparedTransaction) (*wire.MsgTx, txscript.PrevOutputFetcher, error) {
	tx, err := validatePreparedEncoding(prepared)
	if err != nil {
		return nil, nil, err
	}
	if err := validateFixedTxFields(tx); err != nil {
		return nil, nil, err
	}
	prevFetcher := txscript.NewMultiPrevOutFetcher(nil)
	var totalIn int64
	for i, input := range prepared.inputs {
		if prepared.policy.ConfirmationDepth > math.MaxInt64 || input.Confirmations < int64(prepared.policy.ConfirmationDepth) {
			return nil, nil, fmt.Errorf("input %d has %d confirmations, need %d", i, input.Confirmations, prepared.policy.ConfirmationDepth)
		}
		if totalIn > math.MaxInt64-input.AmountSats {
			return nil, nil, fmt.Errorf("input value total overflows int64")
		}
		totalIn += input.AmountSats
		prevFetcher.AddPrevOut(tx.TxIn[i].PreviousOutPoint, wire.NewTxOut(input.AmountSats, append([]byte(nil), input.PreviousOutputScript...)))
	}
	var totalOut int64
	for i, output := range tx.TxOut {
		if output.Value < 0 {
			return nil, nil, fmt.Errorf("output %d has negative value %d", i, output.Value)
		}
		if totalOut > math.MaxInt64-output.Value {
			return nil, nil, fmt.Errorf("output value total overflows int64")
		}
		totalOut += output.Value
	}
	fee := totalIn - totalOut
	if fee < 0 {
		return nil, nil, fmt.Errorf("outputs exceed inputs (fee %d)", fee)
	}
	if prepared.policy.FeeCapSatPerVByte > 0 {
		vsize := int64(txOverheadVBytes) + int64(len(tx.TxIn))*p2wshInputVBytes + int64(len(tx.TxOut))*outputVBytes
		if prepared.policy.FeeCapSatPerVByte <= math.MaxInt64/vsize && fee > vsize*prepared.policy.FeeCapSatPerVByte {
			return nil, nil, fmt.Errorf("fee %d exceeds ceiling %d", fee, vsize*prepared.policy.FeeCapSatPerVByte)
		}
	}
	return tx, prevFetcher, nil
}

func validatePreparedEncoding(prepared *preparedTransaction) (*wire.MsgTx, error) {
	if prepared == nil {
		return nil, fmt.Errorf("nil prepared withdrawal")
	}
	if prepared.formatVersion != PreparedWithdrawalFormatVersion {
		return nil, fmt.Errorf("unsupported format version %d", prepared.formatVersion)
	}
	if prepared.operationKind < preparedOperationWithdrawal || prepared.operationKind > preparedOperationConsolidate {
		return nil, fmt.Errorf("invalid operation kind %d", prepared.operationKind)
	}
	if prepared.authorization == ([sha256.Size]byte{}) {
		return nil, fmt.Errorf("missing authorization hash")
	}
	if err := validatePreparedNetwork(prepared.network); err != nil {
		return nil, err
	}
	if err := validatePreparedVault(prepared.vault, prepared.inputs); err != nil {
		return nil, err
	}
	body, err := marshalPreparedTransactionBody(prepared)
	if err != nil {
		return nil, err
	}
	if sha256.Sum256(body) != prepared.integrityHash {
		return nil, fmt.Errorf("integrity hash mismatch")
	}
	tx, err := deserializeUnsignedCanonical(prepared.canonicalTx)
	if err != nil {
		return nil, err
	}
	if err := validateCanonicalTxID(prepared.expectedTxID, "expected txid"); err != nil {
		return nil, err
	}
	if tx.TxHash().String() != prepared.expectedTxID {
		return nil, fmt.Errorf("canonical transaction txid %s != expected %s", tx.TxHash(), prepared.expectedTxID)
	}
	if len(prepared.inputs) != len(tx.TxIn) {
		return nil, fmt.Errorf("prepared input count %d != transaction input count %d", len(prepared.inputs), len(tx.TxIn))
	}
	seen := make(map[wire.OutPoint]struct{}, len(tx.TxIn))
	for i, input := range prepared.inputs {
		txIn := tx.TxIn[i]
		if input.PreviousTxID != txIn.PreviousOutPoint.Hash.String() || input.PreviousVout != txIn.PreviousOutPoint.Index {
			return nil, fmt.Errorf("prepared input %d outpoint does not match transaction order", i)
		}
		if _, duplicate := seen[txIn.PreviousOutPoint]; duplicate {
			return nil, fmt.Errorf("duplicate input outpoint %s", txIn.PreviousOutPoint)
		}
		seen[txIn.PreviousOutPoint] = struct{}{}
		if input.AmountSats <= 0 {
			return nil, fmt.Errorf("prepared input %d has non-positive value %d", i, input.AmountSats)
		}
		if input.Confirmations < 0 {
			return nil, fmt.Errorf("prepared input %d has negative confirmations %d", i, input.Confirmations)
		}
		wantPreviousScript, err := p2wshScript(input.WitnessScript)
		if err != nil {
			return nil, fmt.Errorf("prepared input %d: %w", i, err)
		}
		if !bytes.Equal(input.PreviousOutputScript, wantPreviousScript) {
			return nil, fmt.Errorf("prepared input %d previous-output script is not P2WSH of its witness script", i)
		}
	}
	if err := validatePreparedPolicy(prepared.policy); err != nil {
		return nil, err
	}
	return tx, nil
}

func deserializeUnsignedCanonical(raw []byte) (*wire.MsgTx, error) {
	tx, err := deserializeExactTx(raw)
	if err != nil {
		return nil, err
	}
	if len(tx.TxIn) == 0 {
		return nil, fmt.Errorf("transaction has no inputs")
	}
	for i, input := range tx.TxIn {
		if len(input.SignatureScript) != 0 || len(input.Witness) != 0 {
			return nil, fmt.Errorf("canonical transaction input %d is not unsigned", i)
		}
	}
	return tx, nil
}

func deserializeExactTx(raw []byte) (*wire.MsgTx, error) {
	if len(raw) == 0 || len(raw) > maxPreparedCanonicalBytes {
		return nil, fmt.Errorf("invalid transaction length %d", len(raw))
	}
	tx, err := deserializeTx(raw)
	if err != nil {
		return nil, err
	}
	encoded, err := serializeTx(tx)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(encoded, raw) {
		return nil, fmt.Errorf("transaction bytes are not an exact canonical serialization")
	}
	return tx, nil
}

func validateWithdrawalOutputs(f *WithdrawalFinalizer, tx *wire.MsgTx, recipient btcutil.Address, amount int64, withdrawalID [32]byte) error {
	if err := validateFixedTxFields(tx); err != nil {
		return err
	}
	if n := len(tx.TxOut); n != 2 && n != 3 {
		return fmt.Errorf("expected 2 or 3 outputs, got %d", n)
	}
	recipientScript, err := txscript.PayToAddrScript(recipient)
	if err != nil {
		return fmt.Errorf("recipient script: %w", err)
	}
	if !bytes.Equal(tx.TxOut[0].PkScript, recipientScript) {
		return fmt.Errorf("output 0 not the op recipient")
	}
	if tx.TxOut[0].Value != amount {
		return fmt.Errorf("output 0 value %d != op amount %d", tx.TxOut[0].Value, amount)
	}
	wantMarker, err := txscript.NullDataScript(withdrawalID[:])
	if err != nil {
		return fmt.Errorf("opreturn script: %w", err)
	}
	last := tx.TxOut[len(tx.TxOut)-1]
	if last.Value != 0 || !bytes.Equal(last.PkScript, wantMarker) {
		return fmt.Errorf("final output is not OP_RETURN <withdrawalID>")
	}
	if len(tx.TxOut) == 3 {
		change := tx.TxOut[1]
		if !bytes.Equal(change.PkScript, f.vaultScript) {
			return fmt.Errorf("change output not paid to the vault")
		}
		if change.Value < dustThresholdSats {
			return fmt.Errorf("change output %d below dust", change.Value)
		}
	}
	return nil
}

func p2wshScript(witnessScript []byte) ([]byte, error) {
	if len(witnessScript) == 0 || len(witnessScript) > maxPreparedScriptBytes {
		return nil, fmt.Errorf("invalid witness script length %d", len(witnessScript))
	}
	hash := sha256.Sum256(witnessScript)
	return txscript.NewScriptBuilder().AddOp(txscript.OP_0).AddData(hash[:]).Script()
}

func sealPreparedTransaction(prepared *preparedTransaction) error {
	body, err := marshalPreparedTransactionBody(prepared)
	if err != nil {
		return fmt.Errorf("btc prepared: seal: %w", err)
	}
	prepared.integrityHash = sha256.Sum256(body)
	return nil
}

func marshalPreparedTransactionBody(prepared *preparedTransaction) ([]byte, error) {
	if prepared == nil {
		return nil, fmt.Errorf("btc prepared: nil prepared withdrawal")
	}
	if prepared.formatVersion != PreparedWithdrawalFormatVersion {
		return nil, fmt.Errorf("btc prepared: unsupported format version %d", prepared.formatVersion)
	}
	if err := validatePreparedPolicy(prepared.policy); err != nil {
		return nil, fmt.Errorf("btc prepared: %w", err)
	}
	if len(prepared.canonicalTx) == 0 || len(prepared.canonicalTx) > maxPreparedCanonicalBytes {
		return nil, fmt.Errorf("btc prepared: invalid canonical transaction length %d", len(prepared.canonicalTx))
	}
	if err := validateCanonicalTxID(prepared.expectedTxID, "expected txid"); err != nil {
		return nil, fmt.Errorf("btc prepared: %w", err)
	}
	if len(prepared.inputs) == 0 || len(prepared.inputs) > maxPreparedInputs {
		return nil, fmt.Errorf("btc prepared: invalid input count %d", len(prepared.inputs))
	}

	var body bytes.Buffer
	body.Grow(len(prepared.canonicalTx) + len(prepared.inputs)*128)
	body.WriteString(preparedWithdrawalMagic)
	writePreparedUint16(&body, prepared.formatVersion)
	body.WriteByte(byte(prepared.operationKind))
	body.Write(prepared.authorization[:])
	writePreparedString(&body, prepared.network.Name)
	writePreparedString(&body, prepared.network.GenesisHash)
	if err := writePreparedVault(&body, prepared.vault); err != nil {
		return nil, err
	}
	writePreparedUint16(&body, prepared.policy.FormatVersion)
	writePreparedUint64(&body, prepared.policy.ConfirmationDepth)
	writePreparedInt64(&body, prepared.policy.FeeCapSatPerVByte)
	writePreparedBytes(&body, prepared.canonicalTx)
	if err := writePreparedTxID(&body, prepared.expectedTxID); err != nil {
		return nil, err
	}
	writePreparedUint32(&body, uint32(len(prepared.inputs)))
	for i, input := range prepared.inputs {
		if err := validateCanonicalTxID(input.PreviousTxID, fmt.Sprintf("input %d previous txid", i)); err != nil {
			return nil, fmt.Errorf("btc prepared: %w", err)
		}
		if input.AmountSats <= 0 {
			return nil, fmt.Errorf("btc prepared: input %d has non-positive amount %d", i, input.AmountSats)
		}
		if input.Confirmations < 0 {
			return nil, fmt.Errorf("btc prepared: input %d has negative confirmations %d", i, input.Confirmations)
		}
		if len(input.PreviousOutputScript) == 0 || len(input.PreviousOutputScript) > maxPreparedScriptBytes {
			return nil, fmt.Errorf("btc prepared: input %d invalid previous-output script length %d", i, len(input.PreviousOutputScript))
		}
		if len(input.WitnessScript) == 0 || len(input.WitnessScript) > maxPreparedScriptBytes {
			return nil, fmt.Errorf("btc prepared: input %d invalid witness script length %d", i, len(input.WitnessScript))
		}
		if err := writePreparedTxID(&body, input.PreviousTxID); err != nil {
			return nil, err
		}
		writePreparedUint32(&body, input.PreviousVout)
		writePreparedInt64(&body, input.AmountSats)
		writePreparedBytes(&body, input.PreviousOutputScript)
		writePreparedBytes(&body, input.WitnessScript)
		writePreparedInt64(&body, input.Confirmations)
	}
	if body.Len()+preparedWithdrawalHashBytes > maxPreparedBinaryBytes {
		return nil, fmt.Errorf("btc prepared: binary length exceeds limit %d", maxPreparedBinaryBytes)
	}
	return body.Bytes(), nil
}

func (f *WithdrawalFinalizer) vaultSnapshot() PreparedVault {
	f.mu.RLock()
	defer f.mu.RUnlock()
	scripts := make([][]byte, 0, len(f.spendScripts))
	for _, script := range f.spendScripts {
		scripts = append(scripts, append([]byte(nil), script...))
	}
	sort.Slice(scripts, func(i, j int) bool { return bytes.Compare(scripts[i], scripts[j]) < 0 })
	keys := cloneByteSlices(f.pubkeys)
	sort.Slice(keys, func(i, j int) bool { return bytes.Compare(keys[i], keys[j]) < 0 })
	v := PreparedVault{PubKeys: keys, Threshold: f.threshold, AllowedScripts: scripts}
	v.Fingerprint = preparedVaultFingerprint(v)
	return v
}

// VaultFingerprint returns the canonical fingerprint of the finalizer's
// current authorizing vault and registered witness-script set.
func (f *WithdrawalFinalizer) VaultFingerprint() [sha256.Size]byte {
	return f.vaultSnapshot().Fingerprint
}

func (f *WithdrawalFinalizer) offlineVerifier(prepared *preparedTransaction) (*WithdrawalFinalizer, error) {
	if prepared == nil {
		return nil, fmt.Errorf("btc prepared: nil transaction")
	}
	if prepared.network.Name != f.net.Name || prepared.network.GenesisHash != f.net.GenesisHash.String() {
		return nil, fmt.Errorf("btc prepared: network snapshot mismatch")
	}
	verifier, err := NewWithdrawalFinalizer(f.net, f.rpc, f.signer, cloneByteSlices(prepared.vault.PubKeys), prepared.vault.Threshold, Config{
		ConfirmationDepth: prepared.policy.ConfirmationDepth,
		FeeCapSatPerVByte: prepared.policy.FeeCapSatPerVByte,
	}, f.assets)
	if err != nil {
		return nil, fmt.Errorf("btc prepared: restore vault: %w", err)
	}
	verifier.mu.Lock()
	verifier.spendScripts = make(map[string][]byte, len(prepared.vault.AllowedScripts))
	for _, witness := range prepared.vault.AllowedScripts {
		pk, err := p2wshScript(witness)
		if err != nil {
			verifier.mu.Unlock()
			return nil, err
		}
		verifier.spendScripts[hex.EncodeToString(pk)] = append([]byte(nil), witness...)
	}
	verifier.mu.Unlock()
	return verifier, nil
}

func validatePreparedNetwork(network PreparedNetwork) error {
	if network.Name == "" {
		return fmt.Errorf("missing network name")
	}
	return validateCanonicalTxID(network.GenesisHash, "network genesis hash")
}

func validatePreparedVault(v PreparedVault, inputs []PreparedInput) error {
	if v.Threshold <= 0 || v.Threshold > len(v.PubKeys) {
		return fmt.Errorf("invalid vault threshold %d", v.Threshold)
	}
	if len(v.AllowedScripts) == 0 {
		return fmt.Errorf("vault has no allowed witness scripts")
	}
	if preparedVaultFingerprint(v) != v.Fingerprint {
		return fmt.Errorf("vault fingerprint mismatch")
	}
	allowed := make(map[string]struct{}, len(v.AllowedScripts))
	for _, script := range v.AllowedScripts {
		allowed[hex.EncodeToString(script)] = struct{}{}
	}
	for i, input := range inputs {
		if _, ok := allowed[hex.EncodeToString(input.WitnessScript)]; !ok {
			return fmt.Errorf("input %d witness script not authorized by vault snapshot", i)
		}
	}
	return nil
}

func preparedVaultFingerprint(v PreparedVault) [sha256.Size]byte {
	var b bytes.Buffer
	b.WriteString("clearnet-sdk/btc/vault/v1")
	writePreparedUint32(&b, uint32(v.Threshold))
	writePreparedUint32(&b, uint32(len(v.PubKeys)))
	for _, key := range v.PubKeys {
		writePreparedBytes(&b, key)
	}
	writePreparedUint32(&b, uint32(len(v.AllowedScripts)))
	for _, script := range v.AllowedScripts {
		writePreparedBytes(&b, script)
	}
	return sha256.Sum256(b.Bytes())
}

func writePreparedVault(w *bytes.Buffer, v PreparedVault) error {
	if err := validatePreparedVault(v, nil); err != nil {
		return fmt.Errorf("btc prepared: %w", err)
	}
	writePreparedUint32(w, uint32(v.Threshold))
	writePreparedUint32(w, uint32(len(v.PubKeys)))
	for _, key := range v.PubKeys {
		writePreparedBytes(w, key)
	}
	writePreparedUint32(w, uint32(len(v.AllowedScripts)))
	for _, script := range v.AllowedScripts {
		writePreparedBytes(w, script)
	}
	w.Write(v.Fingerprint[:])
	return nil
}

func readPreparedVault(r *bytes.Reader) (PreparedVault, error) {
	threshold, err := readPreparedUint32(r)
	if err != nil {
		return PreparedVault{}, err
	}
	keyCount, err := readPreparedUint32(r)
	if err != nil {
		return PreparedVault{}, err
	}
	if keyCount == 0 || keyCount > 1000 {
		return PreparedVault{}, fmt.Errorf("btc prepared: invalid vault key count %d", keyCount)
	}
	v := PreparedVault{Threshold: int(threshold), PubKeys: make([][]byte, keyCount)}
	for i := range v.PubKeys {
		v.PubKeys[i], err = readPreparedBytes(r, 65, fmt.Sprintf("vault key %d", i))
		if err != nil {
			return PreparedVault{}, err
		}
	}
	scriptCount, err := readPreparedUint32(r)
	if err != nil {
		return PreparedVault{}, err
	}
	if scriptCount == 0 || scriptCount > maxPreparedInputs {
		return PreparedVault{}, fmt.Errorf("btc prepared: invalid allowed script count %d", scriptCount)
	}
	v.AllowedScripts = make([][]byte, scriptCount)
	for i := range v.AllowedScripts {
		v.AllowedScripts[i], err = readPreparedBytes(r, maxPreparedScriptBytes, fmt.Sprintf("allowed script %d", i))
		if err != nil {
			return PreparedVault{}, err
		}
	}
	if _, err := io.ReadFull(r, v.Fingerprint[:]); err != nil {
		return PreparedVault{}, fmt.Errorf("btc prepared: read vault fingerprint: %w", err)
	}
	if err := validatePreparedVault(v, nil); err != nil {
		return PreparedVault{}, fmt.Errorf("btc prepared: %w", err)
	}
	return v, nil
}

func cloneByteSlices(in [][]byte) [][]byte {
	out := make([][]byte, len(in))
	for i := range in {
		out[i] = append([]byte(nil), in[i]...)
	}
	return out
}

func writePreparedString(w *bytes.Buffer, s string) { writePreparedBytes(w, []byte(s)) }
func readPreparedString(r *bytes.Reader, max uint32, field string) (string, error) {
	b, err := readPreparedBytes(r, max, field)
	return string(b), err
}

func validateAuthorizationHash(prepared *preparedTransaction, kind preparedOperationKind, want [sha256.Size]byte) error {
	if prepared == nil {
		return fmt.Errorf("btc prepared: nil transaction")
	}
	if prepared.operationKind != kind {
		return fmt.Errorf("btc prepared: wrong operation type")
	}
	if prepared.authorization != want {
		return fmt.Errorf("btc prepared: authorization mismatch")
	}
	return nil
}

func hashWithdrawalAuthorization(auth WithdrawalAuthorization) ([sha256.Size]byte, error) {
	if auth.Operation == nil {
		return [sha256.Size]byte{}, fmt.Errorf("btc withdrawal authorization: operation is required")
	}
	var b bytes.Buffer
	b.WriteString("clearnet-sdk/btc/withdrawal-authorization/v1")
	writePreparedString(&b, string(auth.Operation.AssetURI))
	writePreparedString(&b, auth.Operation.Amount.String())
	writePreparedString(&b, auth.Operation.Recipient)
	writePreparedBytes(&b, auth.Operation.UserSignature)
	b.Write(auth.WithdrawalID[:])
	writePreparedInt64(&b, auth.Deadline)
	return sha256.Sum256(b.Bytes()), nil
}

// HashWithdrawalAuthorization returns the exact domain-separated hash embedded
// in prepared withdrawals.
func HashWithdrawalAuthorization(auth WithdrawalAuthorization) ([sha256.Size]byte, error) {
	return hashWithdrawalAuthorization(auth)
}

func hashRotationAuthorization(auth RotationAuthorization) ([sha256.Size]byte, error) {
	keys, err := parseVaultPubkeys(auth.Signers)
	if err != nil {
		return [sha256.Size]byte{}, err
	}
	if _, err := RedeemScript(auth.Threshold, keys); err != nil {
		return [sha256.Size]byte{}, err
	}
	sort.Slice(keys, func(i, j int) bool { return bytes.Compare(keys[i], keys[j]) < 0 })
	var b bytes.Buffer
	b.WriteString("clearnet-sdk/btc/rotation-authorization/v1")
	b.Write(auth.OperationID[:])
	writePreparedUint32(&b, uint32(auth.Threshold))
	writePreparedUint32(&b, uint32(len(keys)))
	for _, key := range keys {
		writePreparedBytes(&b, key)
	}
	return sha256.Sum256(b.Bytes()), nil
}

// HashRotationAuthorization returns the exact authorization hash embedded in
// prepared rotations.
func HashRotationAuthorization(auth RotationAuthorization) ([sha256.Size]byte, error) {
	return hashRotationAuthorization(auth)
}

func hashConsolidationAuthorization(auth ConsolidationAuthorization) ([sha256.Size]byte, error) {
	if auth.TargetVaultIdentity == "" {
		return [sha256.Size]byte{}, fmt.Errorf("btc consolidation authorization: target vault identity is required")
	}
	var b bytes.Buffer
	b.WriteString("clearnet-sdk/btc/consolidation-authorization/v1")
	b.Write(auth.OperationID[:])
	writePreparedString(&b, auth.TargetVaultIdentity)
	return sha256.Sum256(b.Bytes()), nil
}

// HashConsolidationAuthorization returns the exact authorization hash embedded
// in prepared consolidations.
func HashConsolidationAuthorization(auth ConsolidationAuthorization) ([sha256.Size]byte, error) {
	return hashConsolidationAuthorization(auth)
}

func validatePreparedPolicy(policy PreparedPolicy) error {
	if policy.FormatVersion != PreparedPolicyFormatVersion {
		return fmt.Errorf("unsupported policy format version %d", policy.FormatVersion)
	}
	if policy.ConfirmationDepth > math.MaxInt64 {
		return fmt.Errorf("confirmation depth %d exceeds int64", policy.ConfirmationDepth)
	}
	if policy.FeeCapSatPerVByte < 0 {
		return fmt.Errorf("negative fee cap %d", policy.FeeCapSatPerVByte)
	}
	return nil
}

func clonePreparedInputs(inputs []PreparedInput) []PreparedInput {
	cloned := make([]PreparedInput, len(inputs))
	for i := range inputs {
		cloned[i] = inputs[i]
		cloned[i].PreviousOutputScript = append([]byte(nil), inputs[i].PreviousOutputScript...)
		cloned[i].WitnessScript = append([]byte(nil), inputs[i].WitnessScript...)
	}
	return cloned
}

func candidateError(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", operation, errors.Join(ErrCandidateRejected, err))
}

func validateCanonicalTxID(txid, field string) error {
	if len(txid) != chainhash.MaxHashStringSize || txid != strings.ToLower(txid) {
		return fmt.Errorf("%s is not canonical 64-character lowercase hex", field)
	}
	hash, err := chainhash.NewHashFromStr(txid)
	if err != nil || hash.String() != txid {
		return fmt.Errorf("%s is invalid", field)
	}
	return nil
}

func equalCanonicalTxIDs(got, expected string) bool {
	if validateCanonicalTxID(got, "returned txid") != nil {
		return false
	}
	return got == expected
}

func exactTxIDExists(ctx context.Context, rpc RPC, expectedTxID string) bool {
	raw, err := rpc.GetRawTransaction(ctx, expectedTxID)
	return err == nil && raw != nil && equalCanonicalTxIDs(raw.TxID, expectedTxID)
}

func comparePreparedSignatures(a, b [][]byte) int {
	for i := 0; i < len(a) && i < len(b); i++ {
		if cmp := bytes.Compare(a[i], b[i]); cmp != 0 {
			return cmp
		}
	}
	return len(a) - len(b)
}

func parseCanonicalLowSDER(der []byte) (*btcecdsa.Signature, error) {
	sig, err := btcecdsa.ParseDERSignature(der)
	if err != nil {
		return nil, err
	}
	s := sig.S()
	if s.IsOverHalfOrder() {
		return nil, fmt.Errorf("non-canonical high-S signature")
	}
	if !bytes.Equal(sig.Serialize(), der) {
		return nil, fmt.Errorf("non-canonical DER encoding")
	}
	return sig, nil
}

func writePreparedTxID(w *bytes.Buffer, txid string) error {
	decoded, err := hex.DecodeString(txid)
	if err != nil || len(decoded) != preparedWithdrawalTxIDBytes {
		return fmt.Errorf("btc prepared: invalid txid %q", txid)
	}
	w.Write(decoded)
	return nil
}

func readPreparedTxID(r *bytes.Reader, field string) (string, error) {
	b := make([]byte, preparedWithdrawalTxIDBytes)
	if _, err := io.ReadFull(r, b); err != nil {
		return "", fmt.Errorf("btc prepared: read %s: %w", field, err)
	}
	return hex.EncodeToString(b), nil
}

func writePreparedBytes(w *bytes.Buffer, value []byte) {
	writePreparedUint32(w, uint32(len(value)))
	w.Write(value)
}

func readPreparedBytes(r *bytes.Reader, max uint32, field string) ([]byte, error) {
	length, err := readPreparedUint32(r)
	if err != nil {
		return nil, err
	}
	if length == 0 || length > max || uint64(length) > uint64(r.Len()) {
		return nil, fmt.Errorf("btc prepared: invalid %s length %d", field, length)
	}
	value := make([]byte, int(length))
	if _, err := io.ReadFull(r, value); err != nil {
		return nil, fmt.Errorf("btc prepared: read %s: %w", field, err)
	}
	return value, nil
}

func writePreparedUint16(w *bytes.Buffer, value uint16) {
	var b [2]byte
	binary.BigEndian.PutUint16(b[:], value)
	w.Write(b[:])
}

func writePreparedUint32(w *bytes.Buffer, value uint32) {
	var b [4]byte
	binary.BigEndian.PutUint32(b[:], value)
	w.Write(b[:])
}

func writePreparedUint64(w *bytes.Buffer, value uint64) {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], value)
	w.Write(b[:])
}

func writePreparedInt64(w *bytes.Buffer, value int64) {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], uint64(value))
	w.Write(b[:])
}

func readPreparedUint16(r *bytes.Reader) (uint16, error) {
	var b [2]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		return 0, fmt.Errorf("btc prepared: read uint16: %w", err)
	}
	return binary.BigEndian.Uint16(b[:]), nil
}

func readPreparedUint32(r *bytes.Reader) (uint32, error) {
	var b [4]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		return 0, fmt.Errorf("btc prepared: read uint32: %w", err)
	}
	return binary.BigEndian.Uint32(b[:]), nil
}

func readPreparedUint64(r *bytes.Reader) (uint64, error) {
	var b [8]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		return 0, fmt.Errorf("btc prepared: read uint64: %w", err)
	}
	return binary.BigEndian.Uint64(b[:]), nil
}

func readPreparedInt64(r *bytes.Reader) (int64, error) {
	var b [8]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		return 0, fmt.Errorf("btc prepared: read int64: %w", err)
	}
	return int64(binary.BigEndian.Uint64(b[:])), nil
}
