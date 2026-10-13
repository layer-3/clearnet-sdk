package xrpl

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Peersyst/xrpl-go/xrpl/transaction"

	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/decimal"
	"github.com/layer-3/clearnet-sdk/pkg/sign"
)

const (
	floorTestVault   = "rVaULtAdd1111111111111111111111111"
	floorTestBaseFee = uint64(10)
	floorTestQuorum  = 3
	// floorTestFloor is base_fee × (1 + quorum) for the live quorum.
	floorTestFloor = floorTestBaseFee * (1 + floorTestQuorum)
)

func TestMultisignFeeFloor(t *testing.T) {
	if got, err := multisignFeeFloor(10, 3); err != nil || got != 40 {
		t.Fatalf("multisignFeeFloor(10, 3) = %d, %v; want 40", got, err)
	}
	for _, tc := range []struct {
		base   uint64
		quorum int
	}{{0, 3}, {10, 0}, {10, -1}, {^uint64(0), 1}} {
		if _, err := multisignFeeFloor(tc.base, tc.quorum); err == nil {
			t.Errorf("multisignFeeFloor(%d, %d) accepted", tc.base, tc.quorum)
		}
	}
}

// TestCheckFeeIsInclusive covers the shared [floor, ceiling] range.
func TestCheckFeeIsInclusive(t *testing.T) {
	for _, tc := range []struct {
		name string
		fee  uint64
		ok   bool
	}{
		{"one drop", 1, false},
		{"just below floor", floorTestFloor - 1, false},
		{"exact floor", floorTestFloor, true},
		{"exact ceiling", maxAcceptableFeeDrops, true},
		{"above ceiling", maxAcceptableFeeDrops + 1, false},
	} {
		if err := checkFee("canonical", json.Number(jsonUint(tc.fee)), floorTestFloor); (err == nil) != tc.ok {
			t.Errorf("%s: checkFee(%d) error = %v, want ok=%v", tc.name, tc.fee, err, tc.ok)
		}
	}
	if err := checkFee("canonical", "100", 0); err == nil {
		t.Error("unknown floor (0) accepted")
	}
}

func jsonUint(v uint64) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// floorWithdrawal is a canonical Payment for floorTestVault with the given Fee.
func floorWithdrawal(t *testing.T, fee uint64) ([]byte, *core.WithdrawalOp, [32]byte) {
	t.Helper()
	ctx := context.Background()
	op := &core.WithdrawalOp{
		Recipient: "rDeST1111111111111111111111111111",
		AssetURI:  "yellow://ynet/asset/0x0000000000000000000000000000000000001234/xrpl/0/0",
		Amount:    decimal.NewFromInt(1),
	}
	amt, err := BuildAmount(ctx, NewAssetResolver(AssetResolverConfig{}), op)
	if err != nil {
		t.Fatalf("BuildAmount: %v", err)
	}
	var wid [32]byte
	wid[0], wid[31] = 0xAB, 0xCD
	packed, err := CanonicalJSON(transaction.FlatTransaction{
		"TransactionType": "Payment",
		"Account":         floorTestVault,
		"Destination":     op.Recipient,
		"Amount":          amt,
		"InvoiceID":       strings.ToUpper(hex.EncodeToString(wid[:])),
		"TicketSequence":  uint32(5),
		"Sequence":        uint32(0),
		"Fee":             jsonUint(fee),
		"SigningPubKey":   "",
	})
	if err != nil {
		t.Fatalf("CanonicalJSON: %v", err)
	}
	return packed, op, wid
}

// floorRotation is a canonical SignerListSet for floorTestVault installing
// newSigners at newThreshold with the given Fee.
func floorRotation(t *testing.T, newSigners []string, newThreshold int, fee uint64) []byte {
	t.Helper()
	entries, err := signerEntries(newSigners, newThreshold)
	if err != nil {
		t.Fatalf("signerEntries: %v", err)
	}
	packed, err := CanonicalJSON(transaction.FlatTransaction{
		"TransactionType": "SignerListSet",
		"Account":         floorTestVault,
		"SignerQuorum":    uint32(newThreshold),
		"SignerEntries":   entries,
		"Sequence":        uint32(7),
		"Fee":             jsonUint(fee),
		"SigningPubKey":   "",
	})
	if err != nil {
		t.Fatalf("CanonicalJSON: %v", err)
	}
	return packed
}

