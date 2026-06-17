#!/usr/bin/env bash
# check-preimage-goldens.sh — freeze the byte output of the signing
# preimages against committed goldens.
#
# The SDK owns the byte-for-byte custody↔clearnet contract (ADR-009 §4):
#
#   1. Block.SigningMessage = canonical CBOR of BlockHeader.
#   2. BlockEntry CBOR (input to EntryHash = keccak256 over this).
#   3. BlockEntry.Payload = canonical CBOR of the typed op (Transfer,
#      Swap, Withdrawal, Repeg, SessionClose, SessionChallenge).
#   4. FinalizedWithdrawal.SigningMessage (BLS finality preimage).
#   5. BLS G1/G2 serialization + aggregate layout pinned to the Solidity
#      verifiers (BLS.sol / Slasher.sol).
#
# A change to any of these bytes without a coordinated version-byte
# bump (ADR-009 §5) is a schema-family break. This guard runs the
# committed golden-tests in compare mode and fails on any drift. It
# intentionally does NOT pass `-update` — a re-seed is an explicit
# action under an explicit commit message, not CI-automatic.
#
# Usage:
#
#     ./scripts/ci/check-preimage-goldens.sh           # verify only
#
# To regenerate after an intentional, reviewed change:
#
#     go test ./pkg/core/ -run TestGoldens_Preimages -update
#     go test ./pkg/bls/  -run TestGoldens_Preimages -update
#
# Then commit both the source change and the touched
# testdata/goldens/* files in the same commit so the CI run that
# follows sees a clean tree.

set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

echo "check-preimage-goldens: running TestGoldens_Preimages in compare mode..."
go test -run '^TestGoldens_Preimages$' -count=1 ./pkg/core/ ./pkg/bls/

# Defense in depth: if the testdata directory disappeared or was wiped,
# the test would trivially pass in -update mode only. We explicitly
# assert the fixture files exist so an accidental `rm -rf` gets caught.
missing=0
for f in \
    testdata/goldens/preimages/block_header/empty_accounts.golden.hex \
    testdata/goldens/preimages/block_header/empty_accounts.input.json \
    testdata/goldens/preimages/block_header/single_account.golden.hex \
    testdata/goldens/preimages/block_header/single_account.input.json \
    testdata/goldens/preimages/entry_hash/transfer.golden.hex \
    testdata/goldens/preimages/entry_hash/transfer.input.json \
    testdata/goldens/preimages/op_payload/transfer_single_asset.golden.hex \
    testdata/goldens/preimages/op_payload/transfer_single_asset.input.json \
    testdata/goldens/preimages/op_payload/swap.golden.hex \
    testdata/goldens/preimages/op_payload/swap.input.json \
    testdata/goldens/preimages/op_payload/withdrawal.golden.hex \
    testdata/goldens/preimages/op_payload/withdrawal.input.json \
    testdata/goldens/preimages/op_payload/repeg.golden.hex \
    testdata/goldens/preimages/op_payload/repeg.input.json \
    testdata/goldens/preimages/op_payload/session_close.golden.hex \
    testdata/goldens/preimages/op_payload/session_close.input.json \
    testdata/goldens/preimages/op_payload/session_challenge.golden.hex \
    testdata/goldens/preimages/op_payload/session_challenge.input.json \
    testdata/goldens/preimages/finalized_withdrawal/header.golden.hex \
    testdata/goldens/preimages/finalized_withdrawal/header.input.json \
    testdata/goldens/solidity-preimages/bls/g1_identity.golden.hex \
    testdata/goldens/solidity-preimages/bls/g1_generator.golden.hex \
    testdata/goldens/solidity-preimages/bls/g1_random.golden.hex \
    testdata/goldens/solidity-preimages/bls/g2_identity.golden.hex \
    testdata/goldens/solidity-preimages/bls/g2_generator.golden.hex \
    testdata/goldens/solidity-preimages/bls/g2_random.golden.hex \
    testdata/goldens/solidity-preimages/bls/aggregate_sig.golden.hex \
  ; do
    if [[ ! -s "$f" ]]; then
        echo "check-preimage-goldens: MISSING fixture: $f" >&2
        missing=1
    fi
done
if [[ $missing -ne 0 ]]; then
    echo "check-preimage-goldens: one or more fixture files missing; " \
         "re-run the golden tests with -update to regenerate." >&2
    exit 1
fi

echo "check-preimage-goldens: ok"
