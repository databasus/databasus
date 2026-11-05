package mysqlfamily

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ClientConnection is what a MySQL family client tool connects with. SSLOption is the .my.cnf line
// that sets TLS, which MySQL and MariaDB spell differently.
type ClientConnection struct {
	Host      string
	Port      int
	Username  string
	Password  string
	SSLOption string
}

// clientConfigFile is a .my.cnf with the connection's credentials, read by the tool through
// --defaults-file so the password never appears in the arguments.
type clientConfigFile struct {
	dir  string
	path string
}

// writeClientConfigFile writes into the OS temp directory, because some filesystems, such as ZFS on
// TrueNAS, ignore chmod and the tools then refuse a file with group or world access.
func writeClientConfigFile(connection ClientConnection) (*clientConfigFile, error) {
	dir, err := os.MkdirTemp(os.TempDir(), "mycnf_*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	err = os.Chmod(dir, 0o700)
	if err != nil {
		_ = os.RemoveAll(dir)

		return nil, fmt.Errorf("failed to set temp directory permissions: %w", err)
	}

	configFile := &clientConfigFile{dir: dir, path: filepath.Join(dir, ".my.cnf")}

	content := fmt.Sprintf("[client]\nuser=%s\npassword=\"%s\"\nhost=%s\nport=%d\n%s\n",
		connection.Username,
		escapeClientConfigPassword(connection.Password),
		connection.Host,
		connection.Port,
		connection.SSLOption,
	)

	err = os.WriteFile(configFile.path, []byte(content), 0o600)
	if err != nil {
		_ = os.RemoveAll(dir)

		return nil, fmt.Errorf("failed to write .my.cnf: %w", err)
	}

	return configFile, nil
}

func (f *clientConfigFile) getDefaultsFileArg() string {
	return "--defaults-file=" + f.path
}

func (f *clientConfigFile) remove() {
	_ = os.RemoveAll(f.dir)
}

// escapeClientConfigPassword escapes a password for the double-quoted value of a .my.cnf file.
func escapeClientConfigPassword(password string) string {
	password = strings.ReplaceAll(password, "\\", "\\\\")
	password = strings.ReplaceAll(password, "\"", "\\\"")

	return password
}
