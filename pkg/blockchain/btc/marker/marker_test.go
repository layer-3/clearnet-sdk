package marker

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"testing"
)

// ---- test helpers -------------------------------------------------------

const testAddrHex = "d8da6bf26964af9d7eed9e03e53415d37aa96045"

func mustAddr(t *testing.T, h string) [20]byte {
	t.Helper()
	b, err := hex.DecodeString(h)
	if err != nil || len(b) != 20 {
		t.Fatalf("bad test address hex %q: %v", h, err)
	}
	var a [20]byte
	copy(a[:], b)
	return a
}

func mustRef(t *testing.T, h string) [32]byte {
	t.Helper()
	b, err := hex.DecodeString(h)
	if err != nil || len(b) != 32 {
		t.Fatalf("bad test reference hex %q: %v", h, err)
	}
	var r [32]byte
	copy(r[:], b)
	return r
}

// rawDirectPush returns OP_RETURN followed by a canonical direct-push
// encoding of payload (opcode == len(payload), valid for 0 <= len <= 75).
func rawDirectPush(payload []byte) []byte {
	out := []byte{opReturn, byte(len(payload))}
	return append(out, payload...)
}

// rawPushData1 returns OP_RETURN followed by an OP_PUSHDATA1 encoding.
func rawPushData1(payload []byte) []byte {
	out := []byte{opReturn, opPushData1, byte(len(payload))}
	return append(out, payload...)
}

// rawPushData2 returns OP_RETURN followed by an OP_PUSHDATA2 encoding.
func rawPushData2(payload []byte) []byte {
	out := []byte{opReturn, opPushData2, 0, 0}
	binary.LittleEndian.PutUint16(out[2:4], uint16(len(payload)))
	return append(out, payload...)
}

// rawPushData4 returns OP_RETURN followed by an OP_PUSHDATA4 encoding.
func rawPushData4(payload []byte) []byte {
	out := []byte{opReturn, opPushData4, 0, 0, 0, 0}
	binary.LittleEndian.PutUint32(out[2:6], uint32(len(payload)))
	return append(out, payload...)
}

func v1Payload(t *testing.T, addrHex string) []byte {
	t.Helper()
	addr := mustAddr(t, addrHex)
	p, err := EncodePayload(Marker{Version: Version1, Address: addr})
	if err != nil {
		t.Fatalf("EncodePayload(v1): %v", err)
	}
	return p
}

func v2Payload(t *testing.T, addrHex, refHex string) []byte {
	t.Helper()
	addr := mustAddr(t, addrHex)
	ref := mustRef(t, refHex)
	p, err := EncodePayload(Marker{Version: Version2, Address: addr, Reference: ref})
	if err != nil {
		t.Fatalf("EncodePayload(v2): %v", err)
	}
	return p
}

// rawPayloadV1 / rawPayloadV2 build wire payloads directly, bypassing
// EncodePayload's own validation. They exist to construct the deliberately
// invalid vectors.
func rawPayloadV1(t *testing.T, addrHex string) []byte {
	t.Helper()
	addr := mustAddr(t, addrHex)
	p := append([]byte{}, Magic...)
	p = append(p, Version1)
	p = append(p, addr[:]...)
	return p
}

func rawPayloadV2(t *testing.T, addrHex, refHex string) []byte {
	t.Helper()
	addr := mustAddr(t, addrHex)
	ref := mustRef(t, refHex)
	p := append([]byte{}, Magic...)
	p = append(p, Version2)
	p = append(p, addr[:]...)
	p = append(p, ref[:]...)
	return p
}

// ---- Encode + decode round trips -------------------------------------------

