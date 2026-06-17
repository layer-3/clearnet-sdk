package log

import "testing"

// TestZapLogger_WithKV_NoAlias guards against the parent's KV backing array
// being shared with children: two WithKV calls off the same logger must not
// corrupt each other. Deterministic (no -race needed): the parent is given
// spare capacity so a bare append would write both children into the same slot.
func TestZapLogger_WithKV_NoAlias(t *testing.T) {
	base := NewZapLogger(Config{Level: LevelError}).(*ZapLogger)
	kv := make([]any, 2, 8) // len 2, spare cap → naive append would alias
	kv[0], kv[1] = "base", "0"
	base.keysAndValues = kv

	c1 := base.WithKV("x", "1").(*ZapLogger)
	c2 := base.WithKV("y", "2").(*ZapLogger)

	if got := c1.GetAllKV(); len(got) != 4 || got[2] != "x" || got[3] != "1" {
		t.Errorf("c1 KV = %v, want [base 0 x 1]", got)
	}
	if got := c2.GetAllKV(); len(got) != 4 || got[2] != "y" || got[3] != "2" {
		t.Errorf("c2 KV = %v, want [base 0 y 2]", got)
	}
	if got := base.GetAllKV(); len(got) != 2 {
		t.Errorf("base KV mutated: %v", got)
	}
}
