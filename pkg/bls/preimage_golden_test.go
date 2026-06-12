package bls

// Byte-exact golden vectors for the BLS G1/G2 preimage surfaces that are pinned
// to Solidity verifiers (contracts/evm/src/BLS.sol and Slasher.sol). See
// docs/plans/cbor-encoding.md §8.4 and ADR-009 (Q8): these bytes MUST NOT drift
// without a coordinated on-chain upgrade.
//
// Layout captured here:
//   - G1 serialization: 64 bytes = X(32) || Y(32), big-endian.
//   - G2 serialization: 128 bytes = X.A1(im,32) || X.A0(re,32) || Y.A1(im,32) || Y.A0(re,32).
//     Matches BLS.sol's pairing-precompile input order (EIP-197, imaginary first).
//   - aggregate_sig: two partial signatures aggregated on G1 plus the matching
//     aggregate G2 pubkey, packed as G1(64) || G2(128) = 192 bytes.
//
// To regenerate fixtures after a legitimate (coordinated) change, run:
//   go test ./pkg/bls/ -run TestGoldens_Preimages -update
// Then inspect & commit the diff.

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"flag"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/consensys/gnark-crypto/ecc/bn254"
)

var updateGoldens = flag.Bool("update", false, "regenerate golden fixtures from current Go output")

// fixtureRoot returns the repo-root testdata directory. Go runs tests with the
// package dir as CWD, so we walk up until we find the repo root (directory
// containing testdata/goldens/).
func fixtureRoot(t *testing.T) string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	dir := cwd
	for i := 0; i < 8; i++ {
		cand := filepath.Join(dir, "testdata", "goldens", "solidity-preimages")
		if st, err := os.Stat(cand); err == nil && st.IsDir() {
			return cand
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatalf("could not locate testdata/goldens/solidity-preimages from %s", cwd)
	return ""
}

// writeOrCompare writes (when -update) or compares (default) a golden hex
// fixture plus its human-readable input JSON.
func writeOrCompare(t *testing.T, base string, inputJSON []byte, goldenBytes []byte) {
	t.Helper()
	hexPath := base + ".golden.hex"
	jsonPath := base + ".input.json"
	wantHex := strings.ToLower(hex.EncodeToString(goldenBytes))

	if *updateGoldens {
		if err := os.WriteFile(jsonPath, append(inputJSON, '\n'), 0o644); err != nil {
			t.Fatalf("write input: %v", err)
		}
		if err := os.WriteFile(hexPath, []byte(wantHex+"\n"), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		return
	}

	raw, err := os.ReadFile(hexPath)
	if err != nil {
		t.Fatalf("read golden %s: %v (re-run with -update to create)", hexPath, err)
	}
	got := strings.TrimSpace(string(raw))
	if got != wantHex {
		t.Fatalf("preimage drift at %s:\n  want (current Go): %s\n  have (on disk):    %s\n"+
			"If this change is intentional and coordinated with Solidity verifiers, "+
			"re-run with -update and commit the diff.", hexPath, wantHex, got)
	}

	// Sanity: input.json must also exist so future readers see the literal inputs.
	if _, err := os.Stat(jsonPath); err != nil {
		t.Fatalf("missing input fixture %s: %v", jsonPath, err)
	}
}

// --- helpers to build the three canonical points ---

// g1Identity is the BN254 G1 point at infinity. Our SerializeG1 writes (0,0) for it.
func g1Identity() bn254.G1Affine {
	var p bn254.G1Affine // zero value = point at infinity
	return p
}

// g2Identity is the BN254 G2 point at infinity.
func g2Identity() bn254.G2Affine {
	var p bn254.G2Affine
	return p
}

// g1Generator returns the fixed G1 generator (1, 2) — matches BLS.sol G1_X/G1_Y.
func g1Generator() bn254.G1Affine {
	_, _, g1Gen, _ := bn254.Generators()
	return g1Gen
}

// g2Generator returns the standard BN254 G2 generator — matches BLS.sol constants.
func g2Generator() bn254.G2Affine {
	_, _, _, g2Gen := bn254.Generators()
	return g2Gen
}

// keypairFromSeed returns a deterministic BLS key pair from a literal byte seed.
// Uses the production KeyPairFromSeed so the scalar derivation is exactly the
// same as clearnode uses at runtime.
func keypairFromSeed(seed string) *KeyPair {
	return KeyPairFromSeed([]byte(seed))
}

// --- test ---

type g1Vector struct {
	Name string `json:"name"`
	Kind string `json:"kind"` // identity | generator | scalarMult
	Seed string `json:"seed,omitempty"`
	// When Kind == scalarMult, we derive scalar from KeyPairFromSeed(seed).Secret
	// and multiply the fixed G1 generator. We record the resulting X/Y in the
	// JSON for human inspection; they are *outputs*, not inputs.
	DerivedX string `json:"derivedX,omitempty"`
	DerivedY string `json:"derivedY,omitempty"`
}

type g2Vector struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"` // identity | generator | scalarMult
	Seed      string `json:"seed,omitempty"`
	DerivedXI string `json:"derivedXIm,omitempty"`
	DerivedXR string `json:"derivedXRe,omitempty"`
	DerivedYI string `json:"derivedYIm,omitempty"`
	DerivedYR string `json:"derivedYRe,omitempty"`
}

type aggVector struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	MsgHashHex   string `json:"msgHashHex"`
	SeedA        string `json:"signerASeed"`
	SeedB        string `json:"signerBSeed"`
	SigmaAHex    string `json:"partialSigmaAHex"`
	SigmaBHex    string `json:"partialSigmaBHex"`
	AggSigHex    string `json:"aggregateSigmaHex"`
	AggApkG2Hex  string `json:"aggregateApkG2Hex"`
	PayloadOrder string `json:"payloadOrder"`
}

