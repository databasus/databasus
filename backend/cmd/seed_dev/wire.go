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

// namedItemPage is one page of a contract list, such as the workspaces, the storages or the databases.
type namedItemPage struct {
	Items      []namedItem `json:"items"`
	NextCursor *string     `json:"nextCursor"`
}

type createWorkspaceRequest struct {
	Name string `json:"name"`
}

type createStorageRequest struct {
	WorkspaceID uuid.UUID            `json:"workspaceId"`
	Name        string               `json:"name"`
	Settings    localStorageSettings `json:"settings"`
}

type localStorageSettings struct {
	Type string `json:"type"`
}

type createDatabaseRequest struct {
	WorkspaceID uuid.UUID   `json:"workspaceId"`
	Name        string      `json:"name"`
	NotifierIDs []uuid.UUID `json:"notifierIds"`
	Settings    any         `json:"settings"`
}

type disabledSSHTunnel struct {
	IsEnabled bool `json:"isEnabled"`
}

type postgresqlLogicalSettings struct {
	Type               string            `json:"type"`
	Host               string            `json:"host"`
	Port               int               `json:"port"`
	Username           string            `json:"username"`
	Password           string            `json:"password"`
	Database           string            `json:"database"`
	SslMode            string            `json:"sslMode"`
	SSHTunnel          disabledSSHTunnel `json:"sshTunnel"`
	IncludeSchemas     []string          `json:"includeSchemas"`
	ExcludeTables      []string          `json:"excludeTables"`
	ExcludeTableData   []string          `json:"excludeTableData"`
	CPUCount           int               `json:"cpuCount"`
	IsSkipUserMappings bool              `json:"isSkipUserMappings"`
}

type postgresqlPhysicalSettings struct {
	Type       string            `json:"type"`
	BackupType string            `json:"backupType"`
	Host       string            `json:"host"`
	Port       int               `json:"port"`
	Username   string            `json:"username"`
	Password   string            `json:"password"`
	SslMode    string            `json:"sslMode"`
	SSHTunnel  disabledSSHTunnel `json:"sshTunnel"`
}

type startLogicalBackupRequest struct {
	DatabaseID uuid.UUID `json:"databaseId"`
}
