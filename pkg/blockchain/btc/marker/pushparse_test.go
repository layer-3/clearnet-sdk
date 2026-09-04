package marker

// This file cross-checks the hand-rolled push reader (decodePush, in
// marker.go) against btcd's txscript.PushedData. It exists to buy back
// btcd's battle-testing of push-opcode parsing without coupling the
// non-test code to it. txscript is a test-only import: it does not appear
// in `go list -deps` for the package itself.
//
// txscript.PushedData accepts non-canonical pushes (OP_PUSHDATA1/2/4) that
// decodePush classifies as non-canonical, so the two readers can and do
// disagree on the VERDICT for those cases by design. What must never
// disagree is the extracted PAYLOAD - comparing payloads, not verdicts, is
// therefore the whole point of this file.

import (
	"bytes"
	"math/rand"
	"testing"

	"github.com/btcsuite/btcd/txscript"
)

// btcdFirstPush runs txscript.PushedData over script and returns the first
// pushed data item, mirroring what decodePush extracts for the first push.
// txscript.PushedData tolerates the leading OP_RETURN (it simply is not a
// push opcode, so it contributes no output), so no stripping is needed.
func btcdFirstPush(script []byte) ([]byte, bool) {
	data, err := txscript.PushedData(script)
	if err != nil || len(data) == 0 {
		return nil, false
	}
	return data[0], true
}

// TestPushParseCrossCheckCorpus runs the cross-check over every push shape
// used in the vector corpus: canonical direct pushes and all three
// non-canonical wrappings, at both marker lengths.
func TestPushParseCrossCheckCorpus(t *testing.T) {
	v1p := v1Payload(t, testAddrHex)
	v2p := v2Payload(t, testAddrHex, "000000000000000000000000000000000000000000000000000000000000002a")

	scripts := [][]byte{
		rawDirectPush(v1p),
		rawDirectPush(v2p),
		rawPushData1(v1p),
		rawPushData2(v1p),
		rawPushData4(v2p),
		rawDirectPush(bytes.Repeat([]byte{0x01}, 32)), // the withdrawal-ID shape
		rawPushData1(withMagic(v1p, "YNEU")),          // non-canonical wrapping whose magic does not match -- ignored, not held
	}

	for _, script := range scripts {
		mine, mineOK := decodePush(script[1:])
		theirs, theirsOK := btcdFirstPush(script)
		if mineOK != theirsOK {
			t.Fatalf("script %x: decodePush ok=%v, txscript.PushedData ok=%v", script, mineOK, theirsOK)
		}
		if mineOK && !bytes.Equal(mine.payload, theirs) {
			t.Fatalf("script %x: payload mismatch\n  mine:  %x\n  theirs: %x", script, mine.payload, theirs)
		}
	}
}

// TestPushParseCrossCheckFuzz generates a fixed-seed population of random
// push encodings and asserts the two readers extract the same payload
// whenever both consider the input parseable.
func TestPushParseCrossCheckFuzz(t *testing.T) {
	const (
		seed       = 0x59a4c0de
		iterations = 2000
	)
	rng := rand.New(rand.NewSource(seed))

	for i := 0; i < iterations; i++ {
		payload := make([]byte, rng.Intn(80))
		rng.Read(payload)

		var script []byte
		switch rng.Intn(4) {
		case 0:
			script = rawDirectPushFuzz(payload)
		case 1:
			script = rawPushData1(payload)
		case 2:
			script = rawPushData2(payload)
		case 3:
			script = rawPushData4(payload)
		}

		mine, mineOK := decodePush(script[1:])
		theirs, theirsOK := btcdFirstPush(script)

		if !mineOK {
			// decodePush only accepts a direct push whose opcode fits
			// 0x01-0x4b; a random payload longer than 75 bytes deliberately
			// cannot be wrapped as a well-formed direct push by
			// rawDirectPushFuzz below (it clamps), so !mineOK here only
			// happens if the generator produced something decodePush
			// legitimately rejects. Nothing to compare in that case.
			continue
		}
		if !theirsOK {
			t.Fatalf("iter %d: decodePush accepted %x but txscript.PushedData rejected it", i, script)
		}
		if !bytes.Equal(mine.payload, theirs) {
			t.Fatalf("iter %d: payload mismatch on %x\n  mine:   %x\n  theirs: %x", i, script, mine.payload, theirs)
		}
	}
}

// rawDirectPushFuzz builds a canonical direct push, clamping to the 0-75
// byte range a direct push can represent (longer payloads are truncated,
// which is fine for this generator - it only needs *a* well-formed script,
// not a faithful encoding of the original random payload).
func rawDirectPushFuzz(payload []byte) []byte {
	if len(payload) > 75 {
		payload = payload[:75]
	}
	return rawDirectPush(payload)
}