func TestGoldens_Preimages(t *testing.T) {
	root := fixtureRoot(t)
	blsDir := filepath.Join(root, "bls")

	// --- G1 points ---
	g1Cases := []struct {
		name string
		pt   bn254.G1Affine
		meta g1Vector
	}{
		{
			name: "g1_identity",
			pt:   g1Identity(),
			meta: g1Vector{Name: "g1_identity", Kind: "identity"},
		},
		{
			name: "g1_generator",
			pt:   g1Generator(),
			meta: g1Vector{Name: "g1_generator", Kind: "generator"},
		},
	}
	// Random (deterministic) G1: scalar-mult of G1 generator by KeyPairFromSeed.
	randKp := keypairFromSeed("clearnet/cbor-w0/bls/g1_random")
	var randX, randY big.Int
	randKp.PublicG1.X.BigInt(&randX)
	randKp.PublicG1.Y.BigInt(&randY)
	g1Cases = append(g1Cases, struct {
		name string
		pt   bn254.G1Affine
		meta g1Vector
	}{
		name: "g1_random",
		pt:   randKp.PublicG1,
		meta: g1Vector{
			Name:     "g1_random",
			Kind:     "scalarMult",
			Seed:     "clearnet/cbor-w0/bls/g1_random",
			DerivedX: randX.String(),
			DerivedY: randY.String(),
		},
	})

	for _, c := range g1Cases {
		t.Run(c.name, func(t *testing.T) {
			b := SerializeG1(c.pt)
			if len(b) != 64 {
				t.Fatalf("SerializeG1 produced %d bytes, want 64", len(b))
			}
			js, _ := json.MarshalIndent(c.meta, "", "  ")
			writeOrCompare(t, filepath.Join(blsDir, c.name), js, b)
		})
	}

	// --- G2 points ---
	g2Cases := []struct {
		name string
		pt   bn254.G2Affine
		meta g2Vector
	}{
		{
			name: "g2_identity",
			pt:   g2Identity(),
			meta: g2Vector{Name: "g2_identity", Kind: "identity"},
		},
		{
			name: "g2_generator",
			pt:   g2Generator(),
			meta: g2Vector{Name: "g2_generator", Kind: "generator"},
		},
	}
	randKp2 := keypairFromSeed("clearnet/cbor-w0/bls/g2_random")
	var g2XI, g2XR, g2YI, g2YR big.Int
	randKp2.PublicG2.X.A1.BigInt(&g2XI)
	randKp2.PublicG2.X.A0.BigInt(&g2XR)
	randKp2.PublicG2.Y.A1.BigInt(&g2YI)
	randKp2.PublicG2.Y.A0.BigInt(&g2YR)
	g2Cases = append(g2Cases, struct {
		name string
		pt   bn254.G2Affine
		meta g2Vector
	}{
		name: "g2_random",
		pt:   randKp2.PublicG2,
		meta: g2Vector{
			Name:      "g2_random",
			Kind:      "scalarMult",
			Seed:      "clearnet/cbor-w0/bls/g2_random",
			DerivedXI: g2XI.String(),
			DerivedXR: g2XR.String(),
			DerivedYI: g2YI.String(),
			DerivedYR: g2YR.String(),
		},
	})

	for _, c := range g2Cases {
		t.Run(c.name, func(t *testing.T) {
			b := SerializeG2(c.pt)
			if len(b) != 128 {
				t.Fatalf("SerializeG2 produced %d bytes, want 128", len(b))
			}
			js, _ := json.MarshalIndent(c.meta, "", "  ")
			writeOrCompare(t, filepath.Join(blsDir, c.name), js, b)
		})
	}

	// --- Aggregate signature over a deterministic message ---
	t.Run("aggregate_sig", func(t *testing.T) {
		var msgHash [32]byte
		// Literal 32-byte message: keccak of "clearnet/cbor-w0/bls/agg".
		// We use a hand-picked hex string so the input is fully literal.
		msgHex := "6d91a2b8e2f9c0d1a3b4c5d6e7f8091a2b3c4d5e6f70819a2b3c4d5e6f708192"
		raw, err := hex.DecodeString(msgHex)
		if err != nil || len(raw) != 32 {
			t.Fatalf("bad msg hex")
		}
		copy(msgHash[:], raw)

		kpA := keypairFromSeed("clearnet/cbor-w0/bls/agg/A")
		kpB := keypairFromSeed("clearnet/cbor-w0/bls/agg/B")

		sigA, err := Sign(&kpA.Secret, msgHash)
		if err != nil {
			t.Fatalf("sign A: %v", err)
		}
		sigB, err := Sign(&kpB.Secret, msgHash)
		if err != nil {
			t.Fatalf("sign B: %v", err)
		}
		aggSig, err := AggregateG1([]bn254.G1Affine{sigA, sigB})
		if err != nil {
			t.Fatalf("aggregate G1: %v", err)
		}
		aggApk, err := AggregateG2([]bn254.G2Affine{kpA.PublicG2, kpB.PublicG2})
		if err != nil {
			t.Fatalf("aggregate G2: %v", err)
		}

		// Verify the aggregate just to catch accidental bit-rot in production code
		// that a pure byte comparison would miss.
		ok, err := Verify(aggSig, aggApk, msgHash)
		if err != nil || !ok {
			t.Fatalf("aggregate verify failed: ok=%v err=%v", ok, err)
		}

		sigBytes := SerializeG1(aggSig)
		apkBytes := SerializeG2(aggApk)
		payload := append(append([]byte{}, sigBytes...), apkBytes...)
		if len(payload) != 192 {
			t.Fatalf("payload len = %d, want 192", len(payload))
		}

		meta := aggVector{
			Name:         "aggregate_sig",
			Description:  "Two partial signatures over the literal 32-byte msgHash, aggregated via AggregateG1; aggregate G2 pubkey via AggregateG2. Layout: sigma(G1,64B) || apkG2(128B).",
			MsgHashHex:   msgHex,
			SeedA:        "clearnet/cbor-w0/bls/agg/A",
			SeedB:        "clearnet/cbor-w0/bls/agg/B",
			SigmaAHex:    hex.EncodeToString(SerializeG1(sigA)),
			SigmaBHex:    hex.EncodeToString(SerializeG1(sigB)),
			AggSigHex:    hex.EncodeToString(sigBytes),
			AggApkG2Hex:  hex.EncodeToString(apkBytes),
			PayloadOrder: "aggSigmaG1(64) || aggApkG2(128)",
		}
		js, _ := json.MarshalIndent(meta, "", "  ")
		writeOrCompare(t, filepath.Join(blsDir, "aggregate_sig"), js, payload)
	})
}

