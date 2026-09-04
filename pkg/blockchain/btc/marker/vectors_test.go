package marker

// vectors_test.go drives testdata/vectors.json, the language-neutral
// golden-vector file a non-Go (e.g. TypeScript) mirror implementation
// conforms to. Every case below is exercised against the real
// Encode/Decode/Scan functions, so a bug in the implementation cannot
// silently produce a wrong golden file - the same assertions that check
// the code also generate the fixture.
//
// Run with -update to (re)write testdata/vectors.json:
//
//	go test ./pkg/blockchain/btc/marker/... -update -run TestVectors

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"
)

// vectorsUpdate mirrors sdk pkg/cborx/goldens_test.go:31's flag and policy:
// CI treats a missing or drifted fixture as a failure, not a reason to
// re-seed.
var vectorsUpdate = flag.Bool("update", false, "regenerate testdata/vectors.json")

// ---- JSON schema ----------------------------------------------------------

type vectorsFile struct {
	Format               string           `json:"format"`
	FormatVersion        int              `json:"formatVersion"`
	ADR                  string           `json:"adr"`
	Notes                string           `json:"notes"`
	ValidationOrderNotes string           `json:"validationOrderNotes"`
	GenericTag           genericTagJSON   `json:"genericTag"`
	ErrorCodes           []string         `json:"errorCodes"`
	ValidationOrder      []string         `json:"validationOrder"`
	Encode               []encodeCaseJSON `json:"encode"`
	Decode               []decodeCaseJSON `json:"decode"`
	Scan                 []scanCaseJSON   `json:"scan"`
}

type genericTagJSON struct {
	Preimage  string `json:"preimage"`
	SHA256Hex string `json:"sha256Hex"`
}

type markerJSON struct {
	Version      byte    `json:"version"`
	AddressHex   string  `json:"addressHex"`
	ReferenceHex *string `json:"referenceHex"`
}

type encodeCaseJSON struct {
	Case        string     `json:"case"`
	Notes       string     `json:"notes,omitempty"`
	Marker      markerJSON `json:"marker"`
	PayloadHex  string     `json:"payloadHex,omitempty"`
	ScriptHex   string     `json:"scriptHex,omitempty"`
	ExpectError string     `json:"expectError,omitempty"`
}

type decodeExpectJSON struct {
	OK           bool    `json:"ok"`
	Version      byte    `json:"version,omitempty"`
	AddressHex   string  `json:"addressHex,omitempty"`
	ReferenceHex *string `json:"referenceHex,omitempty"`
	Error        string  `json:"error,omitempty"`
}

type decodeCaseJSON struct {
	Case      string           `json:"case"`
	Notes     string           `json:"notes,omitempty"`
	ScriptHex string           `json:"scriptHex"`
	Expect    decodeExpectJSON `json:"expect"`
}

type scanCaseJSON struct {
	Case  string `json:"case"`
	Notes string `json:"notes,omitempty"`
	// ScriptPubKeysHex is every scriptPubKey of one transaction, in order.
	// Output values are deliberately absent from the contract: they are not an
	// input to the rule, so no conforming implementation can filter on what it
	// is never given.
	ScriptPubKeysHex []string         `json:"scriptPubKeysHex"`
	Expect           decodeExpectJSON `json:"expect"`
}

// ---- error <-> stable string code -----------------------------------------

// codeByErr maps every sentinel this package exports to the stable string
// code vectors.json uses. Error codes are contract, not Go error text:
// changing a code is a formatVersion bump, not a refactor.
var codeByErr = map[error]string{
	ErrNotMarker:        "not_marker",
	ErrMultiplePushes:   "multiple_pushes",
	ErrNonCanonicalPush: "non_canonical_push",
	ErrUnknownVersion:   "unknown_version",
	ErrBadLength:        "bad_length",
	ErrZeroAddress:      "zero_address",
	ErrZeroReference:    "zero_reference",
	ErrNoMarker:         "no_marker",
	ErrMultipleMarkers:  "multiple_markers",
}

func errCode(t *testing.T, err error) string {
	t.Helper()
	if err == nil {
		return ""
	}
	for e, code := range codeByErr {
		if err == e {
			return code
		}
	}
	t.Fatalf("unmapped error: %v", err)
	return ""
}

// ---- test-side marker/hex helpers ------------------------------------------