func TestEncodeDecodeV1(t *testing.T) {
	addr := mustAddr(t, testAddrHex)
	m := Marker{Version: Version1, Address: addr}

	payload, err := EncodePayload(m)
	if err != nil {
		t.Fatalf("EncodePayload: %v", err)
	}
	if len(payload) != PayloadLenV1 {
		t.Fatalf("payload length = %d, want %d", len(payload), PayloadLenV1)
	}
	wantPayload := "594e455401d8da6bf26964af9d7eed9e03e53415d37aa96045"
	if hex.EncodeToString(payload) != wantPayload {
		t.Fatalf("payload = %x, want %s", payload, wantPayload)
	}

	script, err := EncodeScript(m)
	if err != nil {
		t.Fatalf("EncodeScript: %v", err)
	}
	wantScript := "6a19594e455401d8da6bf26964af9d7eed9e03e53415d37aa96045"
	if hex.EncodeToString(script) != wantScript {
		t.Fatalf("script = %x, want %s", script, wantScript)
	}

	got, err := DecodeScript(script)
	if err != nil {
		t.Fatalf("DecodeScript: %v", err)
	}
	if got != m {
		t.Fatalf("DecodeScript(EncodeScript(m)) = %+v, want %+v", got, m)
	}
}

func TestEncodeDecodeV2(t *testing.T) {
	addr := mustAddr(t, testAddrHex)
	ref := mustRef(t, "000000000000000000000000000000000000000000000000000000000000002a")
	m := Marker{Version: Version2, Address: addr, Reference: ref}

	payload, err := EncodePayload(m)
	if err != nil {
		t.Fatalf("EncodePayload: %v", err)
	}
	if len(payload) != PayloadLenV2 {
		t.Fatalf("payload length = %d, want %d", len(payload), PayloadLenV2)
	}

	script, err := EncodeScript(m)
	if err != nil {
		t.Fatalf("EncodeScript: %v", err)
	}
	wantScript := "6a39594e455402d8da6bf26964af9d7eed9e03e53415d37aa96045" +
		"000000000000000000000000000000000000000000000000000000000000002a"
	if hex.EncodeToString(script) != wantScript {
		t.Fatalf("script = %x, want %s", script, wantScript)
	}

	got, err := DecodeScript(script)
	if err != nil {
		t.Fatalf("DecodeScript: %v", err)
	}
	if got != m {
		t.Fatalf("DecodeScript(EncodeScript(m)) = %+v, want %+v", got, m)
	}
}

// ---- The two all-zero rejections, both directions --------------------------

func TestZeroReferenceRejected(t *testing.T) {
	addr := mustAddr(t, testAddrHex)
	var zeroRef [32]byte
	m := Marker{Version: Version2, Address: addr, Reference: zeroRef}

	if _, err := EncodePayload(m); !errors.Is(err, ErrZeroReference) {
		t.Fatalf("EncodePayload: err = %v, want ErrZeroReference", err)
	}

	script := rawDirectPush(rawPayloadV2(t, testAddrHex, "0000000000000000000000000000000000000000000000000000000000000000"))
	if _, err := DecodeScript(script); !errors.Is(err, ErrZeroReference) {
		t.Fatalf("DecodeScript: err = %v, want ErrZeroReference", err)
	}
}

func TestZeroAddressRejected(t *testing.T) {
	var zeroAddr [20]byte
	nonZeroRef := mustRef(t, "000000000000000000000000000000000000000000000000000000000000002a")

	for _, m := range []Marker{
		{Version: Version1, Address: zeroAddr},
		{Version: Version2, Address: zeroAddr, Reference: nonZeroRef},
	} {
		if _, err := EncodePayload(m); !errors.Is(err, ErrZeroAddress) {
			t.Fatalf("EncodePayload(v%d): err = %v, want ErrZeroAddress", m.Version, err)
		}
	}

	v1Script := rawDirectPush(rawPayloadV1(t, "0000000000000000000000000000000000000000"))
	if _, err := DecodeScript(v1Script); !errors.Is(err, ErrZeroAddress) {
		t.Fatalf("DecodeScript(v1 zero addr): err = %v, want ErrZeroAddress", err)
	}

	v2Script := rawDirectPush(rawPayloadV2(t, "0000000000000000000000000000000000000000", "000000000000000000000000000000000000000000000000000000000000002a"))
	if _, err := DecodeScript(v2Script); !errors.Is(err, ErrZeroAddress) {
		t.Fatalf("DecodeScript(v2 zero addr): err = %v, want ErrZeroAddress", err)
	}
}

