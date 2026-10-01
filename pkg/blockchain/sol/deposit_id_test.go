package sol

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/layer-3/clearnet-sdk/pkg/blockchain/internal/depositidtest"
	"github.com/layer-3/clearnet-sdk/pkg/blockchain/sol/custody"
	"github.com/layer-3/clearnet-sdk/pkg/core"
)

type depositIDVector struct {
	Name      string `json:"name"`
	Signature string `json:"signature"`
	Index     string `json:"index"`
	ID        string `json:"id"`
}

func readDepositIDVectors(t *testing.T) []depositIDVector {
	return depositidtest.Vectors[depositIDVector](t, "sol")
}

// The vectors were generated from custody's deposit watcher; DepositID must
// reproduce them byte for byte, and parseDepositID must invert them.
func TestDepositID_GoldenVectors(t *testing.T) {
	for _, v := range readDepositIDVectors(t) {
		t.Run(v.Name, func(t *testing.T) {
			sig, err := solana.SignatureFromBase58(v.Signature)
			if err != nil {
				t.Fatalf("bad signature: %v", err)
			}
			index, err := strconv.ParseUint(v.Index, 10, 64)
			if err != nil {
				t.Fatalf("bad index: %v", err)
			}
			if got := DepositID(sig, index); got != v.ID {
				t.Fatalf("DepositID() = %s, want %s", got, v.ID)
			}
			gotIndex, err := parseDepositID(sig, v.ID)
			if err != nil || gotIndex != index {
				t.Fatalf("parseDepositID() = (%d, %v), want (%d, nil)", gotIndex, err, index)
			}
		})
	}
}

func TestParseDepositIDRejectsOtherShapes(t *testing.T) {
	var sig, other solana.Signature
	sig[0], other[0] = 1, 2
	id := DepositID(sig, 3)
	for _, bad := range []string{
		"",
		sig.String(),
		strings.TrimSuffix(id, ":3"),
		strings.TrimSuffix(id, "3") + "03",
		strings.TrimSuffix(id, "3") + "-3",
		strings.TrimSuffix(id, "3") + "18446744073709551616",
		strings.ToUpper(id[:2]) + id[2:],
		"0x" + strings.ToUpper(id[2:]),
		DepositID(other, 3),
	} {
		if _, err := parseDepositID(sig, bad); err == nil {
			t.Fatalf("parseDepositID(%q) accepted a deposit ID for another shape or signature", bad)
		}
	}
}

var testProgramID = solana.MustPublicKeyFromBase58("Custody111111111111111111111111111111111111")

func eventCPIData(disc [8]byte, bodyLen int) solana.Base58 {
	data := append(append(eventIxTag[:], disc[:]...), make([]byte, bodyLen)...)
	return solana.Base58(data)
}

func depositedCPI(programIndex uint16) rpc.CompiledInstruction {
	return rpc.CompiledInstruction{ProgramIDIndex: programIndex, Data: eventCPIData(custody.Event_Deposited, depositedEventMinLen)}
}

func executedCPI(programIndex uint16) rpc.CompiledInstruction {
	return rpc.CompiledInstruction{ProgramIDIndex: programIndex, Data: eventCPIData(custody.Event_Executed, executedEventMinLen)}
}

