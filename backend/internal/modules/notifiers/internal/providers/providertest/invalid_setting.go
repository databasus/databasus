package providertest

import (
	"errors"
	"testing"

	"databasus-backend/internal/lib/apierror"
)

// GetInvalidSettingDetail fails the test unless err refuses the setting field of the contract, and
// returns why it refused it.
func GetInvalidSettingDetail(tb testing.TB, err error, field string) string {
	tb.Helper()

	var problem *apierror.Problem
	if !errors.As(err, &problem) || problem.Code != apierror.CodeInvalidRequest {
		tb.Fatalf("expected the setting %s to be refused, got %v", field, err)
	}

	for _, fieldError := range problem.Errors {
		if fieldError.Field == "settings."+field {
			return fieldError.Detail
		}
	}

	tb.Fatalf("expected the setting %s to be refused, got the fields %v", field, problem.Errors)

	return ""
}
