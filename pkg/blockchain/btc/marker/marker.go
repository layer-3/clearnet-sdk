package marker

import (
	"encoding/binary"
	"errors"
)

// Magic is the 4-byte prefix every Yellow deposit marker payload begins with.
const Magic = "YNET"

// Marker payload versions. An unknown version is a hard parse failure.
const (
	Version1 byte = 0x01 // "YNET" || 0x01 || eth_addr(20)                  = 25 bytes
	Version2 byte = 0x02 // "YNET" || 0x02 || eth_addr(20) || reference(32) = 57 bytes
)

const (
	PayloadLenV1 = len(Magic) + 1 + 20      // 25
	PayloadLenV2 = len(Magic) + 1 + 20 + 32 // 57
)

// opReturn is the script opcode a marker output must begin with.
const opReturn = 0x6a

// The three non-canonical data-push opcodes. A canonical push of a payload no
// longer than 75 bytes -- every marker payload qualifies -- is the direct
// form, where the opcode byte itself equals the payload length.
const (
	opPushData1 = 0x4c
	opPushData2 = 0x4d
	opPushData4 = 0x4e
)

// ---- types ------------------------------------------------------------

// Marker is a decoded deposit-attribution marker.
//
// Address is guaranteed non-zero: there is no account whose identifier is
// 0x000...0, so an all-zero address is rejected for both versions.
//
// Reference is the zero value for Version1, non-zero for Version2.
type Marker struct {
	Version   byte
	Address   [20]byte
	Reference [32]byte
}

// Output is one transaction output, reduced to the two fields the
// attribution rule reads.
//
// ValueSats now is unused by the attribution rule, but it is kept on the
// type anyway so a future reversal of that rule is not an API change.
type Output struct {
	ValueSats    int64
	ScriptPubKey []byte
}

// ---- errors -----------------------------------------------------------

// ErrNotMarker means the script is not a marker candidate at all.
var ErrNotMarker = errors.New("marker: not a YNET marker output")

// The remaining errors mean the output IS a marker candidate - its payload
// begins with Magic - but is not a valid marker, which fails validation and
// makes the whole transaction unattributed.
var (
	ErrMultiplePushes   = errors.New("marker: OP_RETURN carries more than one data push")
	ErrNonCanonicalPush = errors.New("marker: payload is not a canonical direct push")
	ErrUnknownVersion   = errors.New("marker: unknown version byte")
	ErrBadLength        = errors.New("marker: payload length does not match version")

	// There is no account whose
	// identifier is 0x000...0.
	ErrZeroAddress = errors.New("marker: address is all-zero")

	// An all-zero reference would compose the identical account URI as a
	// Version1 marker.
	ErrZeroReference = errors.New("marker: version 0x02 reference is all-zero")
)

// Transaction-scoped outcomes (ScanOutputs only).
var (
	ErrNoMarker        = errors.New("marker: transaction carries no marker candidate")
	ErrMultipleMarkers = errors.New("marker: transaction carries more than one marker candidate")
)

// ---- writer -----------------------------------------------------------

// EncodePayload serialises m to its wire payload (25 or 57 bytes). It refuses
// an unknown version, an all-zero Address, and - for Version2 - an all-zero
// Reference. The writer enforces exactly the rules the reader does, so a
// construction bug cannot emit a marker this package would reject.
func EncodePayload(m Marker) ([]byte, error) {
	switch m.Version {
	case Version1:
		if isZero(m.Address[:]) {
			return nil, ErrZeroAddress
		}
		payload := make([]byte, 0, PayloadLenV1)
		payload = append(payload, Magic...)
		payload = append(payload, Version1)
		payload = append(payload, m.Address[:]...)
		return payload, nil
	case Version2:
		if isZero(m.Address[:]) {
			return nil, ErrZeroAddress
		}
		if isZero(m.Reference[:]) {
			return nil, ErrZeroReference
		}
		payload := make([]byte, 0, PayloadLenV2)
		payload = append(payload, Magic...)
		payload = append(payload, Version2)
		payload = append(payload, m.Address[:]...)
		payload = append(payload, m.Reference[:]...)
		return payload, nil
	default:
		return nil, ErrUnknownVersion
	}
}

