package mongodb

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func newURISettings() Settings {
	return Settings{
		Host:               "localhost",
		Port:               27017,
		Username:           "testuser",
		Password:           "testpass123",
		Database:           "mydb",
		AuthDatabase:       "admin",
		IsTLS:              false,
		IsSrv:              false,
		IsDirectConnection: false,
		SSHTunnel:          nil,
	}
}

func newSrvURISettings() Settings {
	settings := newURISettings()
	settings.Host = "cluster0.example.mongodb.net"
	settings.Port = 0
	settings.IsSrv = true

	return settings
}

func TestBuildConnectionURI(t *testing.T) {
	t.Parallel()

	t.Run("builds a standard uri with the database in the path", func(t *testing.T) {
		t.Parallel()

		connectionURI := newURISettings().BuildConnectionURI()

		assert.Equal(t,
			"mongodb://testuser:testpass123@localhost:27017/mydb?authSource=admin&connectTimeoutMS=15000&tls=false",
			connectionURI)
	})

	t.Run("builds an srv uri without a port", func(t *testing.T) {
		t.Parallel()

		connectionURI := newSrvURISettings().BuildConnectionURI()

		assert.Equal(t,
			"mongodb+srv://testuser:testpass123@cluster0.example.mongodb.net/mydb"+
				"?authSource=admin&connectTimeoutMS=15000&tls=false",
			connectionURI)
	})

	t.Run("uses the default port without a port", func(t *testing.T) {
		t.Parallel()
		settings := newURISettings()
		settings.Port = 0

		assert.Contains(t, settings.BuildConnectionURI(), "localhost:27017")
	})

	t.Run("uses admin as the auth database without one", func(t *testing.T) {
		t.Parallel()
		settings := newURISettings()
		settings.AuthDatabase = ""

		assert.Contains(t, settings.BuildConnectionURI(), "authSource=admin")
	})

	t.Run("adds direct connection and insecure tls when both are on", func(t *testing.T) {
		t.Parallel()
		settings := newURISettings()
		settings.IsTLS = true
		settings.IsDirectConnection = true

		connectionURI := settings.BuildConnectionURI()

		assert.Contains(t, connectionURI, "&tls=true&tlsInsecure=true&directConnection=true")
		assert.NotContains(t, connectionURI, "tls=false")
	})

	t.Run("omits direct connection when it is off", func(t *testing.T) {
		t.Parallel()

		assert.NotContains(t, newURISettings().BuildConnectionURI(), "directConnection")
	})

	t.Run("escapes the credentials and the database name", func(t *testing.T) {
		t.Parallel()
		settings := newURISettings()
		settings.Username = "user@corp"
		settings.Password = "p@ss:word/1"
		settings.Database = "db/with#hash?and%percent"

		connectionURI := settings.BuildConnectionURI()

		assert.Contains(t, connectionURI, "mongodb://user%40corp:p%40ss%3Aword%2F1@localhost:27017/")
		assert.Contains(t, connectionURI, "/db%2Fwith%23hash%3Fand%25percent?")
	})
}

func TestBuildRestoreURI(t *testing.T) {
	t.Parallel()

	t.Run("omits the database from the path with a standard connection", func(t *testing.T) {
		t.Parallel()

		restoreURI := newURISettings().BuildRestoreURI()

		assert.Equal(t,
			"mongodb://testuser:testpass123@localhost:27017/?authSource=admin&connectTimeoutMS=15000&tls=false",
			restoreURI)
	})

	t.Run("omits the database from the path with an srv connection", func(t *testing.T) {
		t.Parallel()

		restoreURI := newSrvURISettings().BuildRestoreURI()

		assert.Contains(t, restoreURI, "mongodb+srv://")
		assert.Contains(t, restoreURI, "/?authSource=admin")
		assert.NotContains(t, restoreURI, "/mydb")
	})
}
