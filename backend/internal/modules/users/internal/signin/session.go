// Package signin signs users up and in: the first-user rule through accounts, password sign-in
// with the emailed second factor, GitHub and Google sign-in, and the sessions they issue.
package signin

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"

	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/modules/users/internal/accounts"
)

const sessionLifetime = 10 * 365 * 24 * time.Hour

// The claim names are also written by the test helpers of the old code, which sign tokens for the
// users they insert. A drift makes every old test that signs in fail with 401.
const (
	claimSubject              = "sub"
	claimExpiresAt            = "exp"
	claimIssuedAt             = "iat"
	claimRole                 = "role"
	claimPasswordCreationTime = "passwordCreationTime"
)

// sessionTokens issues and reads the session tokens: HS256 JWTs signed with the master key. A
// token names its user and the moment that user's password was set, so a password change ends
// every earlier session.
type sessionTokens struct {
	masterKey *crypto.MasterKey
	clock     clock.Source
}

type sessionClaims struct {
	userID               uuid.UUID
	passwordCreationTime time.Time
}

func (t sessionTokens) issue(record accounts.UserRecord) (string, error) {
	now := t.clock.Now()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		claimSubject:              record.ID.String(),
		claimExpiresAt:            now.Add(sessionLifetime).Unix(),
		claimIssuedAt:             now.Unix(),
		claimRole:                 string(record.Role),
		claimPasswordCreationTime: record.PasswordCreationTime.Unix(),
	})

	signedToken, err := token.SignedString([]byte(t.masterKey.GetSecret()))
	if err != nil {
		return "", fmt.Errorf("sign the session token: %w", err)
	}

	return signedToken, nil
}

// read checks the signature and the expiry against the clock, and returns the claims the session
// check compares with the user.
func (t sessionTokens) read(token string) (sessionClaims, error) {
	parser := jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithoutClaimsValidation())

	parsedToken, err := parser.Parse(token, func(*jwt.Token) (any, error) {
		return []byte(t.masterKey.GetSecret()), nil
	})
	if err != nil {
		return sessionClaims{}, fmt.Errorf("%w: %w", ErrInvalidSession, err)
	}

	claims, isMapClaims := parsedToken.Claims.(jwt.MapClaims)
	if !isMapClaims {
		return sessionClaims{}, ErrInvalidSession
	}

	expiresAt, isExpiryNumber := claims[claimExpiresAt].(float64)
	if !isExpiryNumber || !t.clock.Now().Before(time.Unix(int64(expiresAt), 0)) {
		return sessionClaims{}, fmt.Errorf("%w: the token expired", ErrInvalidSession)
	}

	subject, isSubjectText := claims[claimSubject].(string)
	if !isSubjectText {
		return sessionClaims{}, fmt.Errorf("%w: the token names no user", ErrInvalidSession)
	}

	userID, err := uuid.Parse(subject)
	if err != nil {
		return sessionClaims{}, fmt.Errorf("%w: the token names no user", ErrInvalidSession)
	}

	passwordCreationTime, isPasswordTimeNumber := claims[claimPasswordCreationTime].(float64)
	if !isPasswordTimeNumber {
		return sessionClaims{}, fmt.Errorf("%w: the token has no password creation time", ErrInvalidSession)
	}

	return sessionClaims{userID: userID, passwordCreationTime: time.Unix(int64(passwordCreationTime), 0)}, nil
}
