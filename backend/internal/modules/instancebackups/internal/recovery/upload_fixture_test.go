package recovery

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"mime/multipart"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/diskusage"
)

const defaultMaxArchiveBytes = 1 << 30

// uploadTarget is an initialized installation that takes recovery uploads.
type uploadTarget struct {
	target

	service *UploadService
	clock   *clock.Manual
}

type uploadTargetSpec struct {
	release         string
	maxArchiveBytes int64
	inspectDisk     func(string) (diskusage.Usage, error)
}

func newUploadTarget(t *testing.T) uploadTarget {
	t.Helper()

	return newUploadTargetWith(t, uploadTargetSpec{
		release: testRelease, maxArchiveBytes: defaultMaxArchiveBytes, inspectDisk: diskusage.Get,
	})
}

func newUploadTargetWith(t *testing.T, spec uploadTargetSpec) uploadTarget {
	t.Helper()

	initializedTarget := newInitializedTarget(t)
	directory := initializedTarget.getDirectory()
	require.NoError(t, directory.Ensure())

	manualClock := clock.NewManual(time.Date(2026, 11, 14, 13, 0, 0, 0, time.UTC))
	logger := slog.New(slog.NewJSONHandler(initializedTarget.logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	journal := NewJournalStore(directory)

	return uploadTarget{
		target: initializedTarget,
		clock:  manualClock,
		service: NewUploadService(UploadServiceDeps{
			Directory: directory,
			Journal:   journal,
			Server:    initializedTarget.server,
			Preparer: NewPreparer(PreparerDeps{
				Server:      initializedTarget.server,
				ClientTools: getClientTools(t),
				Release:     spec.release,
				Bundle:      getTestBundle(t),
				Normalizers: getModuleNormalizers(),
				Clock:       manualClock,
				Logger:      logger,
			}),
			Publisher: NewPublisher(PublisherDeps{
				Directory:     directory,
				Journal:       journal,
				Server:        initializedTarget.server,
				SecretKeyPath: initializedTarget.secretKeyPath,
				Clock:         manualClock,
				Logger:        logger,
			}),
			Users:           nil,
			MaxArchiveBytes: spec.maxArchiveBytes,
			RunningRelease:  testRelease,
			Clock:           manualClock,
			Logger:          logger,
			InspectDisk:     spec.inspectDisk,
		}),
	}
}

// uploadPart is one field of a recovery upload, in the order the client sends it.
type uploadPart struct {
	name     string
	fileName string
	content  []byte
}

// newUploadParts is what the import form sends over HTTP: the acknowledgment, the key, then the
// archive.
func newUploadParts(t *testing.T, archivePath string, keyContent []byte) []uploadPart {
	t.Helper()

	archiveContent, err := os.ReadFile(archivePath)
	require.NoError(t, err)

	return []uploadPart{
		{name: acknowledgmentField, fileName: "", content: []byte("true")},
		{name: masterKeyField, fileName: "", content: keyContent},
		{name: archiveField, fileName: "databasus.dbbackup", content: archiveContent},
	}
}

func newUploadBody(t *testing.T, parts []uploadPart) *multipart.Reader {
	t.Helper()

	var body bytes.Buffer

	writer := multipart.NewWriter(&body)
	for _, part := range parts {
		writePart(t, writer, part)
	}

	require.NoError(t, writer.Close())

	return multipart.NewReader(&body, writer.Boundary())
}

// newInterruptedUploadBody ends the request in the middle of the archive, as a client that
// disconnects does.
func newInterruptedUploadBody(t *testing.T, parts []uploadPart) *multipart.Reader {
	t.Helper()

	reader, writer := io.Pipe()
	multipartWriter := multipart.NewWriter(writer)

	go func() {
		for _, part := range parts[:len(parts)-1] {
			writePart(t, multipartWriter, part)
		}

		lastPart := parts[len(parts)-1]
		partWriter, err := multipartWriter.CreateFormFile(lastPart.name, lastPart.fileName)
		if err == nil {
			_, err = partWriter.Write(lastPart.content[:len(lastPart.content)/2])
		}

		_ = writer.CloseWithError(errors.Join(errClientDisconnected, err))
	}()

	return multipart.NewReader(reader, multipartWriter.Boundary())
}

var errClientDisconnected = errors.New("client disconnected")

func writePart(t *testing.T, writer *multipart.Writer, part uploadPart) {
	t.Helper()

	var partWriter io.Writer
	var err error

	if part.fileName == "" {
		partWriter, err = writer.CreateFormField(part.name)
	} else {
		partWriter, err = writer.CreateFormFile(part.name, part.fileName)
	}

	if err == nil {
		_, err = partWriter.Write(part.content)
	}

	if err != nil {
		t.Errorf("write the upload part %s: %v", part.name, err)
	}
}

func (ut uploadTarget) upload(t *testing.T, parts []uploadPart) (Recovery, error) {
	t.Helper()

	return ut.service.receive(t.Context(), newUploadBody(t, parts))
}

// uploadAndPrepare uploads and runs the preparation pass, and returns the recovery as root admin
// then sees it.
func (ut uploadTarget) uploadAndPrepare(t *testing.T, parts []uploadPart) Recovery {
	t.Helper()

	uploadedRecovery, err := ut.upload(t, parts)
	require.NoError(t, err)
	require.Equal(t, StatusValidating, uploadedRecovery.Status)
	ut.runPass(t)

	return ut.getRecovery(t, uploadedRecovery)
}

func (ut uploadTarget) runPass(t *testing.T) {
	t.Helper()

	require.NoError(t, ut.service.RunPass(t.Context(), ut.clock.Now()))
}

func (ut uploadTarget) getRecovery(t *testing.T, knownRecovery Recovery) Recovery {
	t.Helper()

	foundRecovery, isFound := findRecovery(ut.readJournal(t), knownRecovery.ID, nil, testRelease)
	require.True(t, isFound)

	return foundRecovery
}

// assertNothingStaged checks that no recovery left a journal operation, a staged file or a staging
// database behind, and that the active installation kept its key and workspaces.
func (ut uploadTarget) assertNothingStaged(t *testing.T, activeKey []byte) {
	t.Helper()

	assert.Nil(t, ut.readJournal(t).Operation)
	operationDirectories, err := filepath.Glob(filepath.Join(ut.getDirectory().root, "operations", "*"))
	require.NoError(t, err)
	assert.Empty(t, operationDirectories)
	assert.Empty(t, ut.listRecoveryDatabases(t))
	assert.Equal(t, activeKey, ut.readKey(t))
	assert.Empty(t, ut.listWorkspaceNames(t))
}

func requireFieldProblem(t *testing.T, err error, field string) {
	t.Helper()

	var problem *apierror.Problem
	require.ErrorAs(t, err, &problem)
	assert.Equal(t, apierror.CodeInvalidRequest, problem.Code)
	require.Len(t, problem.Errors, 1)
	assert.Equal(t, field, problem.Errors[0].Field)
}
