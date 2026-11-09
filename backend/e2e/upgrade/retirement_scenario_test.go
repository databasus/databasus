package upgrade

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/moby/moby/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type retirementScenario struct {
	image    string
	guidance string
	settings []any
}

func runRetirementScenario(t *testing.T, scenario retirementScenario) {
	t.Helper()

	_, currentImage := getUpgradeImages(t)
	resources, err := newUpgradeResources()
	require.NoError(t, err)
	docker, err := newDockerHost()
	require.NoError(t, err)
	t.Cleanup(docker.close)
	t.Cleanup(func() {
		cleanupCtx, cancelCleanup := context.WithTimeout(context.WithoutCancel(t.Context()), cleanupTimeout)
		defer cancelCleanup()
		require.NoError(t, docker.removeResources(cleanupCtx, resources))
	})

	require.NoError(t, docker.createNetwork(t.Context(), resources.networkName))
	require.NoError(t, docker.createVolume(t.Context(), resources.volumeName))

	previousAPI := startRetirementInstance(t, docker, retirementInstanceSpec{
		resources: resources, image: scenario.image, name: resources.previousContainerName,
	})

	var account previousTokenResponse
	require.NoError(t, previousAPI.post(t.Context(), "/users", previousSignUpRequest{
		Email: accountEmail, Password: accountPassword, Name: accountName,
	}, &account))
	previousAPI.token = account.Token

	var workspace previousCreatedItem
	require.NoError(t, previousAPI.post(t.Context(), "/workspaces", previousWorkspaceRequest{
		Name: workspaceName,
	}, &workspace))

	blockingStorageIDs := make([]string, 0, len(scenario.settings))
	otherWorkspaceSnapshots := make(map[string]currentStoragePage)
	for index, settings := range scenario.settings {
		blockingWorkspaceID := workspace.ID
		if index > 0 {
			var otherWorkspace previousCreatedItem
			require.NoError(
				t,
				previousAPI.post(
					t.Context(),
					"/workspaces",
					previousWorkspaceRequest{Name: fmt.Sprintf("Retirement workspace %d", index)},
					&otherWorkspace,
				),
			)
			blockingWorkspaceID = otherWorkspace.ID
		}

		var retiredStorage previousCreatedItem
		require.NoError(t, previousAPI.post(t.Context(), "/storages", previousStorageRequest{
			WorkspaceID: blockingWorkspaceID, Name: "Retired archive", Settings: settings,
		}, &retiredStorage))
		blockingStorageIDs = append(blockingStorageIDs, retiredStorage.ID)
		if index > 0 {
			var workspaceStorages currentStoragePage
			require.NoError(
				t,
				previousAPI.get(t.Context(), "/storages?workspaceId="+blockingWorkspaceID, &workspaceStorages),
			)
			otherWorkspaceSnapshots[blockingWorkspaceID] = workspaceStorages
		}
	}

	var supportedStorage previousCreatedItem
	require.NoError(t, previousAPI.post(t.Context(), "/storages", newWrittenStorages(workspace.ID)[1].request,
		&supportedStorage))

	var originalStorages currentStoragePage
	require.NoError(t, previousAPI.get(t.Context(), "/storages?workspaceId="+workspace.ID, &originalStorages))
	require.Len(t, originalStorages.Items, 2)

	originalSupportedSecrets := getSupportedRetirementSecrets(t, docker, resources.previousContainerName)

	require.NoError(t, docker.stopContainer(t.Context(), resources.previousContainerName))
	removeRetirementContainer(t, docker, resources.previousContainerName)

	originalSnapshot := getStoppedRetirementSnapshot(
		t,
		docker,
		retirementSnapshotSpec{resources: resources, image: scenario.image},
	)
	hasPlaintextCredentials := strings.Contains(originalSnapshot, retirementClientSecret) ||
		strings.Contains(
			originalSnapshot,
			retirementRefreshToken,
		) || strings.Contains(originalSnapshot, retirementRcloneSecret)
	assert.False(t, hasPlaintextCredentials)

	_, err = startInstance(t, docker, containerSpec{
		name: resources.currentContainerName, image: currentImage, entrypoint: nil, env: nil,
		volumeMount: resources.volumeName + ":" + instanceDataPath,
		networkName: resources.networkName, networkAlias: resources.currentContainerName,
		publishedPort: instancePublishedPort,
	})

	require.ErrorIs(t, err, errInstanceExited)
	assert.Contains(t, err.Error(), scenario.guidance)
	assert.NotContains(t, err.Error(), retirementClientSecret)
	assert.NotContains(t, err.Error(), retirementRefreshToken)
	assert.NotContains(t, err.Error(), retirementRcloneSecret)

	inspection, err := docker.client.ContainerInspect(
		t.Context(),
		resources.currentContainerName,
		client.ContainerInspectOptions{},
	)
	require.NoError(t, err)
	require.NotNil(t, inspection.Container.State)
	assert.NotZero(t, inspection.Container.State.ExitCode)

	removeRetirementContainer(t, docker, resources.currentContainerName)

	hasUnchangedSnapshot := originalSnapshot == getStoppedRetirementSnapshot(
		t,
		docker,
		retirementSnapshotSpec{resources: resources, image: scenario.image},
	)
	require.True(t, hasUnchangedSnapshot, "refusal must preserve all public records and migration history")

	previousAPI = startRetirementInstance(t, docker, retirementInstanceSpec{
		resources: resources, image: scenario.image, name: resources.previousContainerName,
	})
	signInRetirementAccount(t, previousAPI)

	var recoveredStorages currentStoragePage
	require.NoError(t, previousAPI.get(t.Context(), "/storages?workspaceId="+workspace.ID, &recoveredStorages))
	assert.Equal(t, originalStorages, recoveredStorages)

	for workspaceID, originalWorkspaceStorages := range otherWorkspaceSnapshots {
		var recoveredWorkspaceStorages currentStoragePage
		require.NoError(
			t,
			previousAPI.get(t.Context(), "/storages?workspaceId="+workspaceID, &recoveredWorkspaceStorages),
		)
		assert.Equal(t, originalWorkspaceStorages, recoveredWorkspaceStorages)
	}

	for _, storageID := range blockingStorageIDs {
		require.NoError(t, previousAPI.send(t.Context(), http.MethodDelete, "/storages/"+storageID, nil, nil))
	}

	require.NoError(t, docker.stopContainer(t.Context(), resources.previousContainerName))
	removeRetirementContainer(t, docker, resources.previousContainerName)

	currentAPI := startRetirementInstance(t, docker, retirementInstanceSpec{
		resources: resources, image: currentImage, name: resources.currentContainerName,
	})
	signInRetirementAccount(t, currentAPI)

	var upgradedStorages currentStoragePage
	require.NoError(t, currentAPI.get(t.Context(), "/storages?workspaceId="+workspace.ID, &upgradedStorages))
	require.Len(t, upgradedStorages.Items, 1)
	assert.Equal(t, supportedStorage.ID, upgradedStorages.Items[0].ID)
	for _, originalStorage := range originalStorages.Items {
		if originalStorage.ID == supportedStorage.ID {
			assert.Equal(t, originalStorage, upgradedStorages.Items[0])
		}
	}

	schemaState, err := docker.executeRetirementCommand(
		t.Context(),
		resources.currentContainerName,
		[]string{
			"sh",
			"-c",
			`psql "$(cat /dev/shm/databasus-database-dsn)" -AtXq -v ON_ERROR_STOP=1 -c "BEGIN READ ONLY; SELECT to_regclass('storages_google_drive') IS NULL AND to_regclass('google_drive_storages') IS NULL AND to_regclass('storages_rclone') IS NULL AND to_regclass('rclone_storages') IS NULL; ROLLBACK;"`,
		},
	)
	require.NoError(t, err)
	assert.Equal(t, "t", strings.TrimSpace(schemaState))

	hasUnchangedSupportedSecrets := originalSupportedSecrets ==
		getSupportedRetirementSecrets(t, docker, resources.currentContainerName)
	assert.True(t, hasUnchangedSupportedSecrets, "successful upgrade must preserve supported encrypted credentials")
}

type retirementInstanceSpec struct {
	resources upgradeResources
	image     string
	name      string
}

func startRetirementInstance(t *testing.T, docker *dockerHost, spec retirementInstanceSpec) *apiClient {
	t.Helper()

	api, err := startInstance(t, docker, containerSpec{
		name: spec.name, image: spec.image, entrypoint: nil,
		env:         []string{"IS_DISABLE_ANONYMOUS_TELEMETRY=true"},
		volumeMount: spec.resources.volumeName + ":" + instanceDataPath,
		networkName: spec.resources.networkName, networkAlias: spec.name, publishedPort: instancePublishedPort,
	})
	require.NoError(t, err)

	return api
}

func signInRetirementAccount(t *testing.T, api *apiClient) {
	t.Helper()

	var signIn currentSignInResponse
	require.NoError(t, api.post(t.Context(), "/sessions", currentSignInRequest{
		Email: accountEmail, Password: accountPassword,
	}, &signIn))
	require.NotEmpty(t, signIn.Token)
	api.token = signIn.Token
}