// ---- Malformed candidates ---------------------------------------------------

func TestDecodeMalformedCandidates(t *testing.T) {
	v1p := v1Payload(t, testAddrHex)
	v2p := v2Payload(t, testAddrHex, "000000000000000000000000000000000000000000000000000000000000002a")

	cases := []struct {
		name   string
		script []byte
		want   error
	}{
		// version 0x03 on a 25-byte payload -> unknown_version.
		{"unknown_version_25", rawDirectPush(withVersion(v1p, 0x03)), ErrUnknownVersion},
		// version 0x03 on a 57-byte payload -> unknown_version, not bad_length.
		{"unknown_version_57", rawDirectPush(withVersion(v2p, 0x03)), ErrUnknownVersion},
		// version 0x00 -> unknown_version.
		{"unknown_version_00", rawDirectPush(withVersion(v1p, 0x00)), ErrUnknownVersion},
		// v1, one byte short -> bad_length, regardless of truncated address bytes.
		{"v1_one_short", rawDirectPush(v1p[:len(v1p)-1]), ErrBadLength},
		// v1, one byte long -> bad_length.
		{"v1_one_long", rawDirectPush(append(append([]byte{}, v1p...), 0x00)), ErrBadLength},
		// v2, one byte short -> bad_length.
		{"v2_one_short", rawDirectPush(v2p[:len(v2p)-1]), ErrBadLength},
		// payload is exactly "YNET" (4 bytes, no version byte) -> bad_length.
		{"magic_only", rawDirectPush([]byte(Magic)), ErrBadLength},
		// v2 all-zero reference -> zero_reference (also covered above; kept for table completeness).
		{"v2_zero_reference", rawDirectPush(rawPayloadV2(t, testAddrHex, "0000000000000000000000000000000000000000000000000000000000000000")), ErrZeroReference},
		// OP_RETURN + valid v1 push + a second 4-byte push -> multiple_pushes.
		{"multiple_pushes", append(rawDirectPush(v1p), rawDirectPush([]byte{1, 2, 3, 4})[1:]...), ErrMultiplePushes},
		// v1 payload wrapped as OP_PUSHDATA1 -> non_canonical_push.
		{"pushdata1", rawPushData1(v1p), ErrNonCanonicalPush},
		// v1 payload wrapped as OP_PUSHDATA2 -> non_canonical_push.
		{"pushdata2", rawPushData2(v1p), ErrNonCanonicalPush},
		// v2 payload wrapped as OP_PUSHDATA4 -> non_canonical_push.
		{"pushdata4", rawPushData4(v2p), ErrNonCanonicalPush},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := DecodeScript(c.script)
			if !errors.Is(err, c.want) {
				t.Fatalf("DecodeScript(%x) = %v, want %v", c.script, err, c.want)
			}
		})
	}
}

// withVersion returns a copy of payload with its version byte overwritten.
func withVersion(payload []byte, v byte) []byte {
	out := append([]byte{}, payload...)
	out[len(Magic)] = v
	return out
}

// ---- Not-a-candidate ---------------------------------------------------------

func TestDecodeNotCandidate(t *testing.T) {
	v1p := v1Payload(t, testAddrHex)

	cases := []struct {
		name   string
		script []byte
	}{
		// wrong magic, otherwise valid v1 shape.
		{"wrong_magic", rawDirectPush(withMagic(v1p, "YNEU"))},
		// P2WSH scriptPubKey, not OP_RETURN.
		{"p2wsh", append([]byte{0x00, 0x20}, bytes.Repeat([]byte{0xaa}, 32)...)},
		// bare OP_RETURN, no push.
		{"bare_op_return", []byte{opReturn}},
		// the withdrawal-ID vector -- a 32-byte push, no magic.
		{"withdrawal_id", rawDirectPush(bytes.Repeat([]byte{0x01}, 32))},
		// payload "YNE" (3 bytes, below magic length).
		{"below_magic_length", rawDirectPush([]byte("YNE"))},
		// empty script.
		{"empty", []byte{}},
		// arbitrary (P2PKH-shaped) script.
		{"p2pkh", []byte{0x76, 0xa9, 0x14, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x88, 0xac}},
		// OP_PUSHDATA1-wrapped payload whose magic does NOT match - ignored,
		// not non_canonical_push.
		{"pushdata1_wrong_magic", rawPushData1(withMagic(v1p, "YNEU"))},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := DecodeScript(c.script)
			if !errors.Is(err, ErrNotMarker) {
				t.Fatalf("DecodeScript(%x) = %v, want ErrNotMarker", c.script, err)
			}
		})
	}
}