// EncodeScript serialises m to a complete zero-value OP_RETURN scriptPubKey:
// OP_RETURN <canonical direct push of len(payload)> <payload>. 27 bytes for
// Version1, 59 for Version2. The push opcode is always the direct form
// (0x19 / 0x39); no OP_PUSHDATA variant is ever emitted, and none is accepted
// on the way back in.
func EncodeScript(m Marker) ([]byte, error) {
	payload, err := EncodePayload(m)
	if err != nil {
		return nil, err
	}
	script := make([]byte, 0, 2+len(payload))
	script = append(script, opReturn, byte(len(payload)))
	script = append(script, payload...)
	return script, nil
}

// ---- reader -----------------------------------------------------------

// DecodePayload parses a bare marker payload (no script wrapper).
func DecodePayload(payload []byte) (Marker, error) {
	if len(payload) < len(Magic) || string(payload[:len(Magic)]) != Magic {
		return Marker{}, ErrNotMarker
	}
	if len(payload) < len(Magic)+1 {
		// "YNET" with no version byte: a candidate, but there is no version
		// byte to be unknown, so this is a length failure.
		return Marker{}, ErrBadLength
	}

	version := payload[len(Magic)]
	var wantLen int
	switch version {
	case Version1:
		wantLen = PayloadLenV1
	case Version2:
		wantLen = PayloadLenV2
	default:
		// Version precedes length: an unknown version is reported regardless
		// of what the length happens to be.
		return Marker{}, ErrUnknownVersion
	}
	if len(payload) != wantLen {
		return Marker{}, ErrBadLength
	}

	var m Marker
	m.Version = version
	copy(m.Address[:], payload[len(Magic)+1:len(Magic)+1+20])
	if isZero(m.Address[:]) {
		return Marker{}, ErrZeroAddress
	}
	if version == Version2 {
		copy(m.Reference[:], payload[len(Magic)+1+20:])
		if isZero(m.Reference[:]) {
			return Marker{}, ErrZeroReference
		}
	}
	return m, nil
}

// DecodeScript parses one output's scriptPubKey. ErrNotMarker means "ignore
// this output"; any other error means "this transaction is unattributed".
// Callers MUST distinguish the two.
func DecodeScript(script []byte) (Marker, error) {
	if len(script) == 0 || script[0] != opReturn {
		return Marker{}, ErrNotMarker
	}
	push, ok := decodePush(script[1:])
	if !ok {
		return Marker{}, ErrNotMarker
	}
	if len(push.payload) < len(Magic) || string(push.payload[:len(Magic)]) != Magic {
		return Marker{}, ErrNotMarker
	}

	// From here on the output is a candidate: every outcome below makes the
	// transaction unattributed.

	// The push must account for the whole script. Anything after the first push
	// is an invalid candidate.
	if 1+push.consumed != len(script) {
		return Marker{}, ErrMultiplePushes
	}

	// Deliberately strict: the only accepted push encoding is the canonical
	// direct form (opcode == payload length). OP_PUSHDATA1/2/4 are rejected
	// even though a wrapped marker payload is byte-for-byte identical to a
	// canonical one and even though Bitcoin Core's nulldata standardness
	// rules relay such a push. This is a deliberate ruling, which pins exactly
	// one accepted byte string per marker, and this makes the format trivially
	// mirrorable in TypeScript and removes a whole class of writer/reader
	// disagreement.
	//
	// Loosening to also accept OP_PUSHDATA1 is a deliberate future option.
	if !push.canonical {
		return Marker{}, ErrNonCanonicalPush
	}

	return DecodePayload(push.payload)
}

// ---- transaction-scoped rule -----------------------------------------

