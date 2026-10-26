//go:build testhelpers

package logicaltesting

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	restores_core "databasus-backend/internal/features/restores/core"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/lib/sshtunnel"
)

// NewDatabaseSettings wraps the settings of one type into the contract's settings union.
func NewDatabaseSettings(t *testing.T, settings any) apiclient.DatabaseSettings {
	t.Helper()

	var databaseSettings apiclient.DatabaseSettings

	var err error

	switch typedSettings := settings.(type) {
	case apiclient.PostgresqlLogicalSettings:
		typedSettings.Type = apiclient.PostgresqlLogicalSettingsTypePOSTGRESLOGICAL
		err = databaseSettings.FromPostgresqlLogicalSettings(typedSettings)
	case apiclient.MysqlSettings:
		typedSettings.Type = apiclient.MysqlSettingsTypeMYSQL
		err = databaseSettings.FromMysqlSettings(typedSettings)
	case apiclient.MariadbSettings:
		typedSettings.Type = apiclient.MariadbSettingsTypeMARIADB
		err = databaseSettings.FromMariadbSettings(typedSettings)
	case apiclient.MongodbSettings:
		typedSettings.Type = apiclient.MongodbSettingsTypeMONGODB
		err = databaseSettings.FromMongodbSettings(typedSettings)
	default:
		t.Fatalf("unsupported database settings %T", settings)
	}

	require.NoError(t, err)

	return databaseSettings
}

// UpdateDatabaseSettingsViaAPI replaces the settings of the database through the contract and returns
// the database after the change. A secret the settings leave out keeps its stored value.
func UpdateDatabaseSettingsViaAPI(
	t *testing.T,
	router *gin.Engine,
	database apiclient.Database,
	settings any,
	token string,
) apiclient.Database {
	t.Helper()

	w := workspaces_testing.MakeAPIRequest(router, "PUT", "/api/v1/databases/"+database.Id.String(),
		"Bearer "+token, apiclient.DatabaseUpdate{
			Name:        database.Name,
			NotifierIds: database.NotifierIds,
			Settings:    NewDatabaseSettings(t, settings),
		})
	require.Equalf(t, http.StatusOK, w.Code, "update database: %s", w.Body.String())

	var updatedDatabase apiclient.Database
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &updatedDatabase))

	return updatedDatabase
}

// CreateLimitedUserViaAPI creates a limited user on the server of the saved database, which fills
// the secrets its returned settings leave out.
func CreateLimitedUserViaAPI(
	t *testing.T,
	router *gin.Engine,
	database apiclient.Database,
	token string,
) apiclient.LimitedUser {
	t.Helper()

	w := workspaces_testing.MakeAPIRequest(router, "POST", "/api/v1/limited-users", "Bearer "+token,
		apiclient.DatabaseSettingsTarget{
			DatabaseId:  &database.Id,
			Settings:    database.Settings,
			WorkspaceId: database.WorkspaceId,
		})
	require.Equalf(t, http.StatusCreated, w.Code, "create limited user: %s", w.Body.String())

	var limitedUser apiclient.LimitedUser
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &limitedUser))

	return limitedUser
}

func DeleteDatabaseViaAPI(t *testing.T, router *gin.Engine, databaseID uuid.UUID, token string) {
	t.Helper()

	w := workspaces_testing.MakeAPIRequest(router, "DELETE", "/api/v1/databases/"+databaseID.String(),
		"Bearer "+token, nil)
	require.Equalf(t, http.StatusNoContent, w.Code, "delete database: %s", w.Body.String())
}

func NewSshTunnelSettings(tunnelSettings sshtunnel.Settings) apiclient.SshTunnelSettings {
	authType := apiclient.SshAuthType(tunnelSettings.AuthType)

	return apiclient.SshTunnelSettings{
		IsEnabled:            true,
		Host:                 &tunnelSettings.Host,
		Port:                 &tunnelSettings.Port,
		Username:             &tunnelSettings.Username,
		AuthType:             &authType,
		Password:             emptyToNil(tunnelSettings.Password),
		PrivateKey:           emptyToNil(tunnelSettings.PrivateKey),
		PrivateKeyPassphrase: emptyToNil(tunnelSettings.PrivateKeyPassphrase),
	}
}

func NewTargetSshTunnel(tunnelSettings sshtunnel.Settings) restores_core.TargetSshTunnel {
	return restores_core.TargetSshTunnel{
		IsEnabled:            true,
		Host:                 tunnelSettings.Host,
		Port:                 tunnelSettings.Port,
		Username:             tunnelSettings.Username,
		AuthType:             tunnelSettings.AuthType,
		Password:             tunnelSettings.Password,
		PrivateKey:           tunnelSettings.PrivateKey,
		PrivateKeyPassphrase: tunnelSettings.PrivateKeyPassphrase,
	}
}

func emptyToNil(value string) *string {
	if value == "" {
		return nil
	}

	return &value
}
