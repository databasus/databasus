package container

import (
	"github.com/distribution/reference"
)

// isSameImageRepository compares two repository names the way Docker resolves
// them, so "postgres", "library/postgres" and "docker.io/library/postgres"
// name one repository.
func isSameImageRepository(first, second string) bool {
	firstName, err := reference.ParseNormalizedNamed(first)
	if err != nil {
		return false
	}

	secondName, err := reference.ParseNormalizedNamed(second)
	if err != nil {
		return false
	}

	return firstName.Name() == secondName.Name()
}

func isOfficialPostgresRepository(repository string) bool {
	return isSameImageRepository(repository, stockEngineImageRepo)
}
