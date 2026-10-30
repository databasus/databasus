package backuping

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func Test_BuildBackupLabel_FormatsReadableName(t *testing.T) {
	backupID := uuid.MustParse("9b2c3d4e-5f60-7081-9234-56789abcdef0")
	now := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name         string
		databaseName string
		kind         string
		label        string
	}{
		{
			name:         "clean name and FULL kind",
			databaseName: "production",
			kind:         "FULL",
			label:        "production-FULL-20260530-120000-" + backupID.String(),
		},
		{
			name:         "unsafe characters are sanitized",
			databaseName: "my db/prod",
			kind:         "INCR",
			label:        "my_db-prod-INCR-20260530-120000-" + backupID.String(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.label, buildBackupLabel(tt.databaseName, backupID, now, tt.kind))
		})
	}
}

func Test_BuildObjectName_GivesEachAttemptItsOwnKey(t *testing.T) {
	label := buildBackupLabel("production", uuid.New(), time.Now().UTC(), "FULL")

	databaseID := uuid.New()
	first := buildObjectName(backupObjectName{DatabaseID: databaseID, Label: label, AttemptID: uuid.New()})
	second := buildObjectName(backupObjectName{DatabaseID: databaseID, Label: label, AttemptID: uuid.New()})

	assert.NotEqual(t, first, second, "a retry must not address the object a pending cleanup owns")
	assert.True(t, strings.HasPrefix(first, databaseID.String()+"-"+label+"-"))
	assert.True(t, strings.HasPrefix(second, databaseID.String()+"-"+label+"-"))
}
