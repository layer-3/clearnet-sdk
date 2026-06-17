package decimal

import (
	"math"
	"math/big"
	"math/rand"
	"testing"
)

// ---------------------------------------------------------------------------
// Conservation Properties
// ---------------------------------------------------------------------------

func TestFunctional_Addition_Conservation(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	for i := 0; i < 1000; i++ {
		a := NewFromInt(rng.Int63n(1e15))
		b := NewFromInt(rng.Int63n(1e15))
		result := a.Add(b).Sub(b)
		if !result.Equal(a) {
			t.Fatalf("iteration %d: (a+b)-b != a: a=%s b=%s got=%s", i, a.String(), b.String(), result.String())
		}
	}
}

func TestFunctional_Subtraction_Conservation(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	for i := 0; i < 1000; i++ {
		av := rng.Int63n(1e15)
		bv := rng.Int63n(av + 1) // ensure b <= a
		a := NewFromInt(av)
		b := NewFromInt(bv)
		result := a.Sub(b).Add(b)
		if !result.Equal(a) {
			t.Fatalf("iteration %d: (a-b)+b != a: a=%s b=%s got=%s", i, a.String(), b.String(), result.String())
		}
	}
}

func TestFunctional_AddSub_MixedScaleSignedConservation(t *testing.T) {
	cases := []struct {
		name string
		a    string
		b    string
	}{
		{name: "positive_integer_plus_sub_wei", a: "1000000000000000000", b: "0.000000000000000001"},
		{name: "negative_integer_plus_fraction", a: "-42", b: "0.125"},
		{name: "fraction_plus_large_negative", a: "0.000000123456789", b: "-999999999999999999"},
		{name: "both_negative_mixed_scale", a: "-123.456", b: "-0.000000000000000001"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := RequireFromString(tc.a)
			b := RequireFromString(tc.b)

			if got := a.Add(b).Sub(b); !got.Equal(a) {
				t.Fatalf("(a+b)-b != a: a=%s b=%s got=%s", a, b, got)
			}
			if got := a.Sub(b).Add(b); !got.Equal(a) {
				t.Fatalf("(a-b)+b != a: a=%s b=%s got=%s", a, b, got)
			}
		})
	}
}

func TestFunctional_MultiplicationCommutative(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	for i := 0; i < 1000; i++ {
		a := NewFromInt(rng.Int63n(1e9))
		b := NewFromInt(rng.Int63n(1e9))
		ab := a.Mul(b)
		ba := b.Mul(a)
		if !ab.Equal(ba) {
			t.Fatalf("iteration %d: a*b != b*a: a=%s b=%s a*b=%s b*a=%s", i, a.String(), b.String(), ab.String(), ba.String())
		}
	}
}

// ---------------------------------------------------------------------------
// Precision
// ---------------------------------------------------------------------------

func TestFunctional_Multiplication_Precision_18Decimals(t *testing.T) {
	// 1.000000000000000001 * 1.000000000000000001
	// = 1.000000000000000002000000000000000001 (exact)
	// Decimal should preserve at least the 18-decimal term.
	a := RequireFromString("1.000000000000000001")
	result := a.Mul(a)

	// The exact product is 1.000000000000000002000000000000000001.
	expected := RequireFromString("1.000000000000000002000000000000000001")
	if !result.Equal(expected) {
		t.Fatalf("expected %s, got %s", expected.String(), result.String())
	}
}

func TestFunctional_Division_Deterministic(t *testing.T) {
	one := NewFromInt(1)
	three := NewFromInt(3)
	first := one.Div(three)
	for i := 1; i < 100; i++ {
		got := one.Div(three)
		if !got.Equal(first) {
			t.Fatalf("iteration %d: 1/3 changed: expected %s, got %s", i, first.String(), got.String())
		}
	}
}

// ---------------------------------------------------------------------------
// Boundary Values
// ---------------------------------------------------------------------------

