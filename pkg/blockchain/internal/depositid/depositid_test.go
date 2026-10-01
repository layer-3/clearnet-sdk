package depositid

import "testing"

func TestParseIndex(t *testing.T) {
	tests := []struct {
		id      string
		bitSize int
		want    uint64
		ok      bool
	}{
		{"p:0", 32, 0, true},
		{"p:4294967295", 32, 4294967295, true},
		{"p:4294967296", 32, 0, false},
		{"p:18446744073709551615", 64, 18446744073709551615, true},
		{"p:18446744073709551616", 64, 0, false},
		{"p:", 64, 0, false},
		{"p:01", 64, 0, false},
		{"p:+1", 64, 0, false},
		{"p:-1", 64, 0, false},
		{"p:1_0", 64, 0, false},
		{"p:1:2", 64, 0, false},
		{"p::1", 64, 0, false},
		{"p", 64, 0, false},
		{"P:1", 64, 0, false},
		{"xp:1", 64, 0, false},
		{"", 64, 0, false},
	}
	for _, tc := range tests {
		got, ok := ParseIndex(tc.id, "p", tc.bitSize)
		if got != tc.want || ok != tc.ok {
			t.Fatalf("ParseIndex(%q, %d) = (%d, %v), want (%d, %v)", tc.id, tc.bitSize, got, ok, tc.want, tc.ok)
		}
	}
}
