package cborx_test

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
	"time"

	cbor "github.com/fxamacker/cbor/v2"

	"github.com/layer-3/clearnet-sdk/pkg/cborx"
	"github.com/layer-3/clearnet-sdk/pkg/decimal"
)

// goldensUpdate writes new fixture files under testdata/cbor/primitives/.
// The flag is kept out of `go test` by default — CI treats a missing
// fixture as a failure, not a reason to re-seed. Run
//
//	go test ./internal/cborx/... -update -run TestGoldenFixtures
//
// after intentionally changing an adapter to regenerate.
var goldensUpdate = flag.Bool("update", false, "regenerate testdata/cbor/primitives/* fixtures")

// The fuzz seed is committed. A drift in this seed would re-seed every
// fixture and is a schema-level change, not a routine edit.
const fuzzSeed int64 = 0x0c801e115ca11baa

// Iteration counts for the determinism fuzz. 200 instances × 1000
// encodes per instance = 200,000 encode ops per adapter type, run in
// a randomized order to catch any map-iteration or pool-derived byte
// drift. CI bumps -count=10 for another 10× factor.
const (
	fuzzInstances  = 200
	fuzzIterations = 1000
)

// testdataDir returns the absolute path of testdata/cbor/primitives
// regardless of cwd (go test may run from the package dir or a symlink).
func testdataDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// file = .../internal/cborx/goldens_test.go
	repoRoot := filepath.Join(filepath.Dir(file), "..", "..")
	return filepath.Join(repoRoot, "testdata", "cbor", "primitives")
}

// fixture is one golden: a typed description of the input plus its
// expected canonical-CBOR bytes. The bytes are committed separately as
// `<name>_<case>.golden.hex` and the description as
// `<name>_<case>.input.json` so a human can eyeball what each byte
// string represents.
type fixture struct {
	Case   string `json:"case"`
	Notes  string `json:"notes,omitempty"`
	Input  any    `json:"input"`
	Golden string `json:"-"`
}

// ----- BigInt fixtures --------------------------------------------------

type bigIntInput struct {
	Hex  string `json:"hex"`
	Sign int    `json:"sign"`
}

func biFx(name, notes string, n *big.Int) fixture {
	var buf bytes.Buffer
	if err := (cborx.BigInt{V: n}).MarshalCBOR(&buf); err != nil {
		panic(err)
	}
	return fixture{
		Case:   name,
		Notes:  notes,
		Input:  bigIntInput{Hex: n.Text(16), Sign: n.Sign()},
		Golden: hex.EncodeToString(buf.Bytes()),
	}
}

func bigIntFixtures() []fixture {
	return []fixture{
		biFx("zero", "canonical tag 2 + empty byte string", big.NewInt(0)),
		biFx("one", "smallest unsigned bignum", big.NewInt(1)),
		biFx("minus_one", "canonical tag 3 + empty byte string", big.NewInt(-1)),
		biFx("boundary_255", "last single-byte magnitude", big.NewInt(255)),
		biFx("boundary_256", "first 2-byte magnitude", big.NewInt(256)),
		biFx("u32_max", "2^32 - 1", big.NewInt(1<<32-1)),
		biFx("u32_max_plus_one", "2^32", big.NewInt(1<<32)),
		biFx("u64_max", "2^64 - 1", new(big.Int).SetUint64(^uint64(0))),
		biFx("two_pow_128", "17-byte magnitude", new(big.Int).Lsh(big.NewInt(1), 128)),
		biFx("minus_25", "negative-int encoding boundary", big.NewInt(-25)),
		biFx("neg_two_pow_128", "17-byte negative magnitude", new(big.Int).Neg(new(big.Int).Lsh(big.NewInt(1), 128))),
	}
}

// ----- Decimal fixtures -------------------------------------------------

type decimalInput struct {
	Mantissa string `json:"mantissa"`
	Exponent int32  `json:"exponent"`
}

func decFx(name, notes string, mantissa *big.Int, exp int32) fixture {
	d := decimal.NewFromBigInt(mantissa, exp)
	var buf bytes.Buffer
	if err := (cborx.Decimal{V: d}).MarshalCBOR(&buf); err != nil {
		panic(err)
	}
	return fixture{
		Case:   name,
		Notes:  notes,
		Input:  decimalInput{Mantissa: mantissa.Text(10), Exponent: exp},
		Golden: hex.EncodeToString(buf.Bytes()),
	}
}