// withMagic returns a copy of payload with its 4-byte magic overwritten.
func withMagic(payload []byte, magic string) []byte {
	out := append([]byte{}, payload...)
	copy(out[:len(Magic)], magic)
	return out
}

// TestPushEncodingJudgedOnlyAfterMagicMatches: identical OP_PUSHDATA1
// wrapping, opposite classification, differing only in whether the magic
// matches - proof that canonicality is judged only after the magic test.
func TestPushEncodingJudgedOnlyAfterMagicMatches(t *testing.T) {
	v1p := v1Payload(t, testAddrHex)

	_, err := DecodeScript(rawPushData1(v1p))
	if !errors.Is(err, ErrNonCanonicalPush) {
		t.Fatalf("magic matches: err = %v, want ErrNonCanonicalPush", err)
	}

	_, err = DecodeScript(rawPushData1(withMagic(v1p, "YNEU")))
	if !errors.Is(err, ErrNotMarker) {
		t.Fatalf("magic does not match: err = %v, want ErrNotMarker", err)
	}
}

// ---- Validation precedence ---------------------------------------------------

// withSecondPush appends a second, 4-byte data push to an OP_RETURN script,
// copying rather than aliasing so the caller's script is untouched.
func withSecondPush(script []byte) []byte {
	out := append([]byte{}, script...)
	return append(out, 0x04, 1, 2, 3, 4)
}