// countingSigner records Sign calls so tests can prove Validate never signs.
type countingSigner struct {
	sign.Signer
	calls atomic.Int32
}

func (s *countingSigner) Sign(ctx context.Context, msg []byte) ([]byte, error) {
	s.calls.Add(1)
	return s.Signer.Sign(ctx, msg)
}

// fakeRippled answers server_info, server_state and account_info. The
// validated flags let a test serve unvalidated state.
type fakeRippled struct {
	baseFee          uint64
	ledgerValidated  bool
	quorum           int
	entries          int
	signersValidated bool
}

func (f *fakeRippled) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Method string `json:"method"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var result any
	switch req.Method {
	case "server_info":
		result = map[string]any{"info": map[string]any{"network_id": 0}, "status": "success"}
	case "server_state":
		validated := map[string]any{"base_fee": f.baseFee, "seq": 0}
		if f.ledgerValidated {
			validated["seq"] = 1000
		}
		result = map[string]any{"state": map[string]any{"validated_ledger": validated}, "status": "success"}
	case "account_info":
		entries := make([]any, f.entries)
		for i := range entries {
			entries[i] = map[string]any{"SignerEntry": map[string]any{"Account": "rSigner" + strings.Repeat("1", 26) + string(rune('a'+i)), "SignerWeight": 1}}
		}
		result = map[string]any{
			"account_data": map[string]any{"Account": floorTestVault},
			"signer_lists": []any{map[string]any{"SignerQuorum": f.quorum, "SignerEntries": entries}},
			"validated":    f.signersValidated,
			"status":       "success",
		}
	default:
		http.Error(w, "unexpected method "+req.Method, http.StatusBadRequest)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"result": result})
}

func newFakeRippled(t *testing.T, f *fakeRippled) string {
	t.Helper()
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	return srv.URL
}

func healthyRippled() *fakeRippled {
	return &fakeRippled{baseFee: floorTestBaseFee, ledgerValidated: true, quorum: floorTestQuorum, entries: 5, signersValidated: true}
}

func newFloorWithdrawalFinalizer(t *testing.T, rpcURL string) (*WithdrawalFinalizer, *countingSigner) {
	t.Helper()
	inner, _ := newXRPLSigner(t)
	s := &countingSigner{Signer: inner}
	f, err := NewWithdrawalFinalizer(rpcURL, floorTestVault, 1, s, nil, NewAssetResolver(AssetResolverConfig{}))
	if err != nil {
		t.Fatalf("NewWithdrawalFinalizer: %v", err)
	}
	f.SetStandaloneLedgerMode(true)
	return f, s
}

func newFloorRotationFinalizer(t *testing.T, rpcURL string) (*RotationFinalizer, *countingSigner) {
	t.Helper()
	inner, _ := newXRPLSigner(t)
	s := &countingSigner{Signer: inner}
	f, err := NewRotationFinalizer(rpcURL, floorTestVault, 1, s)
	if err != nil {
		t.Fatalf("NewRotationFinalizer: %v", err)
	}
	f.SetStandaloneLedgerMode(true)
	return f, s
}

var floorTestBounds = core.WithdrawalTimeBounds{FinalizedAt: 1000, ValidUntil: 4600}

// TestWithdrawalFeeFloor drives WithdrawalFinalizer.Validate against validated
// base fee and live quorum read over RPC (the default resolvers).
func TestWithdrawalFeeFloor(t *testing.T) {
	f, s := newFloorWithdrawalFinalizer(t, newFakeRippled(t, healthyRippled()))
	ctx := context.Background()
	for _, tc := range []struct {
		name string
		fee  uint64
		ok   bool
	}{
		{"one drop", 1, false},
		{"just below floor", floorTestFloor - 1, false},
		{"exact floor", floorTestFloor, true},
		{"above ceiling", maxAcceptableFeeDrops + 1, false},
	} {
		packed, op, wid := floorWithdrawal(t, tc.fee)
		err := f.Validate(ctx, packed, op, wid, floorTestBounds)
		if tc.ok {
			if err != nil {
				t.Errorf("%s: Validate rejected Fee %d: %v", tc.name, tc.fee, err)
			}
			continue
		}
		if !errors.Is(err, ErrCandidateRejected) {
			t.Errorf("%s: Validate(Fee %d) error = %v, want ErrCandidateRejected", tc.name, tc.fee, err)
		}
	}
	if n := s.calls.Load(); n != 0 {
		t.Fatalf("Validate invoked the signer %d times", n)
	}
}

// TestWithdrawalFeeFloorUsesThresholdResolver shows an installed threshold and
// base-fee resolver, not the static threshold, size the floor.
func TestWithdrawalFeeFloorUsesThresholdResolver(t *testing.T) {
	f, _ := newFloorWithdrawalFinalizer(t, newFakeRippled(t, healthyRippled()))
	f.SetThresholdResolver(func(context.Context) (int, error) { return 4, nil })
	f.SetBaseFeeResolver(func(context.Context) (uint64, error) { return 12, nil })
	ctx := context.Background()
	const floor = 12 * (1 + 4)
	packed, op, wid := floorWithdrawal(t, floor-1)
	if err := f.Validate(ctx, packed, op, wid, floorTestBounds); !errors.Is(err, ErrCandidateRejected) {
		t.Fatalf("below resolved floor: error = %v, want ErrCandidateRejected", err)
	}
	packed, op, wid = floorWithdrawal(t, floor)
	if err := f.Validate(ctx, packed, op, wid, floorTestBounds); err != nil {
		t.Fatalf("exact resolved floor rejected: %v", err)
	}
}

// TestWithdrawalFeeFloorLookupFailureIsOperational covers failed and
// unvalidated lookups: Validate refuses without ErrCandidateRejected and never
// signs, even for a body whose Fee would pass.
func TestWithdrawalFeeFloorLookupFailureIsOperational(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name  string
		node  func() *fakeRippled
		setup func(*WithdrawalFinalizer)
	}{
		{"unvalidated base fee", func() *fakeRippled { n := healthyRippled(); n.ledgerValidated = false; return n }, nil},
		{"zero base fee", func() *fakeRippled { n := healthyRippled(); n.baseFee = 0; return n }, nil},
		{"unvalidated signer list", func() *fakeRippled { n := healthyRippled(); n.signersValidated = false; return n }, nil},
		{"base fee resolver fails", healthyRippled, func(f *WithdrawalFinalizer) {
			f.SetBaseFeeResolver(func(context.Context) (uint64, error) { return 0, errors.New("boom") })
		}},
		{"threshold resolver fails", healthyRippled, func(f *WithdrawalFinalizer) {
			f.SetThresholdResolver(func(context.Context) (int, error) { return 0, errors.New("boom") })
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, s := newFloorWithdrawalFinalizer(t, newFakeRippled(t, tc.node()))
			if tc.setup != nil {
				tc.setup(f)
			}
			packed, op, wid := floorWithdrawal(t, floorTestFloor*2)
			err := f.Validate(ctx, packed, op, wid, floorTestBounds)
			if err == nil {
				t.Fatal("Validate accepted with a failed floor lookup")
			}
			if errors.Is(err, ErrCandidateRejected) {
				t.Fatalf("lookup failure attributed to the candidate: %v", err)
			}
			if n := s.calls.Load(); n != 0 {
				t.Fatalf("Validate invoked the signer %d times", n)
			}
		})
	}
}

var floorNewSigners = []string{"rAAA1111111111111111111111111111aa", "rBBB1111111111111111111111111111bb"}

// TestRotationFeeFloor drives RotationFinalizer.Validate against the default
// validated resolvers.
func TestRotationFeeFloor(t *testing.T) {
	f, s := newFloorRotationFinalizer(t, newFakeRippled(t, healthyRippled()))
	ctx := context.Background()
	for _, tc := range []struct {
		name string
		fee  uint64
		ok   bool
	}{
		{"one drop", 1, false},
		{"just below floor", floorTestFloor - 1, false},
		{"exact floor", floorTestFloor, true},
		{"above ceiling", maxAcceptableFeeDrops + 1, false},
	} {
		err := f.Validate(ctx, [32]byte{}, floorRotation(t, floorNewSigners, 2, tc.fee), floorNewSigners, 2)
		if tc.ok {
			if err != nil {
				t.Errorf("%s: Validate rejected Fee %d: %v", tc.name, tc.fee, err)
			}
			continue
		}
		if !errors.Is(err, ErrCandidateRejected) {
			t.Errorf("%s: Validate(Fee %d) error = %v, want ErrCandidateRejected", tc.name, tc.fee, err)
		}
	}
	if n := s.calls.Load(); n != 0 {
		t.Fatalf("Validate invoked the signer %d times", n)
	}
}

// TestRotationFeeFloorUsesCurrentSignerList shows the SignerListSet floor is
// sized by the CURRENT list's quorum (which signs it), not the new list's. The
// live quorum is 3; the new list has threshold 1. A Fee sized for the new list
// (base × 2) is rejected; the current-quorum floor (base × 4) is accepted. The
// reverse holds when the new threshold exceeds the current quorum.
func TestRotationFeeFloorUsesCurrentSignerList(t *testing.T) {
	ctx := context.Background()

	lowering, _ := newFloorRotationFinalizer(t, newFakeRippled(t, healthyRippled()))
	if err := lowering.Validate(ctx, [32]byte{}, floorRotation(t, floorNewSigners, 1, floorTestBaseFee*(1+1)), floorNewSigners, 1); !errors.Is(err, ErrCandidateRejected) {
		t.Fatalf("Fee sized for the new quorum accepted: %v", err)
	}
	if err := lowering.Validate(ctx, [32]byte{}, floorRotation(t, floorNewSigners, 1, floorTestFloor), floorNewSigners, 1); err != nil {
		t.Fatalf("Fee sized for the current quorum rejected: %v", err)
	}

	// Raising: current quorum 1, new list threshold 5. The current-quorum
	// floor (base × 2) is enough; the new list's size is irrelevant.
	node := healthyRippled()
	node.quorum = 1
	raising, _ := newFloorRotationFinalizer(t, newFakeRippled(t, node))
	five := []string{
		"rAAA1111111111111111111111111111aa", "rBBB1111111111111111111111111111bb",
		"rCCC1111111111111111111111111111cc", "rDDD1111111111111111111111111111dd",
		"rEEE1111111111111111111111111111ee",
	}
	if err := raising.Validate(ctx, [32]byte{}, floorRotation(t, five, 5, floorTestBaseFee*(1+1)), five, 5); err != nil {
		t.Fatalf("current-quorum floor rejected on a quorum-raising rotation: %v", err)
	}
	if err := raising.Validate(ctx, [32]byte{}, floorRotation(t, five, 5, floorTestBaseFee*(1+1)-1), five, 5); !errors.Is(err, ErrCandidateRejected) {
		t.Fatalf("below current-quorum floor accepted: %v", err)
	}
}

// TestRotationFeeFloorLookupFailureIsOperational is the rotation analogue of
// TestWithdrawalFeeFloorLookupFailureIsOperational.
func TestRotationFeeFloorLookupFailureIsOperational(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name  string
		node  func() *fakeRippled
		setup func(*RotationFinalizer)
	}{
		{"unvalidated base fee", func() *fakeRippled { n := healthyRippled(); n.ledgerValidated = false; return n }, nil},
		{"unvalidated signer list", func() *fakeRippled { n := healthyRippled(); n.signersValidated = false; return n }, nil},
		{"base fee resolver fails", healthyRippled, func(f *RotationFinalizer) {
			f.SetBaseFeeResolver(func(context.Context) (uint64, error) { return 0, errors.New("boom") })
		}},
		{"threshold resolver fails", healthyRippled, func(f *RotationFinalizer) {
			f.SetThresholdResolver(func(context.Context) (int, error) { return 0, errors.New("boom") })
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, s := newFloorRotationFinalizer(t, newFakeRippled(t, tc.node()))
			if tc.setup != nil {
				tc.setup(f)
			}
			err := f.Validate(ctx, [32]byte{}, floorRotation(t, floorNewSigners, 2, floorTestFloor*2), floorNewSigners, 2)
			if err == nil {
				t.Fatal("Validate accepted with a failed floor lookup")
			}
			if errors.Is(err, ErrCandidateRejected) {
				t.Fatalf("lookup failure attributed to the candidate: %v", err)
			}
			if n := s.calls.Load(); n != 0 {
				t.Fatalf("Validate invoked the signer %d times", n)
			}
		})
	}
}
