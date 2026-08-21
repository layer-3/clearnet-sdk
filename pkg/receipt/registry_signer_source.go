package receipt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/layer-3/clearnet-sdk/pkg/core"
)

var ConfigRegistrySignersKey = [32]byte(crypto.Keccak256Hash([]byte("clearnet/issuer-receipt-signers")))

// ConfigRegistryEventReader is the read side of the config event store used by
// RegistrySignerSource. The EVM event forwarder defines the write side; an
// integrator store can implement both.
type ConfigRegistryEventReader interface {
	LatestConfigRegistryEvent(ctx context.Context, registry common.Address, issuerID common.Address, key [32]byte) (core.ConfigRegistryEvent, bool, error)
}

// RegistrySignerSource resolves issuer receipt signers from the latest
// ConfigRegistry KEY_SIGNERS payload event.
type RegistrySignerSource struct {
	registry common.Address
	events   ConfigRegistryEventReader
}

var _ core.ReceiptSignerSource = (*RegistrySignerSource)(nil)

func NewRegistrySignerSource(registry common.Address, events ConfigRegistryEventReader) (*RegistrySignerSource, error) {
	if events == nil {
		return nil, fmt.Errorf("registry signer source: nil event reader")
	}
	return &RegistrySignerSource{registry: registry, events: events}, nil
}

func (s *RegistrySignerSource) LoadReceiptSigners(ctx context.Context, issuerID common.Address) (core.ReceiptSignerSet, error) {
	ev, ok, err := s.events.LatestConfigRegistryEvent(ctx, s.registry, issuerID, ConfigRegistrySignersKey)
	if err != nil {
		return core.ReceiptSignerSet{}, fmt.Errorf("registry signer source: load event: %w", err)
	}
	if !ok {
		return core.ReceiptSignerSet{}, fmt.Errorf("registry signer source: no signer payload for issuer %s", issuerID.Hex())
	}
	if ev.Registry != s.registry {
		return core.ReceiptSignerSet{}, fmt.Errorf("registry signer source: event registry %s != %s", ev.Registry.Hex(), s.registry.Hex())
	}
	if ev.IssuerID != issuerID {
		return core.ReceiptSignerSet{}, fmt.Errorf("registry signer source: event issuer %s != %s", ev.IssuerID.Hex(), issuerID.Hex())
	}
	if ev.Key != ConfigRegistrySignersKey {
		return core.ReceiptSignerSet{}, fmt.Errorf("registry signer source: unexpected key 0x%s", common.Bytes2Hex(ev.Key[:]))
	}
	if !ev.HasData {
		return core.ReceiptSignerSet{}, fmt.Errorf("registry signer source: signer event has no data")
	}
	if crypto.Keccak256Hash(ev.Data) != ev.Checksum {
		return core.ReceiptSignerSet{}, fmt.Errorf("registry signer source: payload checksum mismatch")
	}
	return ParseSignerPayload(ev.Data)
}

const SignerPayloadVersion uint32 = 1

type signerPayload struct {
	Version   uint32   `json:"v"`
	Threshold int      `json:"threshold"`
	Signers   []string `json:"signers"` // ascending, lowercase 0x hex
}

func MarshalSignerPayload(signers []common.Address, threshold int) ([]byte, error) {
	return MarshalReceiptSignerPayload(core.ReceiptSignerSet{Signers: signers, Threshold: threshold})
}

// MarshalReceiptSignerPayload returns the deterministic byte encoding of a
// receipt signer set for ConfigRegistrySignersKey.
func MarshalReceiptSignerPayload(set core.ReceiptSignerSet) ([]byte, error) {
	if len(set.Signers) == 0 {
		return nil, fmt.Errorf("signer payload: empty signer set")
	}
	if set.Threshold <= 0 || set.Threshold > len(set.Signers) {
		return nil, fmt.Errorf("signer payload: threshold %d out of range for %d signers", set.Threshold, len(set.Signers))
	}
	seen := make(map[common.Address]struct{}, len(set.Signers))
	hexes := make([]string, 0, len(set.Signers))
	for _, s := range set.Signers {
		if _, dup := seen[s]; dup {
			return nil, fmt.Errorf("signer payload: duplicate signer %s", s.Hex())
		}
		if s == (common.Address{}) {
			return nil, fmt.Errorf("signer payload: zero address is not a valid signer")
		}
		seen[s] = struct{}{}
		hexes = append(hexes, strings.ToLower(s.Hex()))
	}
	sort.Strings(hexes)
	return json.Marshal(signerPayload{Version: SignerPayloadVersion, Threshold: set.Threshold, Signers: hexes})
}

func ParseSignerPayload(b []byte) (core.ReceiptSignerSet, error) {
	var p signerPayload
	if err := json.Unmarshal(b, &p); err != nil {
		return core.ReceiptSignerSet{}, fmt.Errorf("signer payload: decode: %w", err)
	}
	if p.Version != SignerPayloadVersion {
		return core.ReceiptSignerSet{}, fmt.Errorf("signer payload: unsupported version %d (want %d)", p.Version, SignerPayloadVersion)
	}
	if len(p.Signers) == 0 {
		return core.ReceiptSignerSet{}, fmt.Errorf("signer payload: empty signer set")
	}
	out := make([]common.Address, 0, len(p.Signers))
	var last []byte
	for _, s := range p.Signers {
		if !common.IsHexAddress(s) {
			return core.ReceiptSignerSet{}, fmt.Errorf("signer payload: %q is not a hex address", s)
		}
		a := common.HexToAddress(s)
		if a == (common.Address{}) {
			return core.ReceiptSignerSet{}, fmt.Errorf("signer payload: zero address is not a valid signer")
		}
		if last != nil && bytes.Compare(a[:], last) <= 0 {
			return core.ReceiptSignerSet{}, fmt.Errorf("signer payload: signers not strictly ascending at %s", s)
		}
		last = a[:]
		out = append(out, a)
	}
	if p.Threshold <= 0 || p.Threshold > len(out) {
		return core.ReceiptSignerSet{}, fmt.Errorf("signer payload: threshold %d out of range for %d signers", p.Threshold, len(out))
	}
	return core.ReceiptSignerSet{Signers: out, Threshold: p.Threshold}, nil
}
