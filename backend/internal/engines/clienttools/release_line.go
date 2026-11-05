package clienttools

import (
	"fmt"
	"strconv"
	"strings"
)

// A version identity names a release line rather than an exact release, so it may carry fewer
// components than the version a server or a client reports: "26" is every MySQL 26.x, "10.6" every
// MariaDB 10.6.x.

// CompareReleaseLines orders two version identities numerically and returns a negative number,
// zero or a positive number the way strings.Compare does. Ordering by the numbers means an identity
// added later sorts correctly without an entry in any table.
func CompareReleaseLines(left, right string) (int, error) {
	leftNumbers, err := parseReleaseLine(left)
	if err != nil {
		return 0, err
	}

	rightNumbers, err := parseReleaseLine(right)
	if err != nil {
		return 0, err
	}

	for i := 0; i < len(leftNumbers) || i < len(rightNumbers); i++ {
		leftNumber, rightNumber := 0, 0
		if i < len(leftNumbers) {
			leftNumber = leftNumbers[i]
		}

		if i < len(rightNumbers) {
			rightNumber = rightNumbers[i]
		}

		if leftNumber != rightNumber {
			return leftNumber - rightNumber, nil
		}
	}

	return 0, nil
}

// isVersionOnReleaseLine reports whether a reported version, such as the "10.6.28" a client prints,
// belongs to the line an identity names. A client may sit at a later patch than the line's own
// digits suggest; it may never come from a different line.
func isVersionOnReleaseLine(reported, line string) bool {
	reportedNumbers, err := parseReleaseLine(reported)
	if err != nil {
		return false
	}

	lineNumbers, err := parseReleaseLine(line)
	if err != nil {
		return false
	}

	if len(lineNumbers) > len(reportedNumbers) {
		return false
	}

	for i, lineNumber := range lineNumbers {
		if reportedNumbers[i] != lineNumber {
			return false
		}
	}

	return true
}

func parseReleaseLine(identity string) ([]int, error) {
	if identity == "" {
		return nil, ErrEmptyVersionIdentity
	}

	parts := strings.Split(identity, ".")
	numbers := make([]int, 0, len(parts))

	for _, part := range parts {
		number, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("%w: %q", ErrInvalidVersionIdentity, identity)
		}

		numbers = append(numbers, number)
	}

	return numbers, nil
}
