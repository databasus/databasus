package verification_config

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/intervals"
	users_enums "databasus-backend/internal/features/users/enums"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_controllers "databasus-backend/internal/features/workspaces/controllers"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/modules/users"
	test_utils "databasus-backend/internal/util/testing"
)

func createTestRouter() *gin.Engine {
	return workspaces_testing.CreateTestRouter(
		workspaces_controllers.GetWorkspaceController(),
		workspaces_controllers.GetMembershipController(),
		databases.GetDatabaseController(),
		GetVerificationConfigController(),
	)
}

func newEnabledRequest() SaveBackupVerificationConfigDTO {
	return SaveBackupVerificationConfigDTO{
		IsScheduledVerificationEnabled: true,
		ScheduleType:                   VerificationScheduleInterval,
		VerificationInterval: intervals.Interval{
			Type:      intervals.IntervalWeekly,
			TimeOfDay: new("04:00"),
			Weekday:   new(0),
		},
		SendNotificationsOn: []VerificationNotificationType{
			NotificationVerificationFailed,
		},
	}
}

func createTestDatabaseViaAPI(
	name string,
	workspaceID uuid.UUID,
	token string,
	router *gin.Engine,
) *databases.Database {
	request := databases.Database{
		WorkspaceID:       &workspaceID,
		Name:              name,
		Type:              databases.DatabaseTypePostgresLogical,
		PostgresqlLogical: databases.GetTestPostgresConfig(),
	}

	w := workspaces_testing.MakeAPIRequest(
		router,
		"POST",
		"/api/v1/databases/create",
		"Bearer "+token,
		request,
	)

	if w.Code != http.StatusCreated {
		panic("Failed to create database")
	}

	var database databases.Database
	if err := json.Unmarshal(w.Body.Bytes(), &database); err != nil {
		panic(err)
	}

	return &database
}

func Test_GetByDatabaseID_WhenNoRowExists_LazyCreatesDisabledDefault(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
	database := createTestDatabaseViaAPI("Test Database", workspace.ID, owner.Token, router)

	var response BackupVerificationConfig
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/verification-config/"+database.ID.String(),
		"Bearer "+owner.Token,
		http.StatusOK,
		&response,
	)

	assert.Equal(t, database.ID, response.DatabaseID)
	assert.False(t, response.IsScheduledVerificationEnabled)
	assert.Equal(t, VerificationScheduleAfterBackup, response.ScheduleType)
	assert.Equal(t, intervals.IntervalWeekly, response.VerificationInterval.Type)
	assert.NotNil(t, response.VerificationInterval.TimeOfDay)
	assert.Equal(t, "04:00", *response.VerificationInterval.TimeOfDay)
	assert.NotNil(t, response.VerificationInterval.Weekday)
	assert.Equal(t, 0, *response.VerificationInterval.Weekday)
	assert.Equal(
		t,
		[]VerificationNotificationType{NotificationVerificationFailed},
		response.SendNotificationsOn,
	)

	databases.RemoveTestDatabase(t.Context(), database)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
}

func Test_GetByDatabaseID_WhenUserNotInWorkspace_ReturnsError(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
	database := createTestDatabaseViaAPI("Test Database", workspace.ID, owner.Token, router)

	nonMember := users_testing.CreateTestUser(t.Context(), users.RoleMember)

	resp := test_utils.MakeGetRequest(
		t,
		router,
		"/api/v1/verification-config/"+database.ID.String(),
		"Bearer "+nonMember.Token,
		http.StatusBadRequest,
	)
	assert.Contains(t, string(resp.Body), "insufficient permissions")

	databases.RemoveTestDatabase(t.Context(), database)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
}

func Test_Save_AsOwner_PersistsAndReturns(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
	database := createTestDatabaseViaAPI("Test Database", workspace.ID, owner.Token, router)

	request := newEnabledRequest()
	request.SendNotificationsOn = []VerificationNotificationType{
		NotificationVerificationSuccess,
		NotificationVerificationFailed,
	}

	var response BackupVerificationConfig
	test_utils.MakePutRequestAndUnmarshal(
		t,
		router,
		"/api/v1/verification-config/"+database.ID.String(),
		"Bearer "+owner.Token,
		request,
		http.StatusOK,
		&response,
	)

	assert.Equal(t, database.ID, response.DatabaseID)
	assert.True(t, response.IsScheduledVerificationEnabled)
	assert.Equal(t, intervals.IntervalWeekly, response.VerificationInterval.Type)
	assert.ElementsMatch(t,
		[]VerificationNotificationType{
			NotificationVerificationSuccess,
			NotificationVerificationFailed,
		},
		response.SendNotificationsOn,
	)

	var fetched BackupVerificationConfig
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/verification-config/"+database.ID.String(),
		"Bearer "+owner.Token,
		http.StatusOK,
		&fetched,
	)
	assert.True(t, fetched.IsScheduledVerificationEnabled)
	assert.ElementsMatch(t,
		[]VerificationNotificationType{
			NotificationVerificationSuccess,
			NotificationVerificationFailed,
		},
		fetched.SendNotificationsOn,
	)

	databases.RemoveTestDatabase(t.Context(), database)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
}

