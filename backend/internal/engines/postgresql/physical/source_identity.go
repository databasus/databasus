package physical

import (
	"context"
	"fmt"
	"strconv"
	"time"
)

type sourceIdentityComparison struct {
	Expected string
	Actual   string
}

func verifySourceIdentity(identity sourceIdentityComparison) error {
	if identity.Expected == "" {
		return ErrSourceIdentityMissing
	}
	expectedID, expectedErr := parseSourceIdentity(identity.Expected)
	actualID, actualErr := parseSourceIdentity(identity.Actual)
	if expectedErr != nil || actualErr != nil || expectedID == 0 || actualID != expectedID {
		return &SourceIdentityError{
			ExpectedSystemIdentifier: identity.Expected,
			ActualSystemIdentifier:   identity.Actual,
		}
	}
	return nil
}

func parseSourceIdentity(value string) (uint64, error) {
	identifier, err := strconv.ParseUint(value, 10, 64)
	if err == nil {
		return identifier, nil
	}
	// PostgreSQL exposes the unsigned system identifier as bigint, so identifiers above
	// the int64 range arrive negative and the cast restores their original bits.
	// nosemgrep: trailofbits.go.string-to-int-signedness-cast.string-to-int-signedness-cast
	signedIdentifier, err := strconv.ParseInt(value, 10, 64)
	return uint64(signedIdentifier), err
}

func (s *Source) OpenVerifiedInspection(ctx context.Context, expectedSystemIdentifier string) (*Inspection, error) {
	if expectedSystemIdentifier == "" {
		return nil, ErrSourceIdentityMissing
	}
	inspection, err := s.OpenInspection(ctx)
	if err != nil {
		return nil, err
	}
	var actualSystemIdentifier string
	err = inspection.conn.QueryRow(ctx, "SELECT system_identifier::text FROM pg_control_system()").
		Scan(&actualSystemIdentifier)
	if err == nil {
		err = verifySourceIdentity(
			sourceIdentityComparison{Expected: expectedSystemIdentifier, Actual: actualSystemIdentifier},
		)
	}
	if err != nil {
		closeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		_ = inspection.Close(closeCtx)
		return nil, fmt.Errorf("verify inspection source: %w", err)
	}
	return inspection, nil
}
