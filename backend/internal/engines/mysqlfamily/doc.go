// Package mysqlfamily holds the grant reading, the backup-privilege rules and the read-only user
// that MySQL and MariaDB share. Both speak the same GRANT syntax and both dump through the same tool
// family, so keeping one copy is what keeps the two engine packages from drifting apart.
//
// What stays in the engine packages is what genuinely differs: how active roles are read (MySQL
// expands them with a USING clause, MariaDB has a single active role and no such clause), the
// connection options, the write privileges each server knows and the version enums, which are
// distinct types.
package mysqlfamily
