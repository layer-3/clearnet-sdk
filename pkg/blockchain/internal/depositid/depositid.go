// Package depositid parses deposit IDs of the form "<prefix>:<n>".
package depositid

import (
	"strconv"
	"strings"
)

// ParseIndex returns n when id is exactly prefix + ":" + the canonical decimal
// form of n (no sign, no leading zeros) and n fits in bitSize bits.
func ParseIndex(id, prefix string, bitSize int) (uint64, bool) {
	digits, ok := strings.CutPrefix(id, prefix+":")
	if !ok {
		return 0, false
	}
	n, err := strconv.ParseUint(digits, 10, bitSize)
	if err != nil || strconv.FormatUint(n, 10) != digits {
		return 0, false
	}
	return n, true
}
