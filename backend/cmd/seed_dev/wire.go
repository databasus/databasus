package main

import "github.com/google/uuid"

// The seed declares only the fields it sends and reads instead of importing the backend's request
// types: importing a feature package runs its dependency wiring, which loads the full backend
// configuration and refuses to start without the database client libraries. The seed test drives
// the real controllers, so a renamed field fails there.

type signInRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type signUpRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

// A sign-in answers with a token, or with the outcome CODE_REQUIRED and no token when the instance
// requires a second factor.
type signInResponse struct {
	Outcome string `json:"outcome"`
	Token   string `json:"token"`
}

type sessionResponse struct {
	Token string `json:"token"`
}

type namedItem struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type listWorkspacesResponse struct {
	Workspaces []namedItem `json:"workspaces"`
}

type createWorkspaceRequest struct {
	Name string `json:"name"`
}

type createLocalStorageRequest struct {
	WorkspaceID  uuid.UUID `json:"workspaceId"`
	Type         string    `json:"type"`
	Name         string    `json:"name"`
	LocalStorage struct{}  `json:"localStorage"`
}

type createDatabaseRequest struct {
	WorkspaceID        uuid.UUID                 `json:"workspaceId"`
	Name               string                    `json:"name"`
	Type               string                    `json:"type"`
	PostgresqlLogical  *postgresqlLogicalSource  `json:"postgresqlLogical,omitempty"`
	PostgresqlPhysical *postgresqlPhysicalSource `json:"postgresqlPhysical,omitempty"`
}

type postgresqlLogicalSource struct {
	Version  string `json:"version"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Database string `json:"database"`
	CpuCount int    `json:"cpuCount"`
}

type postgresqlPhysicalSource struct {
	Version    string `json:"version"`
	BackupType string `json:"backupType"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Username   string `json:"username"`
	Password   string `json:"password"`
}

type makeLogicalBackupRequest struct {
	DatabaseID uuid.UUID `json:"database_id"`
}

type triggerPhysicalBackupRequest struct {
	Type string `json:"type"`
}