// TestValidationPrecedence pins the fixed order in which decode checks run:
// one case per ordered pair of failures that can co-occur, asserting the
// earlier-listed check wins.
//
// The order under test is:
//
//	not_marker  <  multiple_pushes  <  non_canonical_push  <
//	unknown_version  <  bad_length  <  zero_address  <  zero_reference
//
// A script that is not an OP_RETURN, or that carries no push at all, cannot
// co-occur with anything later in this order - there is no payload left to
// fail a later check - so every remaining co-occurring pair is covered
// below. The implementation is a chain of early returns, so adjacent pairs
// would be transitively sufficient for this reader; the full matrix exists
// because the order is a cross-language contract that a mirror
// implementation may realise differently.
func TestValidationPrecedence(t *testing.T) {
	v1p := v1Payload(t, testAddrHex)

	// Payloads that individually trip exactly one of the later checks.
	unknownVersionP := withVersion(v1p, 0x03)
	badLengthP := v1p[:len(v1p)-1]
	zeroAddrP := rawPayloadV1(t, "0000000000000000000000000000000000000000")
	zeroRefP := rawPayloadV2(t, testAddrHex, "00000000000000000000000000000000"+
		"00000000000000000000000000000000")
	wrongMagicP := withMagic(v1p, "YNEU")

	cases := []struct {
		name   string
		script []byte
		want   error
	}{
		// A non-magic payload is ignored no matter how many pushes follow it -
		// the magic check runs on the payload before the multiple-pushes check.
		{"not_marker_over_multiple_pushes", withSecondPush(rawDirectPush(wrongMagicP)), ErrNotMarker},
		// The same script as TestPushEncodingJudgedOnlyAfterMagicMatches's
		// wrong-magic case: canonicality is judged only after the magic
		// matches, so this reports not-a-candidate rather than
		// non-canonical.
		{"not_marker_over_non_canonical", rawPushData1(wrongMagicP), ErrNotMarker},

		// Push count is judged before the encoding and before anything
		// inside the payload.
		{"multiple_pushes_over_non_canonical", withSecondPush(rawPushData1(v1p)), ErrMultiplePushes},
		{"multiple_pushes_over_unknown_version", withSecondPush(rawDirectPush(unknownVersionP)), ErrMultiplePushes},
		{"multiple_pushes_over_bad_length", withSecondPush(rawDirectPush(badLengthP)), ErrMultiplePushes},
		{"multiple_pushes_over_zero_address", withSecondPush(rawDirectPush(zeroAddrP)), ErrMultiplePushes},
		{"multiple_pushes_over_zero_reference", withSecondPush(rawDirectPush(zeroRefP)), ErrMultiplePushes},

		// The encoding is judged before the payload contents.
		{"non_canonical_over_unknown_version", rawPushData1(unknownVersionP), ErrNonCanonicalPush},
		{"non_canonical_over_bad_length", rawPushData1(badLengthP), ErrNonCanonicalPush},
		{"non_canonical_over_zero_address", rawPushData1(zeroAddrP), ErrNonCanonicalPush},
		{"non_canonical_over_zero_reference", rawPushData1(zeroRefP), ErrNonCanonicalPush},

		// The version is read before the length is checked and before any
		// field is interpreted.
		{"unknown_version_over_bad_length", rawDirectPush(append(append([]byte(Magic), 0x03), bytes.Repeat([]byte{0xff}, 10)...)), ErrUnknownVersion},
		{"unknown_version_over_zero_address", rawDirectPush(withVersion(zeroAddrP, 0x03)), ErrUnknownVersion},
		{"unknown_version_over_zero_reference", rawDirectPush(withVersion(zeroRefP, 0x03)), ErrUnknownVersion},

		// A short payload may not contain a whole field, so the zero checks
		// must never read past the end of it.
		{"bad_length_over_zero_address", rawDirectPush(append(append([]byte(Magic), Version1), make([]byte, 19)...)), ErrBadLength},
		{"bad_length_over_zero_reference", rawDirectPush(zeroRefP[:len(zeroRefP)-1]), ErrBadLength},

		// The address check runs before the reference check.
		{"zero_address_over_zero_reference", rawDirectPush(append(append([]byte(Magic), Version2), make([]byte, 20+32)...)), ErrZeroAddress},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := DecodeScript(c.script)
			if !errors.Is(err, c.want) {
				t.Fatalf("DecodeScript(%x) = %v, want %v", c.script, err, c.want)
			}
		})
	}
}

// TestTrailingBytesAreNotSilentlyAccepted pins the reading the reader
// implements: the check is "the single push must account for the whole
// script", not "a second well-formed push must follow".
//
// The distinction matters. Under the literal "exactly one push" reading, a
// valid marker with a trailing byte that is not a push opcode at all -
// 0x6a, say - would sail through to a successful decode, and an attacker
// could smuggle an attributed transaction past a mirror implementation that
// counts pushes rather than bytes. Reporting ErrMultiplePushes keeps such an
// scriptPubKey a candidate, so the transaction is held rather than credited.
//
// The same four scripts are in vectors.json as the trailing_* decode cases,
// so a mirror implementation is held to this reading too.
func TestTrailingBytesAreNotSilentlyAccepted(t *testing.T) {
	valid := rawDirectPush(v1Payload(t, testAddrHex))
	if _, err := DecodeScript(valid); err != nil {
		t.Fatalf("precondition: valid script must decode, got %v", err)
	}

	for _, trailing := range [][]byte{
		{opReturn},   // not a push opcode at all
		{0x51},       // OP_1
		{0x19},       // a push opcode whose declared data is absent
		{0x00, 0x00}, // OP_0 twice
	} {
		script := append(append([]byte{}, valid...), trailing...)
		if _, err := DecodeScript(script); !errors.Is(err, ErrMultiplePushes) {
			t.Fatalf("DecodeScript(%x) = %v, want ErrMultiplePushes", script, err)
		}
	}
}

