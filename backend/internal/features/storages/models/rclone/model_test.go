package rclone_storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

func Test_ParseConfigContent_SingleRemote_ParsedCorrectly(t *testing.T) {
	content := `[myremote]
type = s3
provider = AWS
access_key_id = AKIAIOSFODNN7EXAMPLE
secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
region = us-east-1`

	sections, err := parseConfigContent(content)

	require.NoError(t, err)
	require.Len(t, sections, 1)
	assert.Equal(t, "s3", sections["myremote"]["type"])
	assert.Equal(t, "AWS", sections["myremote"]["provider"])
	assert.Equal(t, "AKIAIOSFODNN7EXAMPLE", sections["myremote"]["access_key_id"])
	assert.Equal(t, "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY", sections["myremote"]["secret_access_key"])
	assert.Equal(t, "us-east-1", sections["myremote"]["region"])
}

func Test_ParseConfigContent_MultipleRemotes_AllParsed(t *testing.T) {
	content := `[remote1]
type = s3
region = us-east-1

[remote2]
type = drive
client_id = abc123`

	sections, err := parseConfigContent(content)

	require.NoError(t, err)
	assert.Len(t, sections, 2)
	assert.Equal(t, "s3", sections["remote1"]["type"])
	assert.Equal(t, "us-east-1", sections["remote1"]["region"])
	assert.Equal(t, "drive", sections["remote2"]["type"])
	assert.Equal(t, "abc123", sections["remote2"]["client_id"])
}

func Test_ParseConfigContent_EmptyContent_ReturnsEmptyMap(t *testing.T) {
	sections, err := parseConfigContent("")

	require.NoError(t, err)
	assert.Empty(t, sections)
}

func Test_ParseConfigContent_CommentsAndBlankLines_Ignored(t *testing.T) {
	content := `# This is a comment
; Another comment

[myremote]
type = s3

# inline comment line
region = eu-west-1`

	sections, err := parseConfigContent(content)

	require.NoError(t, err)
	require.Len(t, sections, 1)
	assert.Equal(t, "s3", sections["myremote"]["type"])
	assert.Equal(t, "eu-west-1", sections["myremote"]["region"])
}

func Test_ParseConfigContent_ValueWithEqualsSign_PreservesFullValue(t *testing.T) {
	content := `[myremote]
type = s3
secret_access_key = abc=def=ghi`

	sections, err := parseConfigContent(content)

	require.NoError(t, err)
	assert.Equal(t, "abc=def=ghi", sections["myremote"]["secret_access_key"])
}

func Test_ParseConfigContent_KeyWithoutValue_EmptyString(t *testing.T) {
	content := `[myremote]
type =
provider = AWS`

	sections, err := parseConfigContent(content)

	require.NoError(t, err)
	assert.Equal(t, "", sections["myremote"]["type"])
	assert.Equal(t, "AWS", sections["myremote"]["provider"])
}

func Test_ParseConfigContent_KeyValueOutsideSection_Ignored(t *testing.T) {
	content := `orphan_key = orphan_value
[myremote]
type = s3`

	sections, err := parseConfigContent(content)

	require.NoError(t, err)
	assert.Len(t, sections, 1)
	assert.Equal(t, "s3", sections["myremote"]["type"])
}

func Test_ParseConfigContent_WhitespaceAroundKeysAndValues_Trimmed(t *testing.T) {
	content := `[myremote]
  type   =   s3
  region   =   us-west-2  `

	sections, err := parseConfigContent(content)

	require.NoError(t, err)
	assert.Equal(t, "s3", sections["myremote"]["type"])
	assert.Equal(t, "us-west-2", sections["myremote"]["region"])
}

// buildFs strips a leading slash from RemotePath, so an absolute root would become
// relative to the working directory. The test moves there instead and leaves the
// remote path empty, which puts the rclone root on the temporary directory.
func newLocalRcloneStorage(t *testing.T) (*RcloneStorage, string) {
	t.Helper()

	// The first Encrypt and the first GetLogger load their settings from .env, which
	// they find by walking up from the working directory, so both have to run before
	// the test moves.
	config, err := encryption.GetFieldEncryptor().Encrypt("[testlocal]\ntype = local\n")
	require.NoError(t, err)

	logger.GetLogger()

	root := t.TempDir()
	t.Chdir(root)

	return &RcloneStorage{
		StorageID:     uuid.New(),
		ConfigContent: config,
	}, root
}