const (
	zeroAddrHex = "0000000000000000000000000000000000000000"
	minAddrHex  = "0000000000000000000000000000000000000001"
	maxAddrHex  = "ffffffffffffffffffffffffffffffffffffffff"

	zeroRefHex = "0000000000000000000000000000000000000000000000000000000000000000"
	minRefHex  = "0000000000000000000000000000000000000000000000000000000000000001"
	maxRefHex  = "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
)

func strp(s string) *string { return &s }

// ---- encode cases -----------------------------------------------------------

type encodeSpec struct {
	Case      string
	Notes     string
	Marker    Marker
	RefHexPtr *string // nil for Version1 (no reference field on the wire)
	WantErr   error   // nil means the encode must succeed
}

func encodeSpecs(t *testing.T) []encodeSpec {
	return []encodeSpec{
		{
			Case:   "v1_typical",
			Notes:  "ADR §2 baseline: 25-byte payload, 27-byte script.",
			Marker: Marker{Version: Version1, Address: mustAddr(t, testAddrHex)},
		},
		{
			Case:    "v1_zero_address_refused",
			Notes:   "There is no account whose identifier is 0x000...0, so a marker naming it can only be a construction bug or a probe. Applies to both versions; refused by the writer as well as the reader.",
			Marker:  Marker{Version: Version1, Address: mustAddr(t, zeroAddrHex)},
			WantErr: ErrZeroAddress,
		},
		{
			Case:   "v1_min_nonzero_address",
			Notes:  "Pins the boundary of the zero-address check.",
			Marker: Marker{Version: Version1, Address: mustAddr(t, minAddrHex)},
		},
		{
			Case:   "v1_max_address",
			Notes:  "All-ff address, valid.",
			Marker: Marker{Version: Version1, Address: mustAddr(t, maxAddrHex)},
		},
		{
			Case:      "v2_typical",
			Notes:     "ADR §2 sub-account: 57-byte payload, 59-byte script.",
			Marker:    Marker{Version: Version2, Address: mustAddr(t, testAddrHex), Reference: mustRef(t, "000000000000000000000000000000000000000000000000000000000000002a")},
			RefHexPtr: strp("000000000000000000000000000000000000000000000000000000000000002a"),
		},
		{
			Case:      "v2_min_nonzero_reference",
			Notes:     "Pins the boundary of the zero-reference check.",
			Marker:    Marker{Version: Version2, Address: mustAddr(t, testAddrHex), Reference: mustRef(t, minRefHex)},
			RefHexPtr: strp(minRefHex),
		},
		{
			Case:      "v2_max_reference",
			Notes:     "All-ff reference, valid.",
			Marker:    Marker{Version: Version2, Address: mustAddr(t, testAddrHex), Reference: mustRef(t, maxRefHex)},
			RefHexPtr: strp(maxRefHex),
		},
		{
			Case:      "v2_zero_reference_refused",
			Notes:     "The writer enforces the same rule as the reader (ADR §2), so a construction bug cannot emit a marker this format rejects.",
			Marker:    Marker{Version: Version2, Address: mustAddr(t, testAddrHex), Reference: mustRef(t, zeroRefHex)},
			RefHexPtr: strp(zeroRefHex),
			WantErr:   ErrZeroReference,
		},
		{
			Case:      "v2_zero_address_refused",
			Notes:     "Proves the all-zero-address rejection applies to Version2 too, not only Version1.",
			Marker:    Marker{Version: Version2, Address: mustAddr(t, zeroAddrHex), Reference: mustRef(t, "000000000000000000000000000000000000000000000000000000000000002a")},
			RefHexPtr: strp("000000000000000000000000000000000000000000000000000000000000002a"),
			WantErr:   ErrZeroAddress,
		},
		{
			Case:    "unknown_version_refused",
			Notes:   "Version precedes length/address checks; address here is deliberately non-zero so this case tests only the version check.",
			Marker:  Marker{Version: 0x03, Address: mustAddr(t, testAddrHex)},
			WantErr: ErrUnknownVersion,
		},
	}
}