// TestMagicIsTestedOnTheFirstPushOnly documents the implemented reading: the
// magic test runs on the first push only, not on any push in the script. An
// OP_RETURN whose first push is junk and whose SECOND push carries a
// well-formed marker payload is therefore NOT a candidate: it is ignored
// outright.
//
// The script below is also vectors.json's junk_push_before_marker_is_not_a_candidate
// case, so a mirror is held to the same classification.
func TestMagicIsTestedOnTheFirstPushOnly(t *testing.T) {
	v1p := v1Payload(t, testAddrHex)

	script := []byte{opReturn, 0x01, 0x00}
	script = append(script, rawDirectPush(v1p)[1:]...)

	if _, err := DecodeScript(script); !errors.Is(err, ErrNotMarker) {
		t.Fatalf("DecodeScript(%x) = %v, want ErrNotMarker", script, err)
	}
}

// ---- ScanOutputs --------------------------------------------------------------

func v1Script(t *testing.T, addrHex string) []byte {
	t.Helper()
	s, err := EncodeScript(Marker{Version: Version1, Address: mustAddr(t, addrHex)})
	if err != nil {
		t.Fatalf("EncodeScript: %v", err)
	}
	return s
}

func v2Script(t *testing.T, addrHex, refHex string) []byte {
	t.Helper()
	s, err := EncodeScript(Marker{Version: Version2, Address: mustAddr(t, addrHex), Reference: mustRef(t, refHex)})
	if err != nil {
		t.Fatalf("EncodeScript: %v", err)
	}
	return s
}

func TestScanOutputs(t *testing.T) {
	v1 := v1Script(t, testAddrHex)
	v2 := v2Script(t, testAddrHex, "000000000000000000000000000000000000000000000000000000000000002a")
	valueOut := []byte{0x00, 0x20} // stub non-OP_RETURN scriptPubKey for a value output
	withdrawalID := rawDirectPush(bytes.Repeat([]byte{0x01}, 32))
	unknownVersion := rawDirectPush(withVersion(v1Payload(t, testAddrHex), 0x03))
	nonCanonical := rawPushData1(v1Payload(t, testAddrHex))
	zeroAddress := rawDirectPush(rawPayloadV1(t, "0000000000000000000000000000000000000000"))

	wantV1 := Marker{Version: Version1, Address: mustAddr(t, testAddrHex)}

	cases := []struct {
		name    string
		outputs [][]byte
		want    Marker
		wantErr error
	}{
		{"S1_marker_plus_value", [][]byte{v1, valueOut}, wantV1, nil},
		{"S2_marker_plus_two_values", [][]byte{valueOut, v1, valueOut}, wantV1, nil},
		{"S3_values_only", [][]byte{valueOut, valueOut}, Marker{}, ErrNoMarker},
		{"S4_two_identical_v1", [][]byte{v1, v1}, Marker{}, ErrMultipleMarkers},
		{"S5_v1_and_v2", [][]byte{v1, v2}, Marker{}, ErrMultipleMarkers},
		{"S6_v1_and_withdrawal_id", [][]byte{v1, withdrawalID}, wantV1, nil},
		{"S7_v1_and_unknown_version", [][]byte{v1, unknownVersion}, Marker{}, ErrMultipleMarkers},
		{"S8_marker_at_index_2", [][]byte{valueOut, valueOut, v1}, wantV1, nil},
		{"S9_lone_marker_no_value_output", [][]byte{v1}, wantV1, nil},
		{"S10_empty", nil, Marker{}, ErrNoMarker},
		{"S11_v1_and_non_canonical", [][]byte{v1, nonCanonical}, Marker{}, ErrMultipleMarkers},
		{"S12_lone_non_canonical", [][]byte{nonCanonical}, Marker{}, ErrNonCanonicalPush},
		{"S13_v1_and_zero_address", [][]byte{v1, zeroAddress}, Marker{}, ErrMultipleMarkers},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ScanOutputs(c.outputs)
			if c.wantErr != nil {
				if !errors.Is(err, c.wantErr) {
					t.Fatalf("err = %v, want %v", err, c.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if got != c.want {
				t.Fatalf("got %+v, want %+v", got, c.want)
			}
		})
	}
}
