# Finalized withdrawal verification

Registry policy is required for every signing cluster, including K=1.
`evm.ReadRegistryFinalityPolicy(ctx, reader, registry, confirmations)` requires
`CONFIG()` to return a nonzero Registry-owned Config with finality epoch 1 and
a known K=1..256 checksum. Reads are pinned to one confirmed block hash and
rechecked for an anchor reorg; missing or invalid policy fails closed.

Use a non-upgradeable registry with a frozen Config address and policy row:
the reader verifies the advertised state, not future immutability.

`finality.NewFinalizedWithdrawalVerifier(checker, policy.SigningClusterSize)`
snapshots that trusted K. An unconfigured verifier rejects withdrawals; there
is no implicit quorum. Never select the expected K from an incoming withdrawal.
