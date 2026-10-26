package rclone

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/testwait"
	"databasus-backend/internal/modules/storages/internal/providers"
	"databasus-backend/internal/modules/storages/internal/providers/providertest"
)

func TestParseConfigContent(t *testing.T) {
	t.Parallel()

	t.Run("parses one remote", func(t *testing.T) {
		t.Parallel()

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
	})

	t.Run("parses every remote when there are several", func(t *testing.T) {
		t.Parallel()

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
	})

	t.Run("returns no sections when the content is empty", func(t *testing.T) {
		t.Parallel()

		sections, err := parseConfigContent("")

		require.NoError(t, err)
		assert.Empty(t, sections)
	})

	t.Run("ignores comments and blank lines", func(t *testing.T) {
		t.Parallel()

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
	})

	t.Run("keeps the whole value when it contains an equals sign", func(t *testing.T) {
		t.Parallel()

		content := `[myremote]
	type = s3
	secret_access_key = abc=def=ghi`

		sections, err := parseConfigContent(content)

		require.NoError(t, err)
		assert.Equal(t, "abc=def=ghi", sections["myremote"]["secret_access_key"])
	})

	t.Run("reads an empty value when a key has none", func(t *testing.T) {
		t.Parallel()

		content := `[myremote]
	type =
	provider = AWS`

		sections, err := parseConfigContent(content)

		require.NoError(t, err)
		assert.Empty(t, sections["myremote"]["type"])
		assert.Equal(t, "AWS", sections["myremote"]["provider"])
	})

	t.Run("ignores a key outside a section", func(t *testing.T) {
		t.Parallel()

		content := `orphan_key = orphan_value
	[myremote]
	type = s3`

		sections, err := parseConfigContent(content)

		require.NoError(t, err)
		assert.Len(t, sections, 1)
		assert.Equal(t, "s3", sections["myremote"]["type"])
	})

	t.Run("trims the whitespace around keys and values", func(t *testing.T) {
		t.Parallel()

		content := `[myremote]
	  type   =   s3
	  region   =   us-west-2  `

		sections, err := parseConfigContent(content)

		require.NoError(t, err)
		assert.Equal(t, "s3", sections["myremote"]["type"])
		assert.Equal(t, "us-west-2", sections["myremote"]["region"])
	})
}

// buildFs strips a leading slash from RemotePath, so an absolute root would become
// relative to the working directory. The test moves there instead and leaves the
// remote path empty, which puts the rclone root on the temporary directory.
func newLocalRcloneStorage(t *testing.T) (*configuredRemote, string) {
	t.Helper()

	fieldEncryptor := providertest.NewFieldEncryptor(t)
	root := t.TempDir()
	t.Chdir(root)

	return &configuredRemote{
		provider: New(Deps{Repositories: Repositories{settings: nil}, FieldEncryptor: fieldEncryptor}),
		settings: Settings{
			ConfigContent: providertest.Encrypt(t, fieldEncryptor, "[testlocal]\ntype = local\n"),
			RemotePath:    "",
		},
		storageID: uuid.New(),
	}, root
}

func (r *configuredRemote) access(fileName string) providers.FileAccess {
	return providers.FileAccess{
		Location: providers.Location{StorageID: r.storageID, Settings: r.settings},
		FileName: fileName,
		Logger:   providertest.NewDiscardLogger(),
	}
}

func TestDeleteFile(t *testing.T) {
	t.Run("removes only that path when the object exists", func(t *testing.T) {
		storage, root := newLocalRcloneStorage(t)

		for _, name := range []string{"backup-1", "backup-10", "backup-1.metadata"} {
			require.NoError(t, os.WriteFile(filepath.Join(root, name), []byte("payload"), 0o600))
		}

		err := storage.provider.DeleteFile(t.Context(), storage.access("backup-1"))

		require.NoError(t, err)
		assert.NoFileExists(t, filepath.Join(root, "backup-1"))
		assert.FileExists(t, filepath.Join(root, "backup-10"), "a similarly named object must survive")
		assert.FileExists(t, filepath.Join(root, "backup-1.metadata"), "a sidecar is its own logical file")
	})

	t.Run("succeeds when the object is absent", func(t *testing.T) {
		storage, _ := newLocalRcloneStorage(t)

		err := storage.provider.DeleteFile(t.Context(), storage.access("never-written"))

		assert.NoError(t, err)
	})

	t.Run("succeeds when called twice", func(t *testing.T) {
		storage, root := newLocalRcloneStorage(t)

		require.NoError(t, os.WriteFile(filepath.Join(root, "backup-1"), []byte("payload"), 0o600))

		require.NoError(t, storage.provider.DeleteFile(t.Context(), storage.access("backup-1")))

		assert.NoError(t, storage.provider.DeleteFile(t.Context(), storage.access("backup-1")))
	})
}

func TestSaveFile(t *testing.T) {
	t.Run("builds the remote once when the configuration is unchanged", func(t *testing.T) {
		storage, _ := newLocalRcloneStorage(t)

		saveTestFile(t, storage, "first")
		firstRemote := getCurrentCachedRemote(t, storage)

		saveTestFile(t, storage, "second")

		assert.Same(t, firstRemote, getCurrentCachedRemote(t, storage))
	})

	t.Run("builds a new remote and retires the old one when the configuration changes", func(t *testing.T) {
		storage, _ := newLocalRcloneStorage(t)

		saveTestFile(t, storage, "before-change")
		oldRemote := getCurrentCachedRemote(t, storage)

		storage.settings.ConfigContent = providertest.Encrypt(
			t, storage.provider.fieldEncryptor, "[testlocal]\ntype = local\nlinks = true\n",
		)

		saveTestFile(t, storage, "after-change")
		newRemote := getCurrentCachedRemote(t, storage)

		assert.NotSame(t, oldRemote, newRemote)
		assert.True(
			t,
			isRemoteRetired(storage, oldRemote),
			"the remote built from the old configuration must be retired",
		)
		assert.False(t, isRemoteRetired(storage, newRemote))
	})

	t.Run("drops the remote when it stays unused past the idle timeout", func(t *testing.T) {
		storage, _ := newLocalRcloneStorage(t)
		storage.provider.remotes = newRemoteCache(50 * time.Millisecond)

		saveTestFile(t, storage, "only-write")
		idleRemote := getCurrentCachedRemote(t, storage)

		testwait.Until(t, "the idle remote to be dropped", func(context.Context) error {
			storage.provider.remotes.mutex.Lock()
			defer storage.provider.remotes.mutex.Unlock()

			if storage.provider.remotes.remotes[storage.storageID] != nil {
				return errRemoteStillCached
			}

			return nil
		})

		assert.True(t, isRemoteRetired(storage, idleRemote))
	})
}

func saveTestFile(t *testing.T, storage *configuredRemote, fileName string) {
	t.Helper()

	err := storage.provider.SaveFile(t.Context(), storage.access(fileName), strings.NewReader("payload"))
	require.NoError(t, err)
}

func getCurrentCachedRemote(t *testing.T, storage *configuredRemote) *cachedRemote {
	t.Helper()

	storage.provider.remotes.mutex.Lock()
	defer storage.provider.remotes.mutex.Unlock()

	cached := storage.provider.remotes.remotes[storage.storageID]
	require.NotNil(t, cached, "no remote is cached for the storage")

	return cached
}

func isRemoteRetired(storage *configuredRemote, cached *cachedRemote) bool {
	storage.provider.remotes.mutex.Lock()
	defer storage.provider.remotes.mutex.Unlock()

	return cached.isRetired
}

var errRemoteStillCached = errors.New("the remote is still cached")
