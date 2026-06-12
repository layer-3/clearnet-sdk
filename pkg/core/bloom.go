package core

// Event bloom filter for per-block log filtering (data_layer.md §5.2).
// 2048-bit (256-byte) bloom with 3 hash functions, matching Ethereum's log bloom.
//
// Only the length constant lives in the SDK — Block.EventBloom is
// [BloomByteLength]byte. The add/test helpers (BloomAdd, BloomTest,
// BlockEventBloom) stay in clearnet; they operate on clearnet-internal Events.
const BloomByteLength = 256
