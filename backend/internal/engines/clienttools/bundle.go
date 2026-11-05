// Package clienttools holds what the client tool sets of every engine share: the description of a
// bundle, where the bundles live, the release-line comparison and the check that every client of
// every bundle starts.
package clienttools

// Bundle is one client tool set an engine ships: where it lives, what it must hold and how its
// clients report their version.
type Bundle struct {
	Engine           string
	Version          string
	BinDir           string
	RequiredCommands []string
	// ReleaseLine is the line the clients must report; empty for a bundle that is not named after a
	// version, whose clients only have to start.
	ReleaseLine  string
	ParseVersion func(output string) (string, error)
	// IsFatal marks a bundle the instance cannot run without. An instance that backs up no MySQL
	// database has no reason to refuse to start over a missing mysqldump, so only PostgreSQL is.
	IsFatal bool
}
