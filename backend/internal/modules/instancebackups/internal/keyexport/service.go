// Package keyexport lets root admin take the existing master key off the server. The key leaves only
// as the attachment of a single-use download, and only root admin's explicit acknowledgment records
// it as saved.
package keyexport

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"databasus-backend/internal/lib/audit"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/instancebackups/internal/configs"
	"databasus-backend/internal/modules/instancebackups/internal/tokens"
	"databasus-backend/internal/modules/users"
)

const (
	downloadLifetime = 60 * time.Second
	keyFileName      = "databasus-master-key.txt"
)

type Deps struct {
	Transactor *database.Transactor
	AuditLog   *audit.Log
	MasterKey  *crypto.MasterKey
	Users      users.Module
	Configs    *configs.Service
	Tokens     *tokens.Service
}

type Service struct {
	transactor *database.Transactor
	auditLog   *audit.Log
	masterKey  *crypto.MasterKey
	users      users.Module
	configs    *configs.Service
	tokens     *tokens.Service
}

func NewService(deps Deps) *Service {
	return &Service{
		transactor: deps.Transactor,
		auditLog:   deps.AuditLog,
		masterKey:  deps.MasterKey,
		users:      deps.Users,
		configs:    deps.Configs,
		tokens:     deps.Tokens,
	}
}

// ExportRequest carries the emailed code; only an instance that requires two-factor sign-in needs it.
type ExportRequest struct {
	Code *string
}

type IssuedExport struct {
	Token     string
	FileName  string
	ExpiresAt time.Time
}

// KeyFile is the downloaded master key, exactly the bytes of the installed key file.
type KeyFile struct {
	FileName string
	Content  []byte
}

func exportBinding() users.IdentityProofBinding {
	return users.IdentityProofBinding{Purpose: users.IdentityProofPurposeMasterKeyExport, Subject: ""}
}

func (s *Service) RequestConfirmation(ctx context.Context, actor users.Actor) (time.Time, error) {
	return s.users.SendIdentityCode(ctx, actor, exportBinding())
}

// CreateExport issues a 60-second single-use download token. Root admin's session is enough unless
// the instance requires two-factor sign-in; then the emailed code is checked too. The audit event
// names the export but never the code, the token or the key.
func (s *Service) CreateExport(ctx context.Context, actor users.Actor, request ExportRequest) (IssuedExport, error) {
	rootAdmin, err := s.users.RequireRootAdmin(ctx, actor)
	if err != nil {
		return IssuedExport{}, err
	}

	if err := s.verifyCodeWhenRequired(ctx, actor, request.Code); err != nil {
		return IssuedExport{}, err
	}

	var issued tokens.IssuedToken

	err = s.transactor.Transaction(ctx, func(txCtx context.Context) error {
		issued, err = s.tokens.Issue(txCtx, tokens.Issuance{
			Purpose:  tokens.PurposeMasterKeyDownload,
			UserID:   rootAdmin.ID,
			CopyID:   nil,
			Lifetime: downloadLifetime,
		})
		if err != nil {
			return err
		}

		return s.auditLog.Record(txCtx, audit.Event{
			Author:      rootAdmin.GetAuditAuthor(),
			WorkspaceID: nil,
			Message:     "Root admin requested a master key export",
		})
	})
	if err != nil {
		return IssuedExport{}, err
	}

	return IssuedExport{Token: issued.Token, FileName: keyFileName, ExpiresAt: issued.ExpiresAt}, nil
}

// Download spends the token and returns the key. The download alone never marks the key saved.
func (s *Service) Download(ctx context.Context, tokenText string) (KeyFile, error) {
	var keyFile KeyFile

	err := s.transactor.Transaction(ctx, func(txCtx context.Context) error {
		token, isConsumed, err := s.tokens.Consume(txCtx, tokenText, tokens.PurposeMasterKeyDownload)
		if err != nil {
			return err
		}

		if !isConsumed {
			return ErrTokenNotUsable
		}

		downloader, isFound, err := s.users.GetUser(txCtx, token.UserID)
		if err != nil {
			return err
		}

		if !isFound || !downloader.IsRootAdmin {
			return ErrTokenNotUsable
		}

		keyFile = KeyFile{FileName: keyFileName, Content: []byte(s.masterKey.GetSecret())}

		return s.auditLog.Record(txCtx, audit.Event{
			Author:      downloader.GetAuditAuthor(),
			WorkspaceID: nil,
			Message:     "Root admin downloaded the master key",
		})
	})
	if err != nil {
		return KeyFile{}, err
	}

	return keyFile, nil
}

func (s *Service) Acknowledge(ctx context.Context, actor users.Actor, isSaved bool) error {
	if _, err := s.users.RequireRootAdmin(ctx, actor); err != nil {
		return err
	}

	if !isSaved {
		return ErrAcknowledgmentNotSaved
	}

	return s.configs.AcknowledgeMasterKey(ctx, actor, s.GetMasterKeySha256())
}

func (s *Service) IsExportCodeRequired(ctx context.Context) (bool, error) {
	return s.users.IsTwoFactorAuthRequired(ctx)
}

// IsAcknowledged compares the acknowledgment with the installed key, so a replaced key, such as one
// a recovery installed, asks root admin again.
func (s *Service) IsAcknowledged(ctx context.Context) (bool, error) {
	config, err := s.configs.GetConfig(ctx)
	if err != nil {
		return false, err
	}

	return config.AcknowledgedMasterKeySha256 != nil &&
		*config.AcknowledgedMasterKeySha256 == s.GetMasterKeySha256(), nil
}

func (s *Service) GetMasterKeySha256() string {
	digest := sha256.Sum256([]byte(s.masterKey.GetSecret()))

	return hex.EncodeToString(digest[:])
}

func (s *Service) verifyCodeWhenRequired(ctx context.Context, actor users.Actor, code *string) error {
	isCodeRequired, err := s.IsExportCodeRequired(ctx)
	if err != nil || !isCodeRequired {
		return err
	}

	return s.users.VerifyIdentityCode(ctx, actor, exportBinding(), code)
}
