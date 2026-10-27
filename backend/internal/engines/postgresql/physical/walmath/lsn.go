package walmath

import (
	"database/sql/driver"
	"fmt"
)

type LSN uint64

func (lsn LSN) String() string {
	return fmt.Sprintf("%X/%X", uint32(lsn>>32), uint32(lsn))
}

func ParseLSN(s string) (LSN, error) {
	var hi, lo uint32

	n, err := fmt.Sscanf(s, "%X/%X", &hi, &lo)
	if err != nil {
		return 0, fmt.Errorf("walmath: invalid LSN %q: %w", s, err)
	}
	if n != 2 {
		return 0, fmt.Errorf("walmath: invalid LSN %q: expected two hex parts, got %d", s, n)
	}

	return LSN(uint64(hi)<<32 | uint64(lo)), nil
}

// Value implements driver.Valuer so LSN round-trips through a PostgreSQL
// pg_lsn column. Returns the canonical "X/X" hex form; pgx forwards it as
// text and pg_lsn accepts that representation implicitly.
func (lsn LSN) Value() (driver.Value, error) {
	return lsn.String(), nil
}

// Scan implements sql.Scanner for pg_lsn columns. Accepts the text form (the
// common case under pgx) and the binary uint64 form for safety.
func (lsn *LSN) Scan(value any) error {
	if value == nil {
		*lsn = 0
		return nil
	}

	switch v := value.(type) {
	case string:
		parsed, err := ParseLSN(v)
		if err != nil {
			return err
		}

		*lsn = parsed
	case []byte:
		parsed, err := ParseLSN(string(v))
		if err != nil {
			return err
		}

		*lsn = parsed
	case uint64:
		*lsn = LSN(v)
	case int64:
		*lsn = LSN(uint64(v))
	default:
		return fmt.Errorf("walmath: unsupported scan source for LSN: %T", value)
	}

	return nil
}