func TestEncodeVectors(t *testing.T) {
	for _, spec := range encodeSpecs(t) {
		t.Run(spec.Case, func(t *testing.T) {
			_, err := EncodePayload(spec.Marker)
			if spec.WantErr != nil {
				if err != spec.WantErr {
					t.Fatalf("EncodePayload: err = %v, want %v", err, spec.WantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("EncodePayload: unexpected err %v", err)
			}
			script, err := EncodeScript(spec.Marker)
			if err != nil {
				t.Fatalf("EncodeScript: unexpected err %v", err)
			}
			// Every successful encode case round-trips through decode:
			// writer and reader must not drift apart.
			got, err := DecodeScript(script)
			if err != nil {
				t.Fatalf("DecodeScript(EncodeScript(m)): unexpected err %v", err)
			}
			if got != spec.Marker {
				t.Fatalf("DecodeScript(EncodeScript(m)) = %+v, want %+v", got, spec.Marker)
			}
		})
	}
}

// ---- decode cases -----------------------------------------------------------

type decodeSpec struct {
	Case    string
	Notes   string
	Script  []byte
	WantOK  bool
	Want    Marker // valid only when WantOK
	WantErr error  // valid only when !WantOK
}

func decodeSpecs(t *testing.T) []decodeSpec {
	v1p := v1Payload(t, testAddrHex)
	v2p := v2Payload(t, testAddrHex, "000000000000000000000000000000000000000000000000000000000000002a")
	v1minAddr := v1Payload(t, minAddrHex)
	v2minRef := v2Payload(t, testAddrHex, minRefHex)

	specs := []decodeSpec{
		{Case: "v1_typical", Script: rawDirectPush(v1p), WantOK: true, Want: Marker{Version: Version1, Address: mustAddr(t, testAddrHex)}},
		{Case: "v2_typical", Script: rawDirectPush(v2p), WantOK: true, Want: Marker{Version: Version2, Address: mustAddr(t, testAddrHex), Reference: mustRef(t, "000000000000000000000000000000000000000000000000000000000000002a")}},
		{Case: "v1_min_nonzero_address_roundtrip", Notes: "boundary cases must decode, not only encode: 0x00...01 is the smallest address the zero-address check admits", Script: rawDirectPush(v1minAddr), WantOK: true, Want: Marker{Version: Version1, Address: mustAddr(t, minAddrHex)}},
		{Case: "v2_min_nonzero_reference_roundtrip", Notes: "boundary cases must decode, not only encode: 0x00...01 is the smallest reference the zero-reference check admits", Script: rawDirectPush(v2minRef), WantOK: true, Want: Marker{Version: Version2, Address: mustAddr(t, testAddrHex), Reference: mustRef(t, minRefHex)}},

		{Case: "unknown_version_25", Notes: "version 0x03 on a 25-byte payload", Script: rawDirectPush(withVersion(v1p, 0x03)), WantErr: ErrUnknownVersion},
		{Case: "unknown_version_57", Notes: "version 0x03 on a 57-byte payload -- pins the check order: version before length", Script: rawDirectPush(withVersion(v2p, 0x03)), WantErr: ErrUnknownVersion},
		{Case: "unknown_version_00", Script: rawDirectPush(withVersion(v1p, 0x00)), WantErr: ErrUnknownVersion},
		{Case: "v1_one_short", Notes: "not zero_address, whatever the truncated address bytes are", Script: rawDirectPush(v1p[:len(v1p)-1]), WantErr: ErrBadLength},
		{Case: "v1_one_long", Script: rawDirectPush(append(append([]byte{}, v1p...), 0x00)), WantErr: ErrBadLength},
		{Case: "v2_one_short", Script: rawDirectPush(v2p[:len(v2p)-1]), WantErr: ErrBadLength},
		{Case: "magic_only", Notes: "payload is exactly YNET, 4 bytes, no version byte", Script: rawDirectPush([]byte(Magic)), WantErr: ErrBadLength},
		{Case: "v2_all_zero_reference", Script: rawDirectPush(rawPayloadV2(t, testAddrHex, zeroRefHex)), WantErr: ErrZeroReference},

		{Case: "v1_pushdata1_encoding", Notes: "Strict push rule: magic still matches, so this is a marker CANDIDATE and its transaction is unattributed -- it is NOT ignored.", Script: rawPushData1(v1p), WantErr: ErrNonCanonicalPush},
		{Case: "v1_pushdata2_encoding", Script: rawPushData2(v1p), WantErr: ErrNonCanonicalPush},
		{Case: "v2_pushdata4_encoding", Script: rawPushData4(v2p), WantErr: ErrNonCanonicalPush},
		{Case: "v1_all_zero_address", Script: rawDirectPush(rawPayloadV1(t, zeroAddrHex)), WantErr: ErrZeroAddress},
		{Case: "v2_all_zero_address", Notes: "the Version2 half of the all-zero-address rejection", Script: rawDirectPush(rawPayloadV2(t, zeroAddrHex, "000000000000000000000000000000000000000000000000000000000000002a")), WantErr: ErrZeroAddress},

		{Case: "wrong_magic", Script: rawDirectPush(withMagic(v1p, "YNEU")), WantErr: ErrNotMarker},
		{Case: "p2wsh_not_op_return", Script: append([]byte{0x00, 0x20}, make([]byte, 32)...), WantErr: ErrNotMarker},
		{Case: "bare_op_return_no_push", Script: []byte{opReturn}, WantErr: ErrNotMarker},
		{Case: "withdrawal_id_op_return_is_not_a_marker", Notes: "ADR §9: the deposit parser must not collide with extractWithdrawalID. A 32-byte push is neither 25 nor 57.", Script: rawDirectPush(bytesRepeat(0x01, 32)), WantErr: ErrNotMarker},
		{Case: "below_magic_length", Notes: "payload YNE, 3 bytes", Script: rawDirectPush([]byte("YNE")), WantErr: ErrNotMarker},
		{Case: "empty_script", Script: []byte{}, WantErr: ErrNotMarker},
		{Case: "p2pkh_arbitrary_script", Script: []byte{0x76, 0xa9, 0x14, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x88, 0xac}, WantErr: ErrNotMarker},
		{Case: "non_magic_pushdata1_is_still_ignored", Notes: "Ordering proof paired with v1_pushdata1_encoding: canonicality is judged only after the magic matches.", Script: rawPushData1(withMagic(v1p, "YNEU")), WantErr: ErrNotMarker},

		{Case: "multiple_pushes", Notes: "OP_RETURN + valid v1 push + a second 4-byte push", Script: append(rawDirectPush(v1p), rawDirectPush([]byte{1, 2, 3, 4})[1:]...), WantErr: ErrMultiplePushes},
	}

	// ---- trailing bytes after the single push ---------------------------
	//
	// The rule is "the push must account for the whole script", not "no
	// second push may follow". A mirror that counts pushes instead of bytes
	// accepts these as valid markers and credits a transaction this reader
	// holds, so the distinction belongs in the cross-language contract.
	validV1 := rawDirectPush(v1p)
	for _, tr := range []struct {
		name  string
		notes string
		extra []byte
	}{
		{"trailing_op_return_byte", "trailing 0x6a is not a push opcode at all, yet the marker before it must not be honoured", []byte{opReturn}},
		{"trailing_op_1", "trailing OP_1 pushes no data, so a push-counting reader sees one push and wrongly accepts", []byte{0x51}},
		{"trailing_truncated_push_opcode", "trailing 0x19 declares 25 bytes that are not present", []byte{0x19}},
		{"trailing_two_op_0", "two trailing OP_0s", []byte{0x00, 0x00}},
	} {
		specs = append(specs, decodeSpec{
			Case:    tr.name,
			Notes:   tr.notes,
			Script:  append(append([]byte{}, validV1...), tr.extra...),
			WantErr: ErrMultiplePushes,
		})
	}

	// ---- recognition reads the first push only ---------------------------
	//
	// A junk push before a well-formed marker payload means there was never a
	// candidate: the scriptPubKey is ignored, not held.
	specs = append(specs, decodeSpec{
		Case:    "junk_push_before_marker_is_not_a_candidate",
		Notes:   "OP_RETURN + a 1-byte push + a well-formed v1 marker push. Recognition tests the FIRST push only, so this is not_marker (ignored), not a candidate that fails validation.",
		Script:  append([]byte{opReturn, 0x01, 0x00}, rawDirectPush(v1p)[1:]...),
		WantErr: ErrNotMarker,
	})

	// ---- validation precedence ------------------------------------------
	//
	// One case per ordered pair of failures that can co-occur, asserting the
	// earlier check wins. The reader implements the order as a chain of early
	// returns, for which adjacent pairs would be transitively sufficient; the
	// full matrix is in the vector file because a mirror may realise the order
	// differently and still pass a narrower set. Scripts that are not an
	// OP_RETURN, or that carry no push, cannot co-occur with anything later -
	// there is no payload left to fail a later check.
	//
	// A few of these scripts duplicate a case above under a different name.
	// That is deliberate: the precedence set is meant to be runnable as a
	// self-contained group.
	unknownVersionP := withVersion(v1p, 0x03)
	badLengthP := v1p[:len(v1p)-1]
	zeroAddrP := rawPayloadV1(t, zeroAddrHex)
	zeroRefP := rawPayloadV2(t, testAddrHex, zeroRefHex)
	wrongMagicP := withMagic(v1p, "YNEU")

	for _, pc := range []struct {
		name    string
		script  []byte
		wantErr error
	}{
		{"precedence_not_marker_over_multiple_pushes", withSecondPush(rawDirectPush(wrongMagicP)), ErrNotMarker},
		{"precedence_not_marker_over_non_canonical", rawPushData1(wrongMagicP), ErrNotMarker},

		{"precedence_multiple_pushes_over_non_canonical", withSecondPush(rawPushData1(v1p)), ErrMultiplePushes},
		{"precedence_multiple_pushes_over_unknown_version", withSecondPush(rawDirectPush(unknownVersionP)), ErrMultiplePushes},
		{"precedence_multiple_pushes_over_bad_length", withSecondPush(rawDirectPush(badLengthP)), ErrMultiplePushes},
		{"precedence_multiple_pushes_over_zero_address", withSecondPush(rawDirectPush(zeroAddrP)), ErrMultiplePushes},
		{"precedence_multiple_pushes_over_zero_reference", withSecondPush(rawDirectPush(zeroRefP)), ErrMultiplePushes},

		{"precedence_non_canonical_over_unknown_version", rawPushData1(unknownVersionP), ErrNonCanonicalPush},
		{"precedence_non_canonical_over_bad_length", rawPushData1(badLengthP), ErrNonCanonicalPush},
		{"precedence_non_canonical_over_zero_address", rawPushData1(zeroAddrP), ErrNonCanonicalPush},
		{"precedence_non_canonical_over_zero_reference", rawPushData1(zeroRefP), ErrNonCanonicalPush},

		{"precedence_unknown_version_over_bad_length", rawDirectPush(append(append([]byte(Magic), 0x03), bytesRepeat(0xff, 10)...)), ErrUnknownVersion},
		{"precedence_unknown_version_over_zero_address", rawDirectPush(withVersion(zeroAddrP, 0x03)), ErrUnknownVersion},
		{"precedence_unknown_version_over_zero_reference", rawDirectPush(withVersion(zeroRefP, 0x03)), ErrUnknownVersion},

		{"precedence_bad_length_over_zero_address", rawDirectPush(append(append([]byte(Magic), Version1), make([]byte, 19)...)), ErrBadLength},
		{"precedence_bad_length_over_zero_reference", rawDirectPush(zeroRefP[:len(zeroRefP)-1]), ErrBadLength},

		{"precedence_zero_address_over_zero_reference", rawDirectPush(append(append([]byte(Magic), Version2), make([]byte, 20+32)...)), ErrZeroAddress},
	} {
		specs = append(specs, decodeSpec{
			Case:    pc.name,
			Notes:   "validation precedence: " + codeByErr[pc.wantErr] + " wins over the later check named in the case",
			Script:  pc.script,
			WantErr: pc.wantErr,
		})
	}

	return specs
}

func bytesRepeat(b byte, n int) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = b
	}
	return out
}

