package totp

import (
	"crypto/hmac"
	"crypto/sha1" //nolint:gosec // RFC 6238 apps default to HMAC-SHA1; HMAC does not rely on SHA-1 collision resistance.
	"crypto/subtle"
	"encoding/binary"
	"fmt"
	"time"
)

const (
	codeDigitCount = 6
	codeModulus    = 1_000_000
	stepSeconds    = 30
	// allowedStepDrift accepts the code of the step before and after the current one, so a phone
	// whose clock is up to about thirty seconds off still works.
	allowedStepDrift = 1
)

func GetStep(moment time.Time) int64 {
	return moment.Unix() / stepSeconds
}

func ComputeCode(secret string, step int64) (string, error) {
	key, err := decodeSecret(secret)
	if err != nil {
		return "", err
	}

	return computeCode(key, step), nil
}

// FindMatchingStep reports the latest step within the allowed drift of now whose code matches.
// Every candidate is compared in constant time, so the answer time does not tell which step
// matched.
func FindMatchingStep(secret, code string, now time.Time) (int64, bool, error) {
	key, err := decodeSecret(secret)
	if err != nil {
		return 0, false, err
	}

	currentStep := GetStep(now)
	matchingStep := int64(0)
	isMatched := false

	for step := currentStep - allowedStepDrift; step <= currentStep+allowedStepDrift; step++ {
		if subtle.ConstantTimeCompare([]byte(computeCode(key, step)), []byte(code)) == 1 {
			matchingStep = step
			isMatched = true
		}
	}

	return matchingStep, isMatched, nil
}

func IsCodeFormat(value string) bool {
	if len(value) != codeDigitCount {
		return false
	}

	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}

	return true
}

// computeCode is the dynamic truncation of RFC 4226 over the step counter.
func computeCode(key []byte, step int64) string {
	counter := make([]byte, 8)
	binary.BigEndian.PutUint64(counter, uint64(step)) //nolint:gosec // A step of a real clock is never negative.

	mac := hmac.New(sha1.New, key)
	_, _ = mac.Write(counter)
	digest := mac.Sum(nil)

	offset := digest[len(digest)-1] & 0x0f
	truncated := binary.BigEndian.Uint32(digest[offset:offset+4]) & 0x7fffffff

	return fmt.Sprintf("%0*d", codeDigitCount, truncated%codeModulus)
}