func decimalFixtures() []fixture {
	return []fixture{
		decFx("zero", "mantissa 0, exp 0", big.NewInt(0), 0),
		decFx("one", "mantissa 1, exp 0", big.NewInt(1), 0),
		decFx("one_point_five", "mantissa 15, exp -1", big.NewInt(15), -1),
		decFx("neg_half", "mantissa -5, exp -1", big.NewInt(-5), -1),
		decFx("wei_precision", "1 at exp -18 (wei)", big.NewInt(1), -18),
		decFx("mega_unit", "1 at exp 6", big.NewInt(1), 6),
		decFx("ten_pow_40_at_-18", "mantissa >int64, exp -18", new(big.Int).Exp(big.NewInt(10), big.NewInt(40), nil), -18),
	}
}

// ----- Hash32 / Addr20 fixtures -----------------------------------------

type hashInput struct {
	Hex string `json:"hex"`
}

func hashFx(name string, b []byte) fixture {
	if len(b) != 32 {
		panic("bad hash length")
	}
	var h cborx.Hash32
	copy(h.V[:], b)
	var buf bytes.Buffer
	if err := h.MarshalCBOR(&buf); err != nil {
		panic(err)
	}
	return fixture{
		Case:   name,
		Notes:  "major 2, definite length 32",
		Input:  hashInput{Hex: hex.EncodeToString(b)},
		Golden: hex.EncodeToString(buf.Bytes()),
	}
}

func addrFx(name string, b []byte) fixture {
	if len(b) != 20 {
		panic("bad addr length")
	}
	var a cborx.Addr20
	copy(a.V[:], b)
	var buf bytes.Buffer
	if err := a.MarshalCBOR(&buf); err != nil {
		panic(err)
	}
	return fixture{
		Case:   name,
		Notes:  "major 2, definite length 20",
		Input:  hashInput{Hex: hex.EncodeToString(b)},
		Golden: hex.EncodeToString(buf.Bytes()),
	}
}

func hashFixtures() []fixture {
	return []fixture{
		hashFx("zero", bytes.Repeat([]byte{0}, 32)),
		hashFx("one", bytes.Repeat([]byte{1}, 32)),
		hashFx("max", bytes.Repeat([]byte{0xff}, 32)),
		hashFx("incrementing", func() []byte {
			b := make([]byte, 32)
			for i := range b {
				b[i] = byte(i)
			}
			return b
		}()),
	}
}

func addrFixtures() []fixture {
	return []fixture{
		addrFx("zero", bytes.Repeat([]byte{0}, 20)),
		addrFx("vitalik_like", func() []byte {
			b, _ := hex.DecodeString("d8da6bf26964af9d7eed9e03e53415d37aa96045")
			return b
		}()),
	}
}

// ----- Time fixtures ----------------------------------------------------

type timeInput struct {
	UnixNano int64 `json:"unix_nano"`
}

func timeFx(name, notes string, ns int64) fixture {
	tm := time.Unix(0, ns).UTC()
	var buf bytes.Buffer
	if err := (cborx.Time{V: tm}).MarshalCBOR(&buf); err != nil {
		panic(err)
	}
	return fixture{
		Case:   name,
		Notes:  notes,
		Input:  timeInput{UnixNano: ns},
		Golden: hex.EncodeToString(buf.Bytes()),
	}
}

func timeFixtures() []fixture {
	return []fixture{
		timeFx("epoch", "unix 0 → major 0, value 0", 0),
		timeFx("one_ns", "smallest positive ns", 1),
		timeFx("minus_one_ns", "smallest negative ns", -1),
		timeFx("modern", "a specific block-producing moment", 1_700_000_000_123_456_789),
	}
}

// ----- MaybeBigInt fixtures ---------------------------------------------

type maybeInput struct {
	Present bool   `json:"present"`
	Value   string `json:"value,omitempty"`
}

func maybeFx(name, notes string, n *big.Int) fixture {
	var buf bytes.Buffer
	if err := (cborx.MaybeBigInt{V: n}).MarshalCBOR(&buf); err != nil {
		panic(err)
	}
	input := maybeInput{Present: n != nil}
	if n != nil {
		input.Value = n.Text(10)
	}
	return fixture{
		Case:   name,
		Notes:  notes,
		Input:  input,
		Golden: hex.EncodeToString(buf.Bytes()),
	}
}

func maybeFixtures() []fixture {
	return []fixture{
		maybeFx("nil", "nil encodes as CBOR null (0xf6)", nil),
		maybeFx("zero", "zero is encoded as BigInt zero (non-nil)", big.NewInt(0)),
		maybeFx("present_positive", "", big.NewInt(42)),
		maybeFx("present_negative", "", big.NewInt(-42)),
	}
}

