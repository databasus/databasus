package apptest

import (
	"fmt"
	"runtime"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/lib/database"
)

const isolationMarkerVersion = 99990101000000

func TestNew(t *testing.T) {
	t.Run("passes the response through the middleware of the server", func(t *testing.T) {
		testApp := New(t)
		client := apiclient.New(t, testApp.URL)

		response, err := client.GetVersionWithResponse(t.Context())

		require.NoError(t, err)
		assert.NotEmpty(t, response.HTTPResponse.Header.Get("X-Request-Id"))
	})

	t.Run("writes a row into the database of its own test", func(t *testing.T) {
		testApp := New(t)

		db, err := database.Open(t.Context(), testApp.databaseDsn)
		require.NoError(t, err)
		t.Cleanup(func() { closeDatabase(t, db) })

		require.NoError(t, db.WithContext(t.Context()).Exec(
			"INSERT INTO goose_db_version (version_id, is_applied) VALUES (?, TRUE)", isolationMarkerVersion,
		).Error)
	})

	t.Run("does not see the row a previous test wrote when built after it", func(t *testing.T) {
		testApp := New(t)

		db, err := database.Open(t.Context(), testApp.databaseDsn)
		require.NoError(t, err)
		t.Cleanup(func() { closeDatabase(t, db) })

		var markerCount int64
		require.NoError(t, db.WithContext(t.Context()).Raw(
			"SELECT COUNT(*) FROM goose_db_version WHERE version_id = ?", isolationMarkerVersion,
		).Scan(&markerCount).Error)
		assert.Zero(t, markerCount)
	})

	t.Run("fails the test when another test application is alive", func(t *testing.T) {
		New(t)
		secondTest := &fatalRecorder{TB: t, failure: "", mutex: sync.Mutex{}}

		isFinished := make(chan struct{})
		go func() {
			defer close(isFinished)
			New(secondTest)
		}()
		<-isFinished

		assert.Contains(t, secondTest.getFailure(), "run one at a time")
	})
}

func TestRunJobPass(t *testing.T) {
	t.Run("fails the test when no job has the name", func(t *testing.T) {
		testApp := New(t)
		recordingTest := &fatalRecorder{TB: t, failure: "", mutex: sync.Mutex{}}

		isFinished := make(chan struct{})
		go func() {
			defer close(isFinished)
			testApp.RunJobPass(recordingTest, "no_such_job")
		}()
		<-isFinished

		assert.Contains(t, recordingTest.getFailure(), "no_such_job")
	})
}

func closeDatabase(t *testing.T, db *gorm.DB) {
	t.Helper()

	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())
}

// fatalRecorder stops the goroutine that called Fatalf, as testing.T does, and keeps the message
// for the surrounding test instead of failing it.
type fatalRecorder struct {
	testing.TB

	failure string
	mutex   sync.Mutex
}

func (r *fatalRecorder) Fatalf(format string, args ...any) {
	r.mutex.Lock()
	r.failure = fmt.Sprintf(format, args...)
	r.mutex.Unlock()

	runtime.Goexit()
}

func (r *fatalRecorder) getFailure() string {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return r.failure
}
