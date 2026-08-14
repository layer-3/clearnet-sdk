package xrpl

import (
	"encoding/json"
	"testing"

	"github.com/Peersyst/xrpl-go/xrpl/transaction"
)

func TestValidateNetworkIDPolicy(t *testing.T) {
	tests := []struct {
		name   string
		policy NetworkIDPolicy
		value  any
		set    bool
		ok     bool
	}{
		{name: "standard absent", policy: NetworkIDPolicy{}, ok: true},
		{name: "standard present", policy: NetworkIDPolicy{}, value: uint32(1), set: true},
		{name: "custom exact", policy: NetworkIDPolicy{Required: true, NetworkID: 21337}, value: uint32(21337), set: true, ok: true},
		{name: "custom missing", policy: NetworkIDPolicy{Required: true, NetworkID: 21337}},
		{name: "custom wrong", policy: NetworkIDPolicy{Required: true, NetworkID: 21337}, value: uint32(21338), set: true},
		{name: "custom malformed", policy: NetworkIDPolicy{Required: true, NetworkID: 21337}, value: "21337", set: true},
		{name: "custom fractional", policy: NetworkIDPolicy{Required: true, NetworkID: 21337}, value: json.Number("21337.5"), set: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			flat := transaction.FlatTransaction{}
			if test.set {
				flat["NetworkID"] = test.value
			}
			err := ValidateNetworkID(flat, test.policy)
			if (err == nil) != test.ok {
				t.Fatalf("ValidateNetworkID error = %v, want success %v", err, test.ok)
			}
		})
	}
}