// ----- Envelope fixtures ------------------------------------------------

type envelopeInput struct {
	Version string `json:"version"`
	Body    string `json:"body_description"`
}

func envelopeFixtures() []fixture {
	var out []fixture
	for _, inner := range []struct {
		name string
		n    *big.Int
	}{
		{"v1_bigint_zero", big.NewInt(0)},
		{"v1_bigint_one", big.NewInt(1)},
	} {
		var buf bytes.Buffer
		if err := cborx.WriteEnvelope(&buf, cborx.V1, cborx.BigInt{V: inner.n}); err != nil {
			panic(err)
		}
		out = append(out, fixture{
			Case:   inner.name,
			Notes:  "V1 envelope wrapping a BigInt",
			Input:  envelopeInput{Version: "V1 (0x01)", Body: "BigInt " + inner.n.String()},
			Golden: hex.EncodeToString(buf.Bytes()),
		})
	}
	return out
}

// ----- Fixture driver ---------------------------------------------------

type fixtureGroup struct {
	name string
	fxs  []fixture
}

func allFixtureGroups() []fixtureGroup {
	return []fixtureGroup{
		{"bigint", bigIntFixtures()},
		{"maybe_bigint", maybeFixtures()},
		{"decimal", decimalFixtures()},
		{"hash32", hashFixtures()},
		{"addr20", addrFixtures()},
		{"time", timeFixtures()},
		{"envelope", envelopeFixtures()},
	}
}

func TestGoldenFixtures(t *testing.T) {
	dir := testdataDir(t)
	if *goldensUpdate {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	groups := allFixtureGroups()
	for _, g := range groups {
		for _, fx := range g.fxs {
			stem := filepath.Join(dir, fmt.Sprintf("%s_%s", g.name, fx.Case))
			inputPath := stem + ".input.json"
			goldenPath := stem + ".golden.hex"

			inputJSON, err := json.MarshalIndent(fx.Input, "", "  ")
			if err != nil {
				t.Fatalf("marshal fixture description %s: %v", fx.Case, err)
			}
			inputJSON = append(inputJSON, '\n')

			goldenBytes := []byte(fx.Golden + "\n")

			if *goldensUpdate {
				if err := os.WriteFile(inputPath, inputJSON, 0o644); err != nil {
					t.Fatalf("write %s: %v", inputPath, err)
				}
				if err := os.WriteFile(goldenPath, goldenBytes, 0o644); err != nil {
					t.Fatalf("write %s: %v", goldenPath, err)
				}
				continue
			}

			gotGolden, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Errorf("read %s: %v — run `go test -update` to regenerate", goldenPath, err)
				continue
			}
			if string(gotGolden) != string(goldenBytes) {
				t.Errorf("%s: golden drift\n  file:    %s\n  current: %s",
					stem, string(gotGolden), string(goldenBytes))
			}
			gotInput, err := os.ReadFile(inputPath)
			if err != nil {
				t.Errorf("read %s: %v", inputPath, err)
				continue
			}
			if string(gotInput) != string(inputJSON) {
				t.Errorf("%s: input.json drift\n  file:    %s\n  current: %s",
					stem, string(gotInput), string(inputJSON))
			}

			// Cross-validate with an independent CBOR decoder so we
			// know the bytes parse as RFC 8949 (not just
			// round-trip through cbor-gen). The envelope group
			// prepends a non-CBOR version byte, so skip the body
			// check there — the adapter-level fixtures on their
			// own already prove the body bytes are valid CBOR.
			if g.name == "envelope" {
				continue
			}
			raw, err := hex.DecodeString(fx.Golden)
			if err != nil {
				t.Fatalf("decode %s golden hex: %v", stem, err)
			}
			var any interface{}
			if err := cbor.Unmarshal(raw, &any); err != nil {
				t.Errorf("%s: fxamacker rejects our bytes: %v (hex=%s)", stem, err, fx.Golden)
			}
		}
	}
}

// TestDeterminism is the wave-critical test: generate a fixed-seed
// population of each adapter type and encode each one fuzzInstances ×
// fuzzIterations times in a shuffled order. Every encoding must match
// the first byte-for-byte.
func TestDeterminism(t *testing.T) {
	rng := rand.New(rand.NewSource(fuzzSeed))

	instances := buildFuzzInstances(rng)
	if len(instances) != fuzzInstances*7 {
		t.Fatalf("expected %d instances, got %d", fuzzInstances*7, len(instances))
	}

	// First-pass canonical encoding.
	canon := make(map[int][]byte, len(instances))
	for i, inst := range instances {
		canon[i] = inst.encode()
	}

	// Iterate over the whole population, in shuffled order each round,
	// re-encoding and asserting byte-equality against canon[i].
	for round := 0; round < fuzzIterations; round++ {
		order := rng.Perm(len(instances))
		for _, i := range order {
			got := instances[i].encode()
			if !bytes.Equal(got, canon[i]) {
				t.Fatalf("round %d, inst #%d (%s): byte drift\n  canon: %x\n  got:   %x",
					round, i, instances[i].label, canon[i], got)
			}
		}
	}
}

