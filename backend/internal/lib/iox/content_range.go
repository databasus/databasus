package iox

import (
	"fmt"
	"strconv"
	"strings"
)

func GetRangeStartOfContentRange(contentRange string) (int64, bool) {
	span, isBytesUnit := strings.CutPrefix(contentRange, "bytes ")
	if !isBytesUnit {
		return 0, false
	}

	start, _, isSplit := strings.Cut(span, "-")
	if !isSplit {
		return 0, false
	}

	parsed, err := strconv.ParseInt(strings.TrimSpace(start), 10, 64)
	if err != nil {
		return 0, false
	}

	return parsed, true
}

// A resumed read must not begin before the offset already handed out: replaying bytes into a
// stateful decryptor corrupts the stream. Content-Range is the positive proof the backend applied
// the range, so a reply that cannot be checked is refused rather than trusted.
func VerifyContentRangeStart(contentRange string, expectedOffsetBytes int64) error {
	rangeStart, isParsed := GetRangeStartOfContentRange(contentRange)
	if !isParsed {
		return fmt.Errorf(
			"%w: resumed at offset %d but the response carries no usable Content-Range",
			ErrRangeNotHonoured, expectedOffsetBytes,
		)
	}

	if rangeStart != expectedOffsetBytes {
		return fmt.Errorf(
			"%w: resumed at offset %d but the response starts at %d",
			ErrRangeNotHonoured, expectedOffsetBytes, rangeStart,
		)
	}

	return nil
}
