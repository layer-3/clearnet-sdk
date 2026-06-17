package protocol

import "testing"

// Protocol identifiers and topics are the wire-compatibility boundary. Freeze
// the exact strings so a stray edit can't silently fork the version.
func TestProtocolIDsFrozen(t *testing.T) {
	cases := map[string]string{
		"auth":        ProtocolAuth,
		"burnreceipt": ProtocolBurnReceipt,
		"mintreceipt": ProtocolMintReceipt,
	}
	want := map[string]string{
		"auth":        "/ynp/auth/1.0.0",
		"burnreceipt": "/ynp/burnreceipt/1.0.0",
		"mintreceipt": "/ynp/mintreceipt/1.0.0",
	}
	for k, got := range cases {
		if got != want[k] {
			t.Errorf("%s ID = %q, want %q", k, got, want[k])
		}
	}
}

func TestTopicsFrozen(t *testing.T) {
	cases := map[string]string{
		"blocks":      TopicBlocks,
		"transfers":   TopicTransfers,
		"withdrawals": TopicWithdrawals,
		"challenges":  TopicChallenges,
	}
	want := map[string]string{
		"blocks":      "/clearnet/blocks.v1",
		"transfers":   "/clearnet/transfers.v1",
		"withdrawals": "/clearnet/withdrawals.v1",
		"challenges":  "/clearnet/challenges.v1",
	}
	for k, got := range cases {
		if got != want[k] {
			t.Errorf("%s topic = %q, want %q", k, got, want[k])
		}
	}
}

func TestPoolTopic(t *testing.T) {
	var anchor [32]byte
	anchor[0], anchor[31] = 0xab, 0xcd
	got := PoolTopic(anchor)
	want := "/clearnet/pool/ab000000000000000000000000000000000000000000000000000000000000cd.v1"
	if got != want {
		t.Errorf("PoolTopic = %q, want %q", got, want)
	}
	if hexVariant := PoolTopicHex("ab000000000000000000000000000000000000000000000000000000000000cd"); hexVariant != want {
		t.Errorf("PoolTopicHex = %q, want %q", hexVariant, want)
	}
}