// fuzzInstance is one typed encode target.
type fuzzInstance struct {
	label  string
	encode func() []byte
}

func buildFuzzInstances(rng *rand.Rand) []fuzzInstance {
	var all []fuzzInstance

	// BigInt
	for i := 0; i < fuzzInstances; i++ {
		n := randomBigInt(rng)
		all = append(all, fuzzInstance{
			label: "BigInt",
			encode: func() []byte {
				var buf bytes.Buffer
				if err := (cborx.BigInt{V: n}).MarshalCBOR(&buf); err != nil {
					panic(err)
				}
				return buf.Bytes()
			},
		})
	}

	// MaybeBigInt
	for i := 0; i < fuzzInstances; i++ {
		var n *big.Int
		if rng.Intn(4) == 0 {
			n = nil
		} else {
			n = randomBigInt(rng)
		}
		all = append(all, fuzzInstance{
			label: "MaybeBigInt",
			encode: func() []byte {
				var buf bytes.Buffer
				if err := (cborx.MaybeBigInt{V: n}).MarshalCBOR(&buf); err != nil {
					panic(err)
				}
				return buf.Bytes()
			},
		})
	}

	// Decimal
	for i := 0; i < fuzzInstances; i++ {
		mant := randomBigInt(rng)
		exp := int32(rng.Intn(37) - 18)
		d := decimal.NewFromBigInt(mant, exp)
		all = append(all, fuzzInstance{
			label: "Decimal",
			encode: func() []byte {
				var buf bytes.Buffer
				if err := (cborx.Decimal{V: d}).MarshalCBOR(&buf); err != nil {
					panic(err)
				}
				return buf.Bytes()
			},
		})
	}

	// Hash32
	for i := 0; i < fuzzInstances; i++ {
		var h cborx.Hash32
		rng.Read(h.V[:])
		all = append(all, fuzzInstance{
			label: "Hash32",
			encode: func() []byte {
				var buf bytes.Buffer
				if err := h.MarshalCBOR(&buf); err != nil {
					panic(err)
				}
				return buf.Bytes()
			},
		})
	}

	// Addr20
	for i := 0; i < fuzzInstances; i++ {
		var a cborx.Addr20
		rng.Read(a.V[:])
		all = append(all, fuzzInstance{
			label: "Addr20",
			encode: func() []byte {
				var buf bytes.Buffer
				if err := a.MarshalCBOR(&buf); err != nil {
					panic(err)
				}
				return buf.Bytes()
			},
		})
	}

	// Time
	for i := 0; i < fuzzInstances; i++ {
		ns := rng.Int63n(2_000_000_000_000_000_000) - 1_000_000_000_000_000_000
		tm := time.Unix(0, ns).UTC()
		all = append(all, fuzzInstance{
			label: "Time",
			encode: func() []byte {
				var buf bytes.Buffer
				if err := (cborx.Time{V: tm}).MarshalCBOR(&buf); err != nil {
					panic(err)
				}
				return buf.Bytes()
			},
		})
	}

	// Envelope-wrapped BigInt
	for i := 0; i < fuzzInstances; i++ {
		n := randomBigInt(rng)
		all = append(all, fuzzInstance{
			label: "Envelope(V1, BigInt)",
			encode: func() []byte {
				var buf bytes.Buffer
				if err := cborx.WriteEnvelope(&buf, cborx.V1, cborx.BigInt{V: n}); err != nil {
					panic(err)
				}
				return buf.Bytes()
			},
		})
	}

	// Stable ordering for reproducibility across test invocations.
	sort.SliceStable(all, func(i, j int) bool { return all[i].label < all[j].label })
	return all
}

// randomBigInt produces a uniformly-distributed signed big integer with
// a random byte length between 0 and 32.
func randomBigInt(rng *rand.Rand) *big.Int {
	width := rng.Intn(33) // 0..32 bytes
	b := make([]byte, width)
	rng.Read(b)
	n := new(big.Int).SetBytes(b)
	if rng.Intn(2) == 0 {
		n.Neg(n)
	}
	return n
}