// Round-trip sanity: every G1/G2 golden must deserialize back to the exact
// point the generator produced. This catches the case where someone changes
// SerializeG1/G2 byte order but updates the fixtures without noticing.
func TestGoldens_Preimages_RoundTrip(t *testing.T) {
	root := fixtureRoot(t)
	for _, name := range []string{"g1_identity", "g1_generator", "g1_random"} {
		hexBytes, err := os.ReadFile(filepath.Join(root, "bls", name+".golden.hex"))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		raw, err := hex.DecodeString(strings.TrimSpace(string(hexBytes)))
		if err != nil {
			t.Fatalf("decode hex: %v", err)
		}
		pt, err := DeserializeG1(raw)
		if err != nil {
			t.Fatalf("DeserializeG1(%s): %v", name, err)
		}
		if !bytes.Equal(SerializeG1(pt), raw) {
			t.Fatalf("%s: round-trip serialize mismatch", name)
		}
	}
	for _, name := range []string{"g2_identity", "g2_generator", "g2_random"} {
		hexBytes, err := os.ReadFile(filepath.Join(root, "bls", name+".golden.hex"))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		raw, err := hex.DecodeString(strings.TrimSpace(string(hexBytes)))
		if err != nil {
			t.Fatalf("decode hex: %v", err)
		}
		pt, err := DeserializeG2(raw)
		if err != nil {
			t.Fatalf("DeserializeG2(%s): %v", name, err)
		}
		if !bytes.Equal(SerializeG2(pt), raw) {
			t.Fatalf("%s: round-trip serialize mismatch", name)
		}
	}
}
