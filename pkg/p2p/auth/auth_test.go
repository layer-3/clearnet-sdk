package auth

import (
	"context"
	"crypto/ecdsa"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	libp2p "github.com/libp2p/go-libp2p"
	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/sign"
)

var testIssuerID = common.HexToAddress("0x0000000000000000000000000000000000001234")

type testSignerSource struct {
	sets map[common.Address]core.ReceiptSignerSet
	err  error
}

func (s testSignerSource) LoadReceiptSigners(_ context.Context, issuerID common.Address) (core.ReceiptSignerSet, error) {
	if s.err != nil {
		return core.ReceiptSignerSet{}, s.err
	}
	set, ok := s.sets[issuerID]
	if !ok {
		return core.ReceiptSignerSet{}, nil
	}
	return set, nil
}

func TestAuth_Operator(t *testing.T) {
	srv, cli := newPair(t, nil)

	signer := sign.NewKeySignerFromECDSA(mustKey(t))
	addr, err := sign.EthAddress(signer)
	if err != nil {
		t.Fatal(err)
	}

	results := make(chan Result, 1)
	NewServer(testSignerSource{sets: map[common.Address]core.ReceiptSignerSet{
		testIssuerID: {Signers: []common.Address{addr}, Threshold: 1},
	}}, func(_ network.Conn, r Result) {
		results <- r
	}, nil).Register(srv)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := NewClient(ClientOpts{Signer: signer, IssuerID: testIssuerID}).Authenticate(ctx, cli, srv.ID()); err != nil {
		t.Fatalf("Authenticate: %v", err)
	}

	select {
	case r := <-results:
		if r.Role != RoleOperator {
			t.Errorf("role = %v, want operator", r.Role)
		}
		if !strings.EqualFold(r.Address, addr.Hex()) {
			t.Errorf("address = %s, want %s", r.Address, addr.Hex())
		}
		if r.IssuerID != testIssuerID.Hex() {
			t.Errorf("issuer = %s, want %s", r.IssuerID, testIssuerID.Hex())
		}
	case <-time.After(3 * time.Second):
		t.Fatal("server never reported a successful auth")
	}
}

func TestAuth_OperatorRejectedBySignerSource(t *testing.T) {
	srv, cli := newPair(t, nil)

	signer := sign.NewKeySignerFromECDSA(mustKey(t))
	// Allow-list holds a different address, so this operator is rejected.
	other := sign.NewKeySignerFromECDSA(mustKey(t))
	otherAddr, _ := sign.EthAddress(other)

	results := make(chan Result, 1)
	NewServer(testSignerSource{sets: map[common.Address]core.ReceiptSignerSet{
		testIssuerID: {Signers: []common.Address{otherAddr}, Threshold: 1},
	}}, func(_ network.Conn, r Result) {
		results <- r
	}, nil).Register(srv)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// The client side returns nil — it does not wait for the server's verdict;
	// rejection is observed by the server never invoking onAuth.
	if err := NewClient(ClientOpts{Signer: signer, IssuerID: testIssuerID}).Authenticate(ctx, cli, srv.ID()); err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	select {
	case r := <-results:
		t.Fatalf("expected rejection, but server authenticated %+v", r)
	case <-time.After(time.Second):
		// expected: no callback
	}
}

func TestAuth_OperatorRequiresIssuerID(t *testing.T) {
	srv, cli := newPair(t, nil)
	NewServer(testSignerSource{sets: map[common.Address]core.ReceiptSignerSet{}}, nil, nil).Register(srv)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := NewClient(ClientOpts{Signer: sign.NewKeySignerFromECDSA(mustKey(t))}).Authenticate(ctx, cli, srv.ID())
	if err == nil || !strings.Contains(err.Error(), "requires issuer id") {
		t.Fatalf("Authenticate = %v, want issuer id error", err)
	}
}

func TestAuth_Passive(t *testing.T) {
	priv, _, err := libp2pcrypto.GenerateKeyPair(libp2pcrypto.Ed25519, -1)
	if err != nil {
		t.Fatal(err)
	}
	srv := newHost(t, nil)
	cli := newHost(t, priv) // client identity must match the passive signing key
	connect(t, cli, srv)

	results := make(chan Result, 1)
	// Empty allow-list: operator gate disabled, but passive does not consult it.
	NewServer(nil, func(_ network.Conn, r Result) { results <- r }, nil).Register(srv)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := NewClient(ClientOpts{IdentityKey: priv}).Authenticate(ctx, cli, srv.ID()); err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	select {
	case r := <-results:
		if r.Role != RolePassive {
			t.Errorf("role = %v, want passive", r.Role)
		}
		if r.Address != "" {
			t.Errorf("address = %q, want empty for passive", r.Address)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("server never reported a successful passive auth")
	}
}

func TestAuth_PassiveRejectedWhenSignerSourceConfigured(t *testing.T) {
	priv, _, err := libp2pcrypto.GenerateKeyPair(libp2pcrypto.Ed25519, -1)
	if err != nil {
		t.Fatal(err)
	}
	srv := newHost(t, nil)
	cli := newHost(t, priv)
	connect(t, cli, srv)

	results := make(chan Result, 1)
	NewServer(testSignerSource{sets: map[common.Address]core.ReceiptSignerSet{}}, func(_ network.Conn, r Result) {
		results <- r
	}, nil).Register(srv)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := NewClient(ClientOpts{IdentityKey: priv}).Authenticate(ctx, cli, srv.ID()); err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	select {
	case r := <-results:
		t.Fatalf("expected rejection, but server authenticated %+v", r)
	case <-time.After(time.Second):
	}
}

// ── helpers ───────────────────────────────────────────────────────────────

func mustKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	k, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	return k
}

// newPair builds a server + client host (client identity optional) and connects
// them.
func newPair(t *testing.T, cliKey libp2pcrypto.PrivKey) (srv, cli host.Host) {
	t.Helper()
	srv = newHost(t, nil)
	cli = newHost(t, cliKey)
	connect(t, cli, srv)
	return srv, cli
}

func newHost(t *testing.T, identity libp2pcrypto.PrivKey) host.Host {
	t.Helper()
	opts := []libp2p.Option{libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0")}
	if identity != nil {
		opts = append(opts, libp2p.Identity(identity))
	}
	h, err := libp2p.New(opts...)
	if err != nil {
		t.Fatalf("libp2p new: %v", err)
	}
	t.Cleanup(func() { _ = h.Close() })
	return h
}

func connect(t *testing.T, from, to host.Host) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := from.Connect(ctx, peer.AddrInfo{ID: to.ID(), Addrs: to.Addrs()}); err != nil {
		t.Fatalf("connect: %v", err)
	}
}