func TestDecodeVectors(t *testing.T) {
	for _, spec := range decodeSpecs(t) {
		t.Run(spec.Case, func(t *testing.T) {
			got, err := DecodeScript(spec.Script)
			if spec.WantOK {
				if err != nil {
					t.Fatalf("DecodeScript: unexpected err %v", err)
				}
				if got != spec.Want {
					t.Fatalf("DecodeScript = %+v, want %+v", got, spec.Want)
				}
				return
			}
			if err != spec.WantErr {
				t.Fatalf("DecodeScript: err = %v, want %v", err, spec.WantErr)
			}
		})
	}
}

// TestEncodeDecodePairing asserts that every encode refusal has a decode
// twin reporting the identical error code: v1_zero_address_refused <->
// v1_all_zero_address, v2_zero_address_refused <-> v2_all_zero_address,
// v2_zero_reference_refused <-> v2_all_zero_reference, and
// unknown_version_refused <-> unknown_version_25. If the writer were ever
// more permissive than the reader it would emit markers nothing credits.
func TestEncodeDecodePairing(t *testing.T) {
	pairs := []struct {
		encodeCase string
		decodeCase string
	}{
		{"v1_zero_address_refused", "v1_all_zero_address"},
		{"v2_zero_address_refused", "v2_all_zero_address"},
		{"v2_zero_reference_refused", "v2_all_zero_reference"},
		{"unknown_version_refused", "unknown_version_25"},
	}
	encByCase := map[string]encodeSpec{}
	for _, s := range encodeSpecs(t) {
		encByCase[s.Case] = s
	}
	decByCase := map[string]decodeSpec{}
	for _, s := range decodeSpecs(t) {
		decByCase[s.Case] = s
	}
	for _, p := range pairs {
		enc, ok := encByCase[p.encodeCase]
		if !ok {
			t.Fatalf("no encode case named %q", p.encodeCase)
		}
		dec, ok := decByCase[p.decodeCase]
		if !ok {
			t.Fatalf("no decode case named %q", p.decodeCase)
		}
		if enc.WantErr == nil || dec.WantErr == nil {
			t.Fatalf("pairing %s<->%s: expected both sides to be refusals", p.encodeCase, p.decodeCase)
		}
		if enc.WantErr != dec.WantErr {
			t.Fatalf("pairing %s<->%s: encode refuses %v, decode rejects %v", p.encodeCase, p.decodeCase, enc.WantErr, dec.WantErr)
		}
	}
}