func TestFunctional_MaxValue_NoOverflow(t *testing.T) {
	// Use a very large number near the practical limit.
	maxStr := "99999999999999999999999999999999999999999999999999999999999999999999999999999999"
	maxVal := RequireFromString(maxStr)
	zero := NewFromInt(0)
	result := maxVal.Add(zero)
	if !result.Equal(maxVal) {
		t.Fatalf("max + 0 != max: got %s", result.String())
	}
}

func TestFunctional_ZeroDivision(t *testing.T) {
	// Decimal.Div uses QuoRem which panics on division by zero.
	// Verify it panics rather than silently returning garbage.
	a := NewFromInt(42)
	zero := NewFromInt(0)

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic on division by zero, but did not panic")
		}
	}()

	_ = a.Div(zero)
	t.Fatal("should not reach here after division by zero")
}

func TestFunctional_ZeroOperations(t *testing.T) {
	zero := NewFromInt(0)
	a := RequireFromString("123456.789")

	// 0 + a == a
	if !zero.Add(a).Equal(a) {
		t.Fatal("0 + a != a")
	}

	// 0 * a == 0
	if !zero.Mul(a).IsZero() {
		t.Fatal("0 * a != 0")
	}

	// a - 0 == a
	if !a.Sub(zero).Equal(a) {
		t.Fatal("a - 0 != a")
	}
}

func TestFunctional_Shift_ExponentOverflowPanics(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value Decimal
		shift int32
	}{
		{name: "positive overflow", value: New(1, math.MaxInt32), shift: 1},
		{name: "negative overflow", value: New(1, math.MinInt32), shift: -1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("expected exponent overflow panic")
				}
			}()
			_ = tc.value.Shift(tc.shift)
		})
	}
}

func TestFunctional_Shift_ExponentBoundaryAllowed(t *testing.T) {
	if got := New(1, math.MaxInt32-1).Shift(1); got.Exponent() != math.MaxInt32 {
		t.Fatalf("max boundary exponent = %d, want %d", got.Exponent(), math.MaxInt32)
	}
	if got := New(1, math.MinInt32+1).Shift(-1); got.Exponent() != math.MinInt32 {
		t.Fatalf("min boundary exponent = %d, want %d", got.Exponent(), math.MinInt32)
	}
}

// ---------------------------------------------------------------------------
// String Round-Trip
// ---------------------------------------------------------------------------

func TestFunctional_StringRoundTrip(t *testing.T) {
	rng := rand.New(rand.NewSource(99))
	for i := 0; i < 1000; i++ {
		// Generate value with up to 15 integer digits and up to 10 fractional digits.
		intPart := rng.Int63n(1e15)
		fracExp := int32(rng.Intn(10) + 1) // 1..10
		v := New(intPart, -fracExp)

		str := v.String()
		parsed, err := NewFromString(str)
		if err != nil {
			t.Fatalf("iteration %d: parse error for %q: %v", i, str, err)
		}
		if !parsed.Equal(v) {
			t.Fatalf("iteration %d: round-trip failed: original=%s parsed=%s", i, v.String(), parsed.String())
		}
	}
}

// ---------------------------------------------------------------------------
// Comparison Operators
// ---------------------------------------------------------------------------

func TestFunctional_Comparison_LessThan(t *testing.T) {
	cases := []struct {
		a, b string
	}{
		{"0", "1"},
		{"-1", "0"},
		{"1.5", "1.6"},
		{"0.0001", "0.001"},
		{"-100", "100"},
		{"999999999999999999", "1000000000000000000"},
	}
	for _, tc := range cases {
		a := RequireFromString(tc.a)
		b := RequireFromString(tc.b)
		if !a.LessThan(b) {
			t.Errorf("%s should be < %s", tc.a, tc.b)
		}
		if a.GreaterThanOrEqual(b) {
			t.Errorf("%s should NOT be >= %s", tc.a, tc.b)
		}
	}
}