func TestDepositEventAt(t *testing.T) {
	other := solana.MustPublicKeyFromBase58("11111111111111111111111111111111")
	lookup := solana.PublicKey{9}
	tx := &solana.Transaction{Message: solana.Message{AccountKeys: []solana.PublicKey{other, testProgramID}}}
	const program, foreign, loaded = 1, 0, 2
	truncated := rpc.CompiledInstruction{ProgramIDIndex: program, Data: eventCPIData(custody.Event_Deposited, depositedEventMinLen-1)}
	unknownDisc := rpc.CompiledInstruction{ProgramIDIndex: program, Data: eventCPIData([8]byte{1}, 200)}
	noTag := rpc.CompiledInstruction{ProgramIDIndex: program, Data: solana.Base58(make([]byte, 200))}
	meta := func(ixs ...rpc.CompiledInstruction) *rpc.TransactionMeta {
		return &rpc.TransactionMeta{
			InnerInstructions: []rpc.InnerInstruction{{Index: 0, Instructions: ixs}},
			LoadedAddresses:   rpc.LoadedAddresses{ReadOnly: []solana.PublicKey{lookup}},
		}
	}
	tests := []struct {
		name      string
		meta      *rpc.TransactionMeta
		programID solana.PublicKey
		index     uint64
		want      bool
	}{
		{"single deposit", meta(depositedCPI(program)), testProgramID, 0, true},
		{"index past last event", meta(depositedCPI(program)), testProgramID, 1, false},
		{"executed at index", meta(executedCPI(program), depositedCPI(program)), testProgramID, 0, false},
		{"deposit after executed", meta(executedCPI(program), depositedCPI(program)), testProgramID, 1, true},
		{"foreign program not counted", meta(depositedCPI(foreign), depositedCPI(program)), testProgramID, 0, true},
		{"truncated event not counted", meta(truncated, depositedCPI(program)), testProgramID, 0, true},
		{"unknown event not counted", meta(unknownDisc, depositedCPI(program)), testProgramID, 0, true},
		{"missing event tag not counted", meta(noTag, depositedCPI(program)), testProgramID, 0, true},
		{"program from lookup table", meta(depositedCPI(loaded)), lookup, 0, true},
		{"out-of-range program index", meta(depositedCPI(7)), testProgramID, 0, false},
		{"other program ID", meta(depositedCPI(program)), other, 0, false},
		{"no inner instructions", &rpc.TransactionMeta{}, testProgramID, 0, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := depositEventAt(tx, tc.meta, tc.programID, tc.index); got != tc.want {
				t.Fatalf("depositEventAt() = %v, want %v", got, tc.want)
			}
		})
	}
	// Indexing spans inner-instruction groups in order.
	split := &rpc.TransactionMeta{InnerInstructions: []rpc.InnerInstruction{
		{Index: 0, Instructions: []rpc.CompiledInstruction{executedCPI(program)}},
		{Index: 1, Instructions: []rpc.CompiledInstruction{depositedCPI(program)}},
	}}
	if !depositEventAt(tx, split, testProgramID, 1) {
		t.Fatal("depositEventAt did not count events across inner-instruction groups")
	}
}

// solStatusServer answers getSignatureStatuses and getTransaction for one
// transaction whose inner instructions are ixs.
// statusErr sets the error in the signature status, metaErr in the transaction
// meta; either one alone marks the transaction failed.
func solStatusServer(t *testing.T, confirmation string, statusErr, metaErr bool, served bool, ixs []rpc.CompiledInstruction) (*httptest.Server, solana.Signature) {
	t.Helper()
	payer := solana.PublicKey{5}
	tx, err := solana.NewTransaction([]solana.Instruction{
		solana.NewInstruction(testProgramID, solana.AccountMetaSlice{solana.Meta(payer).WRITE().SIGNER()}, []byte{1}),
	}, solana.Hash{7}, solana.TransactionPayer(payer))
	if err != nil {
		t.Fatal(err)
	}
	var sig solana.Signature
	binary.BigEndian.PutUint64(sig[:8], 0xfeedface)
	tx.Signatures = []solana.Signature{sig}
	rawTx, err := tx.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	programIndex := -1
	for i, k := range tx.Message.AccountKeys {
		if k.Equals(testProgramID) {
			programIndex = i
		}
	}
	for i := range ixs {
		ixs[i].ProgramIDIndex = uint16(programIndex)
	}
	inner, err := json.Marshal([]rpc.InnerInstruction{{Index: 0, Instructions: ixs}})
	if err != nil {
		t.Fatal(err)
	}
	errJSON := func(set bool) string {
		if set {
			return `{"InstructionError":[0,"InvalidArgument"]}`
		}
		return "null"
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			t.Errorf("decode request: %v", err)
		}
		var result string
		switch req.Method {
		case "getSignatureStatuses":
			result = fmt.Sprintf(`{"context":{"slot":10},"value":[{"slot":9,"confirmations":null,"err":%s,"confirmationStatus":%q}]}`, errJSON(statusErr), confirmation)
		case "getTransaction":
			if !served {
				result = "null"
				break
			}
			result = fmt.Sprintf(`{"slot":9,"blockTime":null,"transaction":[%q,"base64"],"meta":{"err":%s,"fee":5000,"preBalances":[],"postBalances":[],"innerInstructions":%s,"loadedAddresses":{"writable":[],"readonly":[]}}}`,
				base64.StdEncoding.EncodeToString(rawTx), errJSON(metaErr), inner)
		default:
			t.Errorf("unexpected method %q", req.Method)
		}
		fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":%s}`, req.ID, result)
	}))
	t.Cleanup(srv.Close)
	return srv, sig
}