// ---- scan cases -------------------------------------------------------------

type scanSpec struct {
	Case  string
	Notes string
	// Outputs is every scriptPubKey of one transaction, in order. There is no
	// value here: value is not an input to recognition, so the contract never
	// hands one to an implementation that might filter on it.
	Outputs [][]byte
	WantOK  bool
	Want    Marker
	WantErr error
}

func scanSpecs(t *testing.T) []scanSpec {
	v1 := v1Script(t, testAddrHex)
	v2 := v2Script(t, testAddrHex, "000000000000000000000000000000000000000000000000000000000000002a")
	valueOut := []byte{0x00, 0x20} // stub non-OP_RETURN scriptPubKey for a value output
	withdrawalID := rawDirectPush(bytesRepeat(0x01, 32))
	unknownVersion := rawDirectPush(withVersion(v1Payload(t, testAddrHex), 0x03))
	nonCanonical := rawPushData1(v1Payload(t, testAddrHex))
	zeroAddress := rawDirectPush(rawPayloadV1(t, zeroAddrHex))

	wantV1 := Marker{Version: Version1, Address: mustAddr(t, testAddrHex)}

	return []scanSpec{
		{Case: "one_marker_two_value_outputs", Notes: "ADR §3: one marker credits EVERY output paying the generic address.", Outputs: [][]byte{valueOut, v1, valueOut}, WantOK: true, Want: wantV1},
		{Case: "one_marker_one_value_output", Outputs: [][]byte{v1, valueOut}, WantOK: true, Want: wantV1},
		{Case: "values_only_no_marker", Outputs: [][]byte{valueOut, valueOut}, WantErr: ErrNoMarker},
		{Case: "two_identical_v1_markers", Outputs: [][]byte{v1, v1}, WantErr: ErrMultipleMarkers},
		{Case: "one_v1_one_v2", Outputs: [][]byte{v1, v2}, WantErr: ErrMultipleMarkers},
		{Case: "v1_plus_withdrawal_id_ignored", Notes: "the foreign OP_RETURN is ignored, not counted. Load-bearing.", Outputs: [][]byte{v1, withdrawalID}, WantOK: true, Want: wantV1},
		{Case: "v1_plus_unknown_version_candidate", Notes: "An invalid candidate still counts toward the exactly-one-marker rule, so the transaction is held rather than credited even though only one candidate is valid.", Outputs: [][]byte{v1, unknownVersion}, WantErr: ErrMultipleMarkers},
		{Case: "marker_at_index_2", Notes: "position-independence, ADR §3", Outputs: [][]byte{valueOut, valueOut, v1}, WantOK: true, Want: wantV1},
		{Case: "lone_marker_no_value_output", Notes: "A marker with nothing paying the vault still decodes: ScanOutputs answers the marker question only. Whether anything was paid is the caller's half of the rule.", Outputs: [][]byte{v1}, WantOK: true, Want: wantV1},
		{Case: "empty_output_list", Outputs: nil, WantErr: ErrNoMarker},
		{Case: "v1_plus_non_canonical_candidate", Notes: "A non-canonical push is an invalid candidate, not an ignored output, so it feeds the exactly-one-marker count like any other invalid candidate", Outputs: [][]byte{v1, nonCanonical}, WantErr: ErrMultipleMarkers},
		{Case: "lone_non_canonical_candidate", Notes: "a lone invalid candidate reports its own error; it does not degrade to no_marker", Outputs: [][]byte{nonCanonical}, WantErr: ErrNonCanonicalPush},
		{Case: "v1_plus_zero_address_candidate", Outputs: [][]byte{v1, zeroAddress}, WantErr: ErrMultipleMarkers},
	}
}

