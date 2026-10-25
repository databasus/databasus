//go:build !testhelpers

package encryption

func loadWithoutApp() *installedEncryption {
	panic("encryption.GetMasterKey or GetFieldEncryptor is called before app.New installed the master key")
}
