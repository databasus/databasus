package crypto

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

const generatedPasswordLength = 24

// GeneratePassword returns 24 characters with at least one lowercase letter, uppercase letter,
// digit and special character each, which meets the password rules of common cloud providers.
func GeneratePassword() string {
	const (
		lowercase = "abcdefghijklmnopqrstuvwxyz"
		uppercase = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		digits    = "0123456789"
		special   = "!@#$%^&*()-_=+"
		all       = lowercase + uppercase + digits + special
	)

	password := make([]byte, generatedPasswordLength)
	password[0] = randomChar(lowercase)
	password[1] = randomChar(uppercase)
	password[2] = randomChar(digits)
	password[3] = randomChar(special)

	for i := 4; i < len(password); i++ {
		password[i] = randomChar(all)
	}

	// The shuffle keeps the required classes off predictable positions.
	shuffleBytes(password)

	return string(password)
}

func randomChar(charset string) byte {
	return charset[randomIndex(len(charset))]
}

func shuffleBytes(b []byte) {
	for i := len(b) - 1; i > 0; i-- {
		j := randomIndex(i + 1)
		b[i], b[j] = b[j], b[i]
	}
}

// randomIndex panics when crypto/rand fails, as the standard library does itself since Go 1.24: a
// weaker password is worse than a stopped operation.
func randomIndex(upperBound int) int {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(upperBound)))
	if err != nil {
		panic(fmt.Sprintf("read crypto/rand: %v", err))
	}

	return int(n.Int64())
}