func TestFunctional_Comparison_Equal(t *testing.T) {
	cases := []struct {
		a, b string
	}{
		{"0", "0"},
		{"1", "1"},
		{"-1", "-1"},
		{"123.456", "123.456"},
		{"1000000000000000000", "1000000000000000000"},
		{"0.000000000000000001", "0.000000000000000001"},
	}
	for _, tc := range cases {
		a := RequireFromString(tc.a)
		b := RequireFromString(tc.b)
		if !a.Equal(b) {
			t.Errorf("%s should equal %s", tc.a, tc.b)
		}
		if a.Cmp(b) != 0 {
			t.Errorf("Cmp(%s, %s) should be 0, got %d", tc.a, tc.b, a.Cmp(b))
		}
	}
}

func TestFunctional_Comparison_GreaterThan(t *testing.T) {
	cases := []struct {
		a, b string
	}{
		{"1", "0"},
		{"0", "-1"},
		{"1.6", "1.5"},
		{"0.001", "0.0001"},
		{"100", "-100"},
		{"1000000000000000000", "999999999999999999"},
	}
	for _, tc := range cases {
		a := RequireFromString(tc.a)
		b := RequireFromString(tc.b)
		if !a.GreaterThan(b) {
			t.Errorf("%s should be > %s", tc.a, tc.b)
		}
		if a.LessThanOrEqual(b) {
			t.Errorf("%s should NOT be <= %s", tc.a, tc.b)
		}
	}
}

// ---------------------------------------------------------------------------
// AMM Arithmetic Precision
// ---------------------------------------------------------------------------

func TestFunctional_AMM_ConstantProduct_Precision(t *testing.T) {
	// Constant-product invariant: (rA + dx) * (rB - dy) >= rA * rB
	// where dy = floor(rB * dx / (rA + dx))
	// Using integer wei values with floor division (QuoRem precision 0) to
	// guarantee the invariant holds -- exactly as an on-chain AMM would.
	rA := RequireFromString("1000000000000000000") // 1e18 wei
	rB := RequireFromString("2000000000000000000") // 2e18 wei
	dx := RequireFromString("1000000000000000")    // 1e15 wei

	rAPlusDx := rA.Add(dx)
	// Floor division: dy = floor(rB * dx / (rA + dx))
	dy, _ := rB.Mul(dx).QuoRem(rAPlusDx, 0)

	kBefore := rA.Mul(rB)
	kAfter := rAPlusDx.Mul(rB.Sub(dy))

	// kAfter >= kBefore (no drain of reserves with floor division)
	if kAfter.LessThan(kBefore) {
		t.Fatalf("constant product violated: before=%s after=%s", kBefore.String(), kAfter.String())
	}

	// Precision loss should be bounded: kAfter - kBefore < rAPlusDx (one unit of rounding)
	diff := kAfter.Sub(kBefore)
	if diff.GreaterThanOrEqual(rAPlusDx) {
		t.Fatalf("precision loss too large: diff=%s bound=%s", diff.String(), rAPlusDx.String())
	}
}

func TestFunctional_AMM_LargeReserves(t *testing.T) {
	// Reserves near 2^128 (~3.4e38). Verify swap math does not overflow.
	bigReserve := new(big.Int).Exp(big.NewInt(2), big.NewInt(128), nil)
	rA := NewFromBigInt(bigReserve, 0)
	rB := NewFromBigInt(bigReserve, 0)
	dx := NewFromInt(1e15) // relatively small swap

	rAPlusDx := rA.Add(dx)
	dy, _ := rB.Mul(dx).QuoRem(rAPlusDx, 0)

	kBefore := rA.Mul(rB)
	kAfter := rAPlusDx.Mul(rB.Sub(dy))

	if kAfter.LessThan(kBefore) {
		t.Fatalf("constant product violated with large reserves: before=%s after=%s", kBefore.String(), kAfter.String())
	}

	// dy must be positive and less than rB
	if dy.IsZero() || dy.IsNegative() {
		t.Fatalf("dy should be positive, got %s", dy.String())
	}
	if dy.GreaterThanOrEqual(rB) {
		t.Fatalf("dy >= rB: dy=%s rB=%s", dy.String(), rB.String())
	}
}

