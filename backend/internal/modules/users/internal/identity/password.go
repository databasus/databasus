package identity

import (
	"fmt"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

// MinPasswordLength applies to every way a password is set: sign-up, a password change, a reset
// and the console. It counts characters, not bytes.
const MinPasswordLength = 8

func IsPasswordLongEnough(password string) bool {
	return utf8.RuneCountInString(password) >= MinPasswordLength
}

func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash the password: %w", err)
	}

	return string(hashedPassword), nil
}