func Test_DeleteFile_WhenObjectExists_RemovesOnlyThatPath(t *testing.T) {
	storage, root := newLocalRcloneStorage(t)

	for _, name := range []string{"backup-1", "backup-10", "backup-1.metadata"} {
		require.NoError(t, os.WriteFile(filepath.Join(root, name), []byte("payload"), 0o600))
	}

	err := storage.DeleteFile(t.Context(), encryption.GetFieldEncryptor(), logger.GetLogger(), "backup-1")

	require.NoError(t, err)
	assert.NoFileExists(t, filepath.Join(root, "backup-1"))
	assert.FileExists(t, filepath.Join(root, "backup-10"), "a similarly named object must survive")
	assert.FileExists(t, filepath.Join(root, "backup-1.metadata"), "a sidecar is its own logical file")
}

func Test_DeleteFile_WhenObjectIsAbsent_Succeeds(t *testing.T) {
	storage, _ := newLocalRcloneStorage(t)

	err := storage.DeleteFile(t.Context(), encryption.GetFieldEncryptor(), logger.GetLogger(), "never-written")

	assert.NoError(t, err)
}

func Test_DeleteFile_WhenCalledTwice_Succeeds(t *testing.T) {
	storage, root := newLocalRcloneStorage(t)

	require.NoError(t, os.WriteFile(filepath.Join(root, "backup-1"), []byte("payload"), 0o600))

	require.NoError(t, storage.DeleteFile(
		t.Context(), encryption.GetFieldEncryptor(), logger.GetLogger(), "backup-1",
	))

	assert.NoError(t, storage.DeleteFile(
		t.Context(), encryption.GetFieldEncryptor(), logger.GetLogger(), "backup-1",
	))
}

func Test_SaveFile_WhenConfigurationIsUnchanged_BuildsRemoteOnce(t *testing.T) {
	storage, _ := newLocalRcloneStorage(t)

	saveTestFile(t, storage, "first")
	firstRemote := getCurrentCachedRemote(t, storage)

	saveTestFile(t, storage, "second")

	assert.Same(t, firstRemote, getCurrentCachedRemote(t, storage))
}

func Test_SaveFile_WhenConfigurationChanges_BuildsNewRemoteAndRetiresOld(t *testing.T) {
	storage, _ := newLocalRcloneStorage(t)

	saveTestFile(t, storage, "before-change")
	oldRemote := getCurrentCachedRemote(t, storage)

	changedConfig, err := encryption.GetFieldEncryptor().Encrypt("[testlocal]\ntype = local\nlinks = true\n")
	require.NoError(t, err)
	storage.ConfigContent = changedConfig

	saveTestFile(t, storage, "after-change")
	newRemote := getCurrentCachedRemote(t, storage)

	assert.NotSame(t, oldRemote, newRemote)
	assert.True(t, isRemoteRetired(oldRemote), "the remote built from the old configuration must be retired")
	assert.False(t, isRemoteRetired(newRemote))
}

func Test_SaveFile_WhenRemoteStaysUnusedPastIdleTimeout_DropsRemote(t *testing.T) {
	previousRemotes := remotes
	remotes = newRemoteCache(50 * time.Millisecond)
	t.Cleanup(func() { remotes = previousRemotes })

	storage, _ := newLocalRcloneStorage(t)

	saveTestFile(t, storage, "only-write")
	idleRemote := getCurrentCachedRemote(t, storage)

	require.Eventually(t, func() bool {
		remotes.mutex.Lock()
		defer remotes.mutex.Unlock()

		return remotes.remotes[storage.StorageID] == nil
	}, 5*time.Second, 10*time.Millisecond, "an idle remote must be dropped after the idle timeout")

	assert.True(t, isRemoteRetired(idleRemote))
}

func saveTestFile(t *testing.T, storage *RcloneStorage, fileName string) {
	t.Helper()

	err := storage.SaveFile(
		t.Context(), encryption.GetFieldEncryptor(), logger.GetLogger(), fileName, strings.NewReader("payload"),
	)
	require.NoError(t, err)
}

func getCurrentCachedRemote(t *testing.T, storage *RcloneStorage) *cachedRemote {
	t.Helper()

	remotes.mutex.Lock()
	defer remotes.mutex.Unlock()

	cached := remotes.remotes[storage.StorageID]
	require.NotNil(t, cached, "no remote is cached for the storage")

	return cached
}

func isRemoteRetired(cached *cachedRemote) bool {
	remotes.mutex.Lock()
	defer remotes.mutex.Unlock()

	return cached.isRetired
}
