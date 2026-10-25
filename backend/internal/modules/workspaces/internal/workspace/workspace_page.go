package workspace

import (
	"encoding/base64"
	"strings"

	"github.com/google/uuid"
)

const (
	defaultWorkspacePageLimit = 20
	maxWorkspacePageLimit     = 100
)

type WorkspaceListQuery struct {
	// Limit is nil for the default page size.
	Limit *int
	// After is the cursor of the previous page, nil for the first page.
	After *string
}

type WorkspacePage struct {
	Workspaces []ListedWorkspace
	// NextCursor is nil on the last page.
	NextCursor *string
}

// workspaceCursor is the position after the last workspace of a page. The list is ordered by name,
// and the ID orders workspaces with the same name.
type workspaceCursor struct {
	name        string
	workspaceID uuid.UUID
}

// encode puts the ID first, because a name may contain the separator.
func (c workspaceCursor) encode() string {
	return base64.RawURLEncoding.EncodeToString([]byte(c.workspaceID.String() + "|" + c.name))
}

func decodeWorkspaceCursor(encoded string) (workspaceCursor, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return workspaceCursor{}, ErrInvalidCursor
	}

	workspaceIDText, name, isFound := strings.Cut(string(decoded), "|")
	if !isFound {
		return workspaceCursor{}, ErrInvalidCursor
	}

	workspaceID, err := uuid.Parse(workspaceIDText)
	if err != nil {
		return workspaceCursor{}, ErrInvalidCursor
	}

	return workspaceCursor{name: name, workspaceID: workspaceID}, nil
}