func Test_Save_AsViewer_RejectedWithPermissionError(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
	database := createTestDatabaseViaAPI("Test Database", workspace.ID, owner.Token, router)

	viewer := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspaces_testing.AddMemberToWorkspace(
		workspace,
		viewer,
		users_enums.WorkspaceRoleViewer,
		owner.Token,
		router,
	)

	resp := test_utils.MakePutRequest(
		t,
		router,
		"/api/v1/verification-config/"+database.ID.String(),
		"Bearer "+viewer.Token,
		newEnabledRequest(),
		http.StatusBadRequest,
	)
	assert.Contains(t, string(resp.Body), "insufficient permissions")

	databases.RemoveTestDatabase(t.Context(), database)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
}

func Test_Save_WhenEnabled_WithInvalidInterval_ReturnsValidationError(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
	database := createTestDatabaseViaAPI("Test Database", workspace.ID, owner.Token, router)

	request := newEnabledRequest()
	request.VerificationInterval = intervals.Interval{
		Type: intervals.IntervalCron,
	}

	resp := test_utils.MakePutRequest(
		t,
		router,
		"/api/v1/verification-config/"+database.ID.String(),
		"Bearer "+owner.Token,
		request,
		http.StatusBadRequest,
	)
	assert.Contains(t, string(resp.Body), "cron expression")

	databases.RemoveTestDatabase(t.Context(), database)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
}

func Test_Save_WithInvalidNotificationType_ReturnsValidationError(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
	database := createTestDatabaseViaAPI("Test Database", workspace.ID, owner.Token, router)

	request := newEnabledRequest()
	request.SendNotificationsOn = []VerificationNotificationType{
		VerificationNotificationType("VERIFICATION_EXPIRED"),
	}

	resp := test_utils.MakePutRequest(
		t,
		router,
		"/api/v1/verification-config/"+database.ID.String(),
		"Bearer "+owner.Token,
		request,
		http.StatusBadRequest,
	)
	assert.Contains(t, string(resp.Body), "invalid verification notification type")

	databases.RemoveTestDatabase(t.Context(), database)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
}

func Test_Save_WhenScheduleTypeAfterBackup_SkipsIntervalValidationAndPersists(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
	database := createTestDatabaseViaAPI("Test Database", workspace.ID, owner.Token, router)

	request := SaveBackupVerificationConfigDTO{
		IsScheduledVerificationEnabled: true,
		ScheduleType:                   VerificationScheduleAfterBackup,
		// Deliberately invalid as a time interval — must be ignored in after-backup mode.
		VerificationInterval: intervals.Interval{Type: intervals.IntervalCron},
		SendNotificationsOn:  []VerificationNotificationType{NotificationVerificationFailed},
	}

	var response BackupVerificationConfig
	test_utils.MakePutRequestAndUnmarshal(
		t,
		router,
		"/api/v1/verification-config/"+database.ID.String(),
		"Bearer "+owner.Token,
		request,
		http.StatusOK,
		&response,
	)

	assert.True(t, response.IsScheduledVerificationEnabled)
	assert.Equal(t, VerificationScheduleAfterBackup, response.ScheduleType)

	var fetched BackupVerificationConfig
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/verification-config/"+database.ID.String(),
		"Bearer "+owner.Token,
		http.StatusOK,
		&fetched,
	)
	assert.Equal(t, VerificationScheduleAfterBackup, fetched.ScheduleType)

	databases.RemoveTestDatabase(t.Context(), database)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
}

func Test_Save_WhenScheduleTypeInvalid_ReturnsValidationError(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
	database := createTestDatabaseViaAPI("Test Database", workspace.ID, owner.Token, router)

	request := newEnabledRequest()
	request.ScheduleType = VerificationScheduleType("WHENEVER")

	resp := test_utils.MakePutRequest(
		t,
		router,
		"/api/v1/verification-config/"+database.ID.String(),
		"Bearer "+owner.Token,
		request,
		http.StatusBadRequest,
	)
	assert.Contains(t, string(resp.Body), "invalid verification schedule type")

	databases.RemoveTestDatabase(t.Context(), database)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
}

func Test_OnDatabaseCopied_CopiesConfigOntoNewDatabase(t *testing.T) {
	router := createTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
	source := createTestDatabaseViaAPI("Source Database", workspace.ID, owner.Token, router)
	target := createTestDatabaseViaAPI("Target Database", workspace.ID, owner.Token, router)

	saveRequest := newEnabledRequest()
	saveRequest.SendNotificationsOn = []VerificationNotificationType{
		NotificationVerificationSuccess,
		NotificationVerificationFailed,
	}

	var saved BackupVerificationConfig
	test_utils.MakePutRequestAndUnmarshal(
		t,
		router,
		"/api/v1/verification-config/"+source.ID.String(),
		"Bearer "+owner.Token,
		saveRequest,
		http.StatusOK,
		&saved,
	)

	getVerificationConfigService().OnDatabaseCopied(t.Context(), source.ID, target.ID)

	copied, err := getVerificationConfigRepository().GetByDatabaseID(target.ID)
	assert.NoError(t, err)
	assert.NotNil(t, copied)
	assert.Equal(t, target.ID, copied.DatabaseID)
	assert.True(t, copied.IsScheduledVerificationEnabled)
	assert.Equal(t, intervals.IntervalWeekly, copied.VerificationInterval.Type)
	assert.ElementsMatch(t,
		[]VerificationNotificationType{
			NotificationVerificationSuccess,
			NotificationVerificationFailed,
		},
		copied.SendNotificationsOn,
	)

	databases.RemoveTestDatabase(t.Context(), source)
	databases.RemoveTestDatabase(t.Context(), target)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
}