func TestScanVectors(t *testing.T) {
	for _, spec := range scanSpecs(t) {
		t.Run(spec.Case, func(t *testing.T) {
			got, err := ScanOutputs(spec.Outputs)
			if spec.WantOK {
				if err != nil {
					t.Fatalf("ScanOutputs: unexpected err %v", err)
				}
				if got != spec.Want {
					t.Fatalf("ScanOutputs = %+v, want %+v", got, spec.Want)
				}
				return
			}
			if err != spec.WantErr {
				t.Fatalf("ScanOutputs: err = %v, want %v", err, spec.WantErr)
			}
		})
	}
}

// ---- vectors.json driver ----------------------------------------------------

func vectorsPath(t *testing.T) string {
	t.Helper()
	return filepath.Join("testdata", "vectors.json")
}

// buildVectorsFile assembles the full vectorsFile from the same specs the
// hand-written tests above already validated against the real code.
func buildVectorsFile(t *testing.T) vectorsFile {
	vf := vectorsFile{
		Format:               "yellow-custody-btc-deposit-marker",
		FormatVersion:        1,
		ADR:                  "custody docs/decisions/adr-023-generic-btc-deposit-address.md",
		Notes:                "GENERATED FILE - do not edit by hand. It is produced from the spec tables in vectors_test.go by `go test ./pkg/blockchain/btc/marker/ -update`, and TestVectorsFile fails CI when the two drift apart, so a hand-edit here is reverted on the next regeneration. Change the Go source instead. Language-neutral conformance vectors. Any implementation of the ADR-023 deposit marker MUST reproduce every encode case byte-for-byte and MUST classify every decode and scan case identically, including the error code. A change to this file is a FORMAT change: bump formatVersion. Two field-level rules: referenceHex is present only for version 0x02 -- version 0x01 carries no reference on the wire, and an all-zero one is what makes a 0x02 marker invalid, so the zero-reference rule MUST be conditioned on the version. And a scan case is a list of scriptPubKeys, not of outputs: output value is not an input to recognition, so it is absent from this contract entirely. An implementation whose own scan function happens to receive whole outputs MUST ignore their values - a transaction carrying one zero-value marker and one 1-sat marker is unattributed, not attributed.",
		ValidationOrderNotes: "validationOrder is the fixed precedence for decode failures. A scriptPubKey failing several checks reports the earliest listed. 'not_marker' means the scriptPubKey is ignored by the attribution rule; every later code means it is a marker candidate and its transaction is unattributed.",
		GenericTag: genericTagJSON{
			Preimage:  GenericDepositTagPreimage,
			SHA256Hex: GenericDepositTagHex,
		},
		ErrorCodes: []string{
			"not_marker",
			"multiple_pushes", "non_canonical_push", "unknown_version",
			"bad_length", "zero_address", "zero_reference",
			"no_marker", "multiple_markers",
		},
		ValidationOrder: []string{
			"not_marker", "multiple_pushes", "non_canonical_push",
			"unknown_version", "bad_length", "zero_address", "zero_reference",
		},
	}

	for _, spec := range encodeSpecs(t) {
		ec := encodeCaseJSON{
			Case:  spec.Case,
			Notes: spec.Notes,
			Marker: markerJSON{
				Version:      spec.Marker.Version,
				AddressHex:   hexEnc(spec.Marker.Address[:]),
				ReferenceHex: spec.RefHexPtr,
			},
		}
		if spec.WantErr != nil {
			ec.ExpectError = errCode(t, spec.WantErr)
		} else {
			payload, err := EncodePayload(spec.Marker)
			if err != nil {
				t.Fatalf("%s: EncodePayload: %v", spec.Case, err)
			}
			script, err := EncodeScript(spec.Marker)
			if err != nil {
				t.Fatalf("%s: EncodeScript: %v", spec.Case, err)
			}
			ec.PayloadHex = hexEnc(payload)
			ec.ScriptHex = hexEnc(script)
		}
		vf.Encode = append(vf.Encode, ec)
	}

	for _, spec := range decodeSpecs(t) {
		dc := decodeCaseJSON{
			Case:      spec.Case,
			Notes:     spec.Notes,
			ScriptHex: hexEnc(spec.Script),
		}
		if spec.WantOK {
			dc.Expect = decodeExpectJSON{
				OK:           true,
				Version:      spec.Want.Version,
				AddressHex:   hexEnc(spec.Want.Address[:]),
				ReferenceHex: refHexFor(spec.Want),
			}
		} else {
			dc.Expect = decodeExpectJSON{OK: false, Error: errCode(t, spec.WantErr)}
		}
		vf.Decode = append(vf.Decode, dc)
	}

	for _, spec := range scanSpecs(t) {
		// ScriptPubKeysHex is initialised non-nil so an empty list serialises
		// as [] rather than null: vectors.json is consumed by non-Go
		// implementations, and a null where an array is declared is a
		// gratuitous special case for every one of them.
		sc := scanCaseJSON{Case: spec.Case, Notes: spec.Notes, ScriptPubKeysHex: []string{}}
		for _, spk := range spec.Outputs {
			sc.ScriptPubKeysHex = append(sc.ScriptPubKeysHex, hexEnc(spk))
		}
		if spec.WantOK {
			sc.Expect = decodeExpectJSON{
				OK:           true,
				Version:      spec.Want.Version,
				AddressHex:   hexEnc(spec.Want.Address[:]),
				ReferenceHex: refHexFor(spec.Want),
			}
		} else {
			sc.Expect = decodeExpectJSON{OK: false, Error: errCode(t, spec.WantErr)}
		}
		vf.Scan = append(vf.Scan, sc)
	}

	return vf
}

