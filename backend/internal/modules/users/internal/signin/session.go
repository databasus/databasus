// Package signin signs users up and in: the first-user rule through accounts, password sign-in
// with the emailed second factor, GitHub and Google sign-in, and the sessions they issue.
package signin

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/modules/users/internal/accounts"
)

const sessionLifetime = 10 * 365 * 24 * time.Hour

const (
	claimSubject         = "sub"
	claimExpiresAt       = "exp"
	claimIssuedAt        = "iat"
	claimRole            = "role"
	claimPasswordVersion = "passwordVersion"
)

type sessionTokens struct {
	masterKey *crypto.MasterKey
	clock     clock.Source
}

type sessionClaims struct {
	userID          uuid.UUID
	passwordVersion uuid.UUID
}

func (t sessionTokens) issue(record accounts.UserRecord) (string, error) {
	now := t.clock.Now()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		claimSubject:         record.ID.String(),
		claimExpiresAt:       now.Add(sessionLifetime).Unix(),
		claimIssuedAt:        now.Unix(),
		claimRole:            string(record.Role),
		claimPasswordVersion: record.PasswordVersion.String(),
	})

	signedToken, err := token.SignedString([]byte(t.masterKey.GetSecret()))
	if err != nil {
		return "", fmt.Errorf("sign the session token: %w", err)
	}

	return signedToken, nil
}

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

	passwordVersionText, isPasswordVersionText := claims[claimPasswordVersion].(string)
	if !isPasswordVersionText {
		return sessionClaims{}, fmt.Errorf("%w: the token has no password version", ErrInvalidSession)
	}

	passwordVersion, err := uuid.Parse(passwordVersionText)
	if err != nil || passwordVersion == uuid.Nil {
		return sessionClaims{}, fmt.Errorf("%w: the token has an invalid password version", ErrInvalidSession)
	}

	return sessionClaims{userID: userID, passwordVersion: passwordVersion}, nil
}