func TestDepositorChainDepositStatusChecksDepositEvent(t *testing.T) {
	tests := []struct {
		name         string
		confirmation string
		statusErr    bool
		metaErr      bool
		served       bool
		ixs          []rpc.CompiledInstruction
		index        uint64
		minConf      uint64
		want         core.DepositStatus
	}{
		{"finalized deposit", "finalized", false, false, true, []rpc.CompiledInstruction{depositedCPI(0)}, 0, 1, core.DepositConfirmed},
		{"confirmed deposit, minConf 0", "confirmed", false, false, true, []rpc.CompiledInstruction{depositedCPI(0)}, 0, 0, core.DepositConfirmed},
		{"confirmed deposit, minConf 1", "confirmed", false, false, true, []rpc.CompiledInstruction{depositedCPI(0)}, 0, 1, core.DepositPending},
		{"processed", "processed", false, false, true, []rpc.CompiledInstruction{depositedCPI(0)}, 0, 0, core.DepositPending},
		{"failed transaction", "finalized", true, true, true, []rpc.CompiledInstruction{depositedCPI(0)}, 0, 1, core.DepositAbsent},
		{"failed status only", "finalized", true, false, true, []rpc.CompiledInstruction{depositedCPI(0)}, 0, 1, core.DepositAbsent},
		{"failed transaction meta only", "finalized", false, true, true, []rpc.CompiledInstruction{depositedCPI(0)}, 0, 1, core.DepositAbsent},
		{"wrong index", "finalized", false, false, true, []rpc.CompiledInstruction{depositedCPI(0)}, 1, 1, core.DepositAbsent},
		{"executed event at index", "finalized", false, false, true, []rpc.CompiledInstruction{executedCPI(0)}, 0, 1, core.DepositAbsent},
		{"no custody event", "finalized", false, false, true, nil, 0, 1, core.DepositAbsent},
		{"transaction not served yet", "finalized", false, false, false, []rpc.CompiledInstruction{depositedCPI(0)}, 0, 1, core.DepositPending},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv, sig := solStatusServer(t, tc.confirmation, tc.statusErr, tc.metaErr, tc.served, tc.ixs)
			d := &Depositor{client: rpc.New(srv.URL), programID: testProgramID}
			got, err := d.ChainDepositStatus(context.Background(), sig.String(), DepositID(sig, tc.index), tc.minConf)
			if err != nil || got != tc.want {
				t.Fatalf("ChainDepositStatus = (%v, %v), want (%v, nil)", got, err, tc.want)
			}
		})
	}
}

func TestDepositorChainDepositStatusRejectsMismatchedDepositID(t *testing.T) {
	srv, sig := solStatusServer(t, "finalized", false, false, true, []rpc.CompiledInstruction{depositedCPI(0)})
	d := &Depositor{client: rpc.New(srv.URL), programID: testProgramID}
	var other solana.Signature
	other[0] = 1
	for _, id := range []string{sig.String(), DepositID(other, 0)} {
		if got, err := d.ChainDepositStatus(context.Background(), sig.String(), id, 1); err == nil || got != core.DepositAbsent {
			t.Fatalf("ChainDepositStatus(%q) = (%v, %v), want (absent, error)", id, got, err)
		}
	}
}

func TestParseDepositIDInputs(t *testing.T) {
	var sig, other solana.Signature
	sig[0], other[0] = 1, 2
	prefix := strings.TrimSuffix(DepositID(sig, 0), ":0")
	otherPrefix := strings.TrimSuffix(DepositID(other, 0), ":0")
	notShape := func(id string) string {
		return fmt.Sprintf("sol: deposit ID %q is not 0x<signature hash>:<index>", id)
	}
	mismatch := func(id string) string {
		return fmt.Sprintf("sol: deposit ID %q does not match signature %s", id, sig)
	}
	accepted := map[string]uint64{
		prefix + ":0":                    0,
		prefix + ":42":                   42,
		prefix + ":18446744073709551615": 18446744073709551615,
	}
	for id, want := range accepted {
		if got, err := parseDepositID(sig, id); err != nil || got != want {
			t.Fatalf("parseDepositID(%q) = (%d, %v), want (%d, nil)", id, got, err, want)
		}
	}
	rejected := map[string]string{
		"":                               notShape(""),
		prefix:                           notShape(prefix),
		"0":                              notShape("0"),
		prefix + ":":                     mismatch(prefix + ":"),
		prefix + ":18446744073709551616": mismatch(prefix + ":18446744073709551616"),
		prefix + ":01":                   mismatch(prefix + ":01"),
		prefix + ":+1":                   mismatch(prefix + ":+1"),
		prefix + ":1:2":                  mismatch(prefix + ":1:2"),
		prefix + "::1":                   mismatch(prefix + "::1"),
		strings.ToUpper(prefix) + ":0":   mismatch(strings.ToUpper(prefix) + ":0"),
		otherPrefix + ":0":               mismatch(otherPrefix + ":0"),
		":0":                             mismatch(":0"),
		"x" + prefix + ":0":              mismatch("x" + prefix + ":0"),
	}
	for id, want := range rejected {
		if _, err := parseDepositID(sig, id); err == nil || err.Error() != want {
			t.Fatalf("parseDepositID(%q) error = %v, want %q", id, err, want)
		}
	}
}
