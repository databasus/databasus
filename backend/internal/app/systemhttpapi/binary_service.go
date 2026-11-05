package systemhttpapi

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"databasus-backend/internal/lib/apierror"
)

func (s *Service) DownloadAgent(
	_ context.Context,
	architecture AgentArchitecture,
) (DownloadVerificationAgentResponseObject, error) {
	if !architecture.Valid() {
		return nil, apierror.NewInvalidFieldsProblem(
			[]apierror.FieldError{{Field: "arch", Detail: "must be amd64 or arm64"}},
		)
	}

	binaryPath := filepath.Join(s.binaryDirectory, "databasus-verification-agent-linux-"+string(architecture))
	binary, err := os.Open(binaryPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil, &apierror.Problem{
			Code:   apierror.CodeNotFound,
			Detail: "verification agent binary is unavailable",
			Errors: nil,
		}
	}
	if err != nil {
		return nil, fmt.Errorf("open verification agent binary: %w", err)
	}

	info, err := binary.Stat()
	if err != nil {
		_ = binary.Close()
		return nil, fmt.Errorf("inspect verification agent binary: %w", err)
	}
	if !info.Mode().IsRegular() {
		_ = binary.Close()
		return nil, errors.New("verification agent binary is not a regular file")
	}
	return DownloadVerificationAgent200ApplicationoctetStreamResponse{
		Body: binary, ContentLength: info.Size(), Headers: DownloadVerificationAgent200ResponseHeaders{
			ContentDisposition: new(
				"attachment; filename=databasus-verification-agent",
			),
			XDatabasusVersion: &s.version,
		},
	}, nil
}