func TestFunctional_AMM_SmallReserves(t *testing.T) {
	// Reserves near 1 wei. Swap math should not underflow.
	rA := NewFromInt(1000) // 1000 wei
	rB := NewFromInt(1000) // 1000 wei
	dx := NewFromInt(1)    // 1 wei swap

	rAPlusDx := rA.Add(dx)
	// Floor division: for tiny reserves this may truncate to 0
	dy, _ := rB.Mul(dx).QuoRem(rAPlusDx, 0)

	// dy should be non-negative
	if dy.IsNegative() {
		t.Fatalf("dy should not be negative, got %s", dy.String())
	}

	// Even if dy rounds to 0, kAfter must be >= kBefore
	kBefore := rA.Mul(rB)
	kAfter := rAPlusDx.Mul(rB.Sub(dy))
	if kAfter.LessThan(kBefore) {
		t.Fatalf("constant product violated with small reserves: before=%s after=%s", kBefore.String(), kAfter.String())
	}
}

// ---------------------------------------------------------------------------
// Edge Cases
// ---------------------------------------------------------------------------

func TestFunctional_NegativeResult_Handling(t *testing.T) {
	// Decimal is signed, so 0-1 should yield -1, not wrap.
	zero := NewFromInt(0)
	one := NewFromInt(1)
	result := zero.Sub(one)

	if !result.IsNegative() {
		t.Fatalf("0 - 1 should be negative, got %s", result.String())
	}

	expected := NewFromInt(-1)
	if !result.Equal(expected) {
		t.Fatalf("0 - 1 should be -1, got %s", result.String())
	}
}

func TestFunctional_VerySmallValues(t *testing.T) {
	// 1 wei = 10^-18
	oneWei := RequireFromString("0.000000000000000001")

	// Addition: 1 wei + 1 wei = 2 wei
	twoWei := RequireFromString("0.000000000000000002")
	if !oneWei.Add(oneWei).Equal(twoWei) {
		t.Fatal("1 wei + 1 wei != 2 wei")
	}

	// Subtraction: 2 wei - 1 wei = 1 wei
	if !twoWei.Sub(oneWei).Equal(oneWei) {
		t.Fatal("2 wei - 1 wei != 1 wei")
	}

	// Multiplication: 1 wei * 1e18 = 1
	scale := RequireFromString("1000000000000000000")
	if !oneWei.Mul(scale).Equal(NewFromInt(1)) {
		t.Fatalf("1 wei * 1e18 != 1, got %s", oneWei.Mul(scale).String())
	}

	// Comparison
	if !oneWei.LessThan(twoWei) {
		t.Fatal("1 wei should be < 2 wei")
	}
	if !oneWei.GreaterThan(NewFromInt(0)) {
		t.Fatal("1 wei should be > 0")
	}
}

func TestFunctional_VeryLargeValues(t *testing.T) {
	// Values near 2^128
	bigVal := new(big.Int).Exp(big.NewInt(2), big.NewInt(128), nil)
	d := NewFromBigInt(bigVal, 0)

	// Add 1
	d1 := d.Add(NewFromInt(1))
	expected := NewFromBigInt(new(big.Int).Add(bigVal, big.NewInt(1)), 0)
	if !d1.Equal(expected) {
		t.Fatalf("2^128 + 1 mismatch: got %s", d1.String())
	}

	// Multiply by 2
	d2 := d.Mul(NewFromInt(2))
	expected2 := NewFromBigInt(new(big.Int).Mul(bigVal, big.NewInt(2)), 0)
	if !d2.Equal(expected2) {
		t.Fatalf("2^128 * 2 mismatch: got %s", d2.String())
	}

	// Subtract to get back
	if !d2.Sub(d).Equal(d) {
		t.Fatal("2^129 - 2^128 != 2^128")
	}

	// Division: 2^128 / 2^128 = 1
	one := d.Div(d)
	if !one.Equal(NewFromInt(1)) {
		t.Fatalf("2^128 / 2^128 should be 1, got %s", one.String())
	}
}
