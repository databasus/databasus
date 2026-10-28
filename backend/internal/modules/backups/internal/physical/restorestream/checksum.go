package restorestream

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"sort"
	"strings"
)

// checksumLedger accumulates "<sha256>  <path>" lines (sha256sum -c format) for
// every regular file written into the tar, so a trailing MANIFEST.sha256 lets
// the user verify the transfer before reconstructing the cluster.
type checksumLedger struct {
	entries  map[string]string
	skipMode bool
}

func newChecksumLedger() *checksumLedger {
	return &checksumLedger{entries: make(map[string]string)}
}

// skip returns a ledger that records nothing — used while writing
// MANIFEST.sha256 itself, which must not list its own checksum.
func (l *checksumLedger) skip() *checksumLedger {
	return &checksumLedger{entries: l.entries, skipMode: true}
}

func (l *checksumLedger) begin(string) hash.Hash {
	return sha256.New()
}

func (l *checksumLedger) commit(name string, hasher hash.Hash) {
	if l.skipMode {
		return
	}

	l.entries[name] = hex.EncodeToString(hasher.Sum(nil))
}

func (l *checksumLedger) render() []byte {
	names := make([]string, 0, len(l.entries))
	for name := range l.entries {
		names = append(names, name)
	}

	sort.Strings(names)

	var b strings.Builder
	for _, name := range names {
		fmt.Fprintf(&b, "%s  %s\n", l.entries[name], name)
	}

	return []byte(b.String())
}
