package sol

import (
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"

	"github.com/layer-3/clearnet-sdk/pkg/blockchain/sol/custody"
)

// EventKind identifies a custody program event.
type EventKind uint8

const (
	EventDeposited EventKind = iota + 1
	EventExecuted
)

// ProgramEvent is one custody program event CPI. Body is the borsh-encoded
// event without the tag and discriminator.
type ProgramEvent struct {
	// Zero-based position among the transaction's counted events; the index
	// DepositID takes. Unset for a malformed event.
	Index uint64
	Kind  EventKind
	Body  []byte
}

// Byte lengths below which a Deposited or Executed event body does not decode.
const (
	depositedEventMinLen = 32 + 20 + 32 + 32 + 8
	executedEventMinLen  = 32 + 32 + 32 + 8
)

// eventIxTag is anchor_lang's EVENT_IX_TAG_LE, the 8-byte prefix of the
// self-CPI instruction data emit_cpi! produces.
var eventIxTag = [8]byte{0xe4, 0x45, 0xa5, 0x2e, 0x51, 0xcb, 0x9a, 0x1d}

// ProgramEvents returns the Deposited and Executed event CPIs that programID
// emitted in tx, in order across all inner instructions. Account keys are the
// static keys followed by the loaded writable and read-only addresses. An
// event whose body is too short to decode goes to malformed and is not
// counted, so it does not shift the Index of later events. tx and meta must
// not be nil.
func ProgramEvents(tx *solana.Transaction, meta *rpc.TransactionMeta, programID solana.PublicKey) (events, malformed []ProgramEvent) {
	keys := make([]solana.PublicKey, 0, len(tx.Message.AccountKeys)+len(meta.LoadedAddresses.Writable)+len(meta.LoadedAddresses.ReadOnly))
	keys = append(keys, tx.Message.AccountKeys...)
	keys = append(keys, meta.LoadedAddresses.Writable...)
	keys = append(keys, meta.LoadedAddresses.ReadOnly...)

	for _, inner := range meta.InnerInstructions {
		for _, ci := range inner.Instructions {
			if int(ci.ProgramIDIndex) >= len(keys) || !keys[ci.ProgramIDIndex].Equals(programID) {
				continue
			}
			data := []byte(ci.Data)
			if len(data) < 16 || [8]byte(data[:8]) != eventIxTag {
				continue
			}
			ev := ProgramEvent{Body: data[16:]}
			var minLen int
			switch [8]byte(data[8:16]) {
			case custody.Event_Deposited:
				ev.Kind, minLen = EventDeposited, depositedEventMinLen
			case custody.Event_Executed:
				ev.Kind, minLen = EventExecuted, executedEventMinLen
			default:
				continue
			}
			if len(ev.Body) < minLen {
				malformed = append(malformed, ev)
				continue
			}
			ev.Index = uint64(len(events))
			events = append(events, ev)
		}
	}
	return events, malformed
}
