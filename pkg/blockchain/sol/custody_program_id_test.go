package sol

import (
	"testing"

	"github.com/gagliardetto/solana-go"

	"github.com/layer-3/clearnet-sdk/pkg/blockchain/sol/custody"
)

func TestCustodyInstructionsUseExplicitProgramAccount(t *testing.T) {
	program := solana.MustPublicKeyFromBase58("9L5445asvUGRypJkQgrEcTZY854ZwkZGK3cR9Y6VXEXp")
	account := solana.MustPublicKeyFromBase58("11111111111111111111111111111111")

	assertProgram := func(name string, ix solana.Instruction, err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if got := ix.ProgramID(); !got.Equals(program) {
			t.Fatalf("%s ProgramID: got %s, want %s", name, got, program)
		}
	}

	ix, err := custody.NewDepositSolInstruction(
		[20]uint8{1}, [32]uint8{2}, 1,
		account, account, solana.SystemProgramID, account, program,
	)
	assertProgram("deposit_sol", ix, err)
	ix, err = custody.NewExecuteInstruction(
		account, solana.PublicKey{}, 1, [32]uint8{3}, 2, 4,
		account, account, account, account, account,
		solana.SysVarInstructionsPubkey, solana.SystemProgramID, account, program,
	)
	assertProgram("execute", ix, err)
	ix, err = custody.NewUpdateSignersInstruction(
		[]solana.PublicKey{account}, 1, 2,
		account, solana.SysVarInstructionsPubkey, account, program,
	)
	assertProgram("update_signers", ix, err)
}
