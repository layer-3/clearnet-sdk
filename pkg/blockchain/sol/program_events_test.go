package sol

import (
	"fmt"
	"testing"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"

	"github.com/layer-3/clearnet-sdk/pkg/blockchain/sol/custody"
)

func TestProgramEvents(t *testing.T) {
	tx := &solana.Transaction{Message: solana.Message{AccountKeys: []solana.PublicKey{{1}, testProgramID}}}
	const program, foreign = 1, 0
	cpi := func(programIndex uint16, disc [8]byte, bodyLen int, first byte) rpc.CompiledInstruction {
		data := []byte(eventCPIData(disc, bodyLen))
		if bodyLen > 0 {
			data[16] = first
		}
		return rpc.CompiledInstruction{ProgramIDIndex: programIndex, Data: solana.Base58(data)}
	}
	meta := &rpc.TransactionMeta{InnerInstructions: []rpc.InnerInstruction{
		{Index: 0, Instructions: []rpc.CompiledInstruction{
			cpi(foreign, custody.Event_Deposited, depositedEventMinLen, 0xf0),
			cpi(program, custody.Event_Deposited, depositedEventMinLen+1, 0xa0),
			cpi(program, custody.Event_Executed, executedEventMinLen-1, 0xb0),
		}},
		{Index: 2, Instructions: []rpc.CompiledInstruction{
			cpi(program, [8]byte{7}, 200, 0xc0),
			cpi(program, custody.Event_Executed, executedEventMinLen, 0xa1),
			cpi(program, custody.Event_Deposited, depositedEventMinLen-1, 0xb1),
			cpi(program, custody.Event_Deposited, depositedEventMinLen, 0xa2),
		}},
	}}

	events, malformed := ProgramEvents(tx, meta, testProgramID)
	render := func(evs []ProgramEvent) string {
		var out []string
		for _, ev := range evs {
			out = append(out, fmt.Sprintf("%d/%d/%d/%x", ev.Index, ev.Kind, len(ev.Body), ev.Body[0]))
		}
		return fmt.Sprint(out)
	}
	wantEvents := fmt.Sprint([]string{
		fmt.Sprintf("0/%d/%d/a0", EventDeposited, depositedEventMinLen+1),
		fmt.Sprintf("1/%d/%d/a1", EventExecuted, executedEventMinLen),
		fmt.Sprintf("2/%d/%d/a2", EventDeposited, depositedEventMinLen),
	})
	if got := render(events); got != wantEvents {
		t.Fatalf("events = %s, want %s", got, wantEvents)
	}
	wantMalformed := fmt.Sprint([]string{
		fmt.Sprintf("0/%d/%d/b0", EventExecuted, executedEventMinLen-1),
		fmt.Sprintf("0/%d/%d/b1", EventDeposited, depositedEventMinLen-1),
	})
	if got := render(malformed); got != wantMalformed {
		t.Fatalf("malformed = %s, want %s", got, wantMalformed)
	}
}
