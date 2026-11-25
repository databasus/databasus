package container

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
)

// The release workflow and scripts/image_pins.py read this file too, so the
// image build, its tags and the test catalog follow the same versions.
//
//go:embed postgres_versions.json
var pinnedPostgresVersionsDocument []byte

// PinnedPostgresVersions holds the exact PostgreSQL server version this agent
// release restores into for each major.
type PinnedPostgresVersions struct {
	versionByMajor map[string]string
}

func LoadPinnedPostgresVersions() (PinnedPostgresVersions, error) {
	return parsePinnedPostgresVersions(pinnedPostgresVersionsDocument)
}

func parsePinnedPostgresVersions(document []byte) (PinnedPostgresVersions, error) {
	var versionByMajor map[string]string
	if err := json.Unmarshal(document, &versionByMajor); err != nil {
		return PinnedPostgresVersions{}, fmt.Errorf("decode pinned PostgreSQL versions: %w", err)
	}

	if len(versionByMajor) == 0 {
		return PinnedPostgresVersions{}, fmt.Errorf("pinned PostgreSQL versions are empty")
	}

	for major, version := range versionByMajor {
		if !isCanonicalNumber(major) || major == "0" {
			return PinnedPostgresVersions{}, fmt.Errorf("pinned PostgreSQL versions: %q is not a major", major)
		}

		minor, hasMajorPrefix := strings.CutPrefix(version, major+".")
		if !hasMajorPrefix || !isCanonicalNumber(minor) {
			return PinnedPostgresVersions{}, fmt.Errorf(
				"pinned PostgreSQL versions: %q is not a release of major %s", version, major,
			)
		}
	}

	return PinnedPostgresVersions{versionByMajor: versionByMajor}, nil
}

func (v PinnedPostgresVersions) GetPinnedPostgresVersion(pgMajor string) (string, bool) {
	version, isPinned := v.versionByMajor[pgMajor]

	return version, isPinned
}

func (v PinnedPostgresVersions) IsPinnedPostgresVersion(version string) bool {
	return slices.Contains(slices.Collect(maps.Values(v.versionByMajor)), version)
}

func (v PinnedPostgresVersions) GetMajors() []string {
	return slices.Sorted(maps.Keys(v.versionByMajor))
}

func isCanonicalNumber(text string) bool {
	number, err := strconv.Atoi(text)

	return err == nil && number >= 0 && strconv.Itoa(number) == text
}
