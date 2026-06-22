package main

import "testing"

func TestSelectProbesDefaultAll(t *testing.T) {
	probes := testProbes()

	got, err := selectProbes(probes, "", false)
	if err != nil {
		t.Fatalf("selectProbes returned error: %v", err)
	}
	if len(got) != len(probes) {
		t.Fatalf("got %d probes, want %d", len(got), len(probes))
	}
	for i := range probes {
		if got[i].name != probes[i].name {
			t.Fatalf("probe %d = %q, want %q", i, got[i].name, probes[i].name)
		}
	}
}

func TestSelectProbesSubset(t *testing.T) {
	got, err := selectProbes(testProbes(), "anvil,solana", true)
	if err != nil {
		t.Fatalf("selectProbes returned error: %v", err)
	}

	want := []string{"anvil", "solana"}
	assertProbeNames(t, got, want)
}

func TestSelectProbesTrimsNames(t *testing.T) {
	got, err := selectProbes(testProbes(), " anvil , rippled ", true)
	if err != nil {
		t.Fatalf("selectProbes returned error: %v", err)
	}

	want := []string{"anvil", "rippled"}
	assertProbeNames(t, got, want)
}

func TestSelectProbesRejectsEmptyName(t *testing.T) {
	if _, err := selectProbes(testProbes(), "anvil,", true); err == nil {
		t.Fatal("expected empty network name to fail")
	}
}

func TestSelectProbesRejectsUnsupportedName(t *testing.T) {
	if _, err := selectProbes(testProbes(), "anvil,ethereum", true); err == nil {
		t.Fatal("expected unsupported network name to fail")
	}
}

func testProbes() []probe {
	return []probe{
		{name: "anvil"},
		{name: "bitcoind"},
		{name: "rippled"},
		{name: "solana"},
	}
}

func assertProbeNames(t *testing.T, probes []probe, want []string) {
	t.Helper()
	if len(probes) != len(want) {
		t.Fatalf("got %d probes, want %d", len(probes), len(want))
	}
	for i := range want {
		if probes[i].name != want[i] {
			t.Fatalf("probe %d = %q, want %q", i, probes[i].name, want[i])
		}
	}
}