// refHexFor returns the reference hex for a decoded marker, or nil when the
// version has no reference field on the wire.
//
// Version1 carries no reference, so its expectation must not name one. Emitting
// the zero value here would put 64 zero bytes in a v1 success case - byte-identical
// to the value that makes a v2 marker invalid - so a mirror applying the
// zero-reference rule without conditioning on the version would contradict its
// own conformance cases. The encode section already represents "no reference"
// as null; this keeps the decode and scan sections consistent with it.
func refHexFor(m Marker) *string {
	if m.Version != Version2 {
		return nil
	}
	h := hexEnc(m.Reference[:])
	return &h
}

func hexEnc(b []byte) string {
	const hexdigits = "0123456789abcdef"
	out := make([]byte, len(b)*2)
	for i, v := range b {
		out[i*2] = hexdigits[v>>4]
		out[i*2+1] = hexdigits[v&0x0f]
	}
	return string(out)
}

// TestVectorsFile is the golden-file driver: with -update it (re)writes
// testdata/vectors.json; otherwise it asserts the file already on disk is
// byte-identical to what the current code produces.
func TestVectorsFile(t *testing.T) {
	vf := buildVectorsFile(t)

	want, err := json.MarshalIndent(vf, "", "  ")
	if err != nil {
		t.Fatalf("marshal vectors: %v", err)
	}
	want = append(want, '\n')

	path := vectorsPath(t)
	if *vectorsUpdate {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(path, want, 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
		return
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v -- run `go test -update` to regenerate", path, err)
	}
	if string(got) != string(want) {
		t.Fatalf("%s is stale -- run `go test -update` to regenerate", path)
	}
}
