package workspace

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
