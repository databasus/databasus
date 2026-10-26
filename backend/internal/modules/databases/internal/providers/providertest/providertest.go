// Package providertest holds what the tests of the database providers share: a field encryptor with a
// key of its own and the reading of a refused setting.
package providertest

import (
	"errors"
	"path/filepath"
	"testing"

	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/lib/crypto"
)

// NewFieldEncryptor returns an encryptor under a master key of the test's own.
func NewFieldEncryptor(tb testing.TB) *crypto.FieldEncryptor {
	tb.Helper()

	masterKey, err := crypto.OpenOrCreateMasterKey(filepath.Join(tb.TempDir(), "secret.key"))
	if err != nil {
		tb.Fatalf("create the master key: %v", err)
	}

	return crypto.NewFieldEncryptor(masterKey)
}

// GetRefusedField returns the field an invalid_request problem names, and fails the test for any
// other error.
func GetRefusedField(tb testing.TB, err error) string {
	tb.Helper()

	var problem *apierror.Problem
	if !errors.As(err, &problem) || problem.Code != apierror.CodeInvalidRequest || len(problem.Errors) != 1 {
		tb.Fatalf("expected an invalid_request problem naming one field, got %v", err)
	}

	return problem.Errors[0].Field
}
