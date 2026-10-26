package mongodb

import (
	"path/filepath"

	"databasus-backend/internal/engines/clienttools"
)

type Executable string

const (
	ExecutableMongodump    Executable = "mongodump"
	ExecutableMongorestore Executable = "mongorestore"
)

// ClientTools finds the MongoDB Database Tools under the bundle root. One set of tools serves every
// supported server version.
type ClientTools struct {
	Root string
}

func (c ClientTools) GetExecutable(executable Executable) string {
	return filepath.Join(c.getBinDir(), string(executable))
}

func (c ClientTools) getBinDir() string {
	return filepath.Join(c.Root, "mongodb", "bin")
}

// GetClientToolBundles declares the single tools bundle. It is not named after a release line, so
// there is no version to hold it to, only the requirement that it starts; a missing bundle disables
// MongoDB, so it is not fatal.
func GetClientToolBundles(root string) []clienttools.Bundle {
	return []clienttools.Bundle{{
		Engine:           "mongodb",
		Version:          "tools",
		BinDir:           ClientTools{Root: root}.getBinDir(),
		RequiredCommands: []string{string(ExecutableMongodump), string(ExecutableMongorestore)},
		ReleaseLine:      "",
		ParseVersion:     nil,
		IsFatal:          false,
	}}
}
