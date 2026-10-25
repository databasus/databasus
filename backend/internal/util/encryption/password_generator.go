package encryption

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// GenerateComplexPassword creates a password that meets common cloud provider requirements:
// - At least one lowercase letter
// - At least one uppercase letter
// - At least one digit
// - At least one special character
// - 24 characters for security
func GenerateComplexPassword() string {
	const (
		lowercase = "abcdefghijklmnopqrstuvwxyz"
		uppercase = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		digits    = "0123456789"
		special   = "!@#$%^&*()-_=+"
		all       = lowercase + uppercase + digits + special
	)

	password := make([]byte, 24)

	// Ensure at least one character from each required set
	password[0] = randomChar(lowercase)
	password[1] = randomChar(uppercase)
	password[2] = randomChar(digits)
	password[3] = randomChar(special)

	// Fill the rest with random characters from all sets
	for i := 4; i < len(password); i++ {
		password[i] = randomChar(all)
	}

	// Shuffle the password to avoid predictable positions
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