// ScanOutputs applies the attribution rule to one transaction's outputs:
// exactly one marker candidate, and that candidate must be valid.
// Zero candidates yields ErrNoMarker; two or more yields ErrMultipleMarkers;
// a single invalid candidate yields that candidate's validation error.
// Outputs that are not candidates are ignored regardless of count.
//
// ScanOutputs deliberately says nothing about WHICH outputs are credited --
// that requires the generic deposit address and the dust floor, neither of
// which this package knows about.
//
// IMPORTANT: An error here means "this transaction is not attributed". It does NOT
// mean "ignore this transaction", and a caller that treats it that way may lose
// funds. In particular ErrNoMarker says only that no output carried a marker;
// it says nothing about whether the transaction paid the vault.
//
// The caller owns the value side and MUST cross it with the result here:
//
//	value paid to the deposit address >= floor,  valid marker    -> credit the named account
//	value paid to the deposit address >= floor,  any error here  -> UNATTRIBUTED INFLOW: the
//	                                                                vault has funds nobody is
//	                                                                credited for. Never drop it;
//	                                                                it must be logged and routed
//	                                                                to whatever destination the
//	                                                                deposit policy defines, or it
//	                                                                surfaces as reconciler drift.
//	no qualifying value,                         valid marker    -> nothing was paid; inert
//	no qualifying value,                         any error here  -> ordinary foreign traffic
//
// The distinction between the second row and the fourth is the whole reason
// this function reports a specific error rather than a bool: an invalid or
// absent marker on a *funded* transaction is an event the vault must account
// for, while the same marker on an unfunded one is noise.
func ScanOutputs(outs []Output) (Marker, error) {
	type candidate struct {
		marker Marker
		err    error
	}
	var candidates []candidate
	for _, o := range outs {
		m, err := DecodeScript(o.ScriptPubKey)
		if errors.Is(err, ErrNotMarker) {
			continue
		}
		candidates = append(candidates, candidate{marker: m, err: err})
	}
	switch len(candidates) {
	case 0:
		return Marker{}, ErrNoMarker
	case 1:
		return candidates[0].marker, candidates[0].err
	default:
		return Marker{}, ErrMultipleMarkers
	}
}

// ---- push parsing (generic across all push encodings) -----------------

// decodedPush is one data push, parsed generically so the magic can be tested
// against its payload before any judgment is made about whether the encoding
// was canonical (see DecodeScript).
type decodedPush struct {
	payload   []byte
	canonical bool // true only for the direct-push form (opcode == length)
	consumed  int  // bytes consumed from s, including the opcode(s) and any length prefix
}

// decodePush decodes one data push starting at s[0]: a canonical direct push
// (opcode 0x01-0x4b), or OP_PUSHDATA1/2/4. ok is false if s does not begin
// with a recognized push opcode, or the declared length runs past the end of s.
func decodePush(s []byte) (decodedPush, bool) {
	if len(s) == 0 {
		return decodedPush{}, false
	}
	switch op := s[0]; {
	case op >= 0x01 && op <= 0x4b:
		n := int(op)
		if len(s) < 1+n {
			return decodedPush{}, false
		}
		return decodedPush{payload: s[1 : 1+n], canonical: true, consumed: 1 + n}, true
	case op == opPushData1:
		if len(s) < 2 {
			return decodedPush{}, false
		}
		n := int(s[1])
		if len(s) < 2+n {
			return decodedPush{}, false
		}
		return decodedPush{payload: s[2 : 2+n], consumed: 2 + n}, true
	case op == opPushData2:
		if len(s) < 3 {
			return decodedPush{}, false
		}
		n := int(binary.LittleEndian.Uint16(s[1:3]))
		if len(s) < 3+n {
			return decodedPush{}, false
		}
		return decodedPush{payload: s[3 : 3+n], consumed: 3 + n}, true
	case op == opPushData4:
		if len(s) < 5 {
			return decodedPush{}, false
		}
		n := int(binary.LittleEndian.Uint32(s[1:5]))
		if n < 0 || len(s) < 5+n {
			return decodedPush{}, false
		}
		return decodedPush{payload: s[5 : 5+n], consumed: 5 + n}, true
	default:
		return decodedPush{}, false
	}
}

// isZero reports whether every byte of b is 0x00.
func isZero(b []byte) bool {
	for _, v := range b {
		if v != 0 {
			return false
		}
	}
	return true
}
