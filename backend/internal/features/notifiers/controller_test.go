package notifiers

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	audit_logs "databasus-backend/internal/features/audit_logs"
	discord_notifier "databasus-backend/internal/features/notifiers/models/discord"
	email_notifier "databasus-backend/internal/features/notifiers/models/email_notifier"
	mattermost_notifier "databasus-backend/internal/features/notifiers/models/mattermost"
	slack_notifier "databasus-backend/internal/features/notifiers/models/slack"
	teams_notifier "databasus-backend/internal/features/notifiers/models/teams"
	telegram_notifier "databasus-backend/internal/features/notifiers/models/telegram"
	webhook_notifier "databasus-backend/internal/features/notifiers/models/webhook"
	users_enums "databasus-backend/internal/features/users/enums"
	users_middleware "databasus-backend/internal/features/users/middleware"
	users_services "databasus-backend/internal/features/users/services"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_controllers "databasus-backend/internal/features/workspaces/controllers"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/lib/mailer"
	"databasus-backend/internal/storage"
	test_utils "databasus-backend/internal/util/testing"
	"databasus-backend/internal/util/testing/containers"
)

func Test_SaveNewNotifier_NotifierReturnedViaGet(t *testing.T) {
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	router := createRouter()
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

	notifier := createNewNotifier(workspace.ID)

	var savedNotifier Notifier
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/notifiers",
		"Bearer "+owner.Token,
		*notifier,
		http.StatusOK,
		&savedNotifier,
	)

	verifyNotifierData(t, notifier, &savedNotifier)
	assert.NotEmpty(t, savedNotifier.ID)

	// Verify notifier is returned via GET
	var retrievedNotifier Notifier
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		fmt.Sprintf("/api/v1/notifiers/%s", savedNotifier.ID.String()),
		"Bearer "+owner.Token,
		http.StatusOK,
		&retrievedNotifier,
	)

	verifyNotifierData(t, &savedNotifier, &retrievedNotifier)

	// Verify notifier is returned via GET all notifiers
	var notifiers []Notifier
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		fmt.Sprintf("/api/v1/notifiers?workspace_id=%s", workspace.ID.String()),
		"Bearer "+owner.Token,
		http.StatusOK,
		&notifiers,
	)

	assert.Len(t, notifiers, 1)

	deleteNotifier(t, router, savedNotifier.ID, workspace.ID, owner.Token)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
}

func Test_UpdateExistingNotifier_UpdatedNotifierReturnedViaGet(t *testing.T) {
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	router := createRouter()
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

	notifier := createNewNotifier(workspace.ID)

	var savedNotifier Notifier
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/notifiers",
		"Bearer "+owner.Token,
		*notifier,
		http.StatusOK,
		&savedNotifier,
	)

	updatedName := "Updated Notifier " + uuid.New().String()
	savedNotifier.Name = updatedName

	var updatedNotifier Notifier
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/notifiers",
		"Bearer "+owner.Token,
		savedNotifier,
		http.StatusOK,
		&updatedNotifier,
	)

	assert.Equal(t, updatedName, updatedNotifier.Name)
	assert.Equal(t, savedNotifier.ID, updatedNotifier.ID)

	deleteNotifier(t, router, updatedNotifier.ID, workspace.ID, owner.Token)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
}

func Test_DeleteNotifier_NotifierNotReturnedViaGet(t *testing.T) {
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	router := createRouter()
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

	notifier := createNewNotifier(workspace.ID)

	var savedNotifier Notifier
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/notifiers",
		"Bearer "+owner.Token,
		*notifier,
		http.StatusOK,
		&savedNotifier,
	)

	test_utils.MakeDeleteRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/notifiers/%s", savedNotifier.ID.String()),
		"Bearer "+owner.Token,
		http.StatusOK,
	)

	response := test_utils.MakeGetRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/notifiers/%s", savedNotifier.ID.String()),
		"Bearer "+owner.Token,
		http.StatusBadRequest,
	)

	assert.Contains(t, string(response.Body), "error")
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
}

func Test_SendTestNotificationDirect_NotificationSent(t *testing.T) {
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	router := createRouter()
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

	GetWebhookStub().ResetCalls()
	notifier := createWebhookNotifier(workspace.ID)

	response := test_utils.MakePostRequest(
		t, router, "/api/v1/notifiers/direct-test", "Bearer "+owner.Token, *notifier, http.StatusOK,
	)

	assert.Contains(t, string(response.Body), "successful")
	assert.Greater(t, GetWebhookStub().CallCount(), 0)

	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
}

func Test_SendTestNotificationExisting_NotificationSent(t *testing.T) {
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	router := createRouter()
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

	GetWebhookStub().ResetCalls()
	notifier := createWebhookNotifier(workspace.ID)

	var savedNotifier Notifier
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/notifiers",
		"Bearer "+owner.Token,
		*notifier,
		http.StatusOK,
		&savedNotifier,
	)

	response := test_utils.MakePostRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/notifiers/%s/test", savedNotifier.ID.String()),
		"Bearer "+owner.Token,
		nil,
		http.StatusOK,
	)

	assert.Contains(t, string(response.Body), "successful")
	assert.Greater(t, GetWebhookStub().CallCount(), 0)

	deleteNotifier(t, router, savedNotifier.ID, workspace.ID, owner.Token)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
}

func Test_WorkspaceRolePermissions_Notifiers(t *testing.T) {
	tests := []struct {
		name          string
		workspaceRole *users_enums.WorkspaceRole
		isGlobalAdmin bool
		canCreate     bool
		canUpdate     bool
		canDelete     bool
	}{
		{
			name:          "owner can manage notifiers",
			workspaceRole: func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleOwner; return &r }(),
			isGlobalAdmin: false,
			canCreate:     true,
			canUpdate:     true,
			canDelete:     true,
		},
		{
			name:          "admin can manage notifiers",
			workspaceRole: func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleAdmin; return &r }(),
			isGlobalAdmin: false,
			canCreate:     true,
			canUpdate:     true,
			canDelete:     true,
		},
		{
			name:          "member can manage notifiers",
			workspaceRole: func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleMember; return &r }(),
			isGlobalAdmin: false,
			canCreate:     true,
			canUpdate:     true,
			canDelete:     true,
		},
		{
			name:          "viewer can view but cannot modify notifiers",
			workspaceRole: func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleViewer; return &r }(),
			isGlobalAdmin: false,
			canCreate:     false,
			canUpdate:     false,
			canDelete:     false,
		},
		{
			name:          "global admin can manage notifiers",
			workspaceRole: nil,
			isGlobalAdmin: true,
			canCreate:     true,
			canUpdate:     true,
			canDelete:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := createRouter()
			GetNotifierService().SetNotifierDatabaseCounter(&mockNotifierDatabaseCounter{})

			owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
			workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

			var testUserToken string
			if tt.isGlobalAdmin {
				admin := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleAdmin)
				testUserToken = admin.Token
			} else if tt.workspaceRole != nil && *tt.workspaceRole == users_enums.WorkspaceRoleOwner {
				testUserToken = owner.Token
			} else if tt.workspaceRole != nil {
				testUser := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
				workspaces_testing.AddMemberToWorkspace(
					workspace,
					testUser,
					*tt.workspaceRole,
					owner.Token,
					router,
				)
				testUserToken = testUser.Token
			}

			// Owner creates initial notifier for all test cases
			var ownerNotifier Notifier
			notifier := createNewNotifier(workspace.ID)
			test_utils.MakePostRequestAndUnmarshal(
				t, router, "/api/v1/notifiers", "Bearer "+owner.Token,
				*notifier, http.StatusOK, &ownerNotifier,
			)

			// Test GET notifiers
			var notifiers []Notifier
			test_utils.MakeGetRequestAndUnmarshal(
				t, router,
				fmt.Sprintf("/api/v1/notifiers?workspace_id=%s", workspace.ID.String()),
				"Bearer "+testUserToken, http.StatusOK, &notifiers,
			)
			assert.Len(t, notifiers, 1)

			// Test CREATE notifier
			createStatusCode := http.StatusOK
			if !tt.canCreate {
				createStatusCode = http.StatusForbidden
			}
			newNotifier := createNewNotifier(workspace.ID)
			var savedNotifier Notifier
			if tt.canCreate {
				test_utils.MakePostRequestAndUnmarshal(
					t, router, "/api/v1/notifiers", "Bearer "+testUserToken,
					*newNotifier, createStatusCode, &savedNotifier,
				)
				assert.NotEmpty(t, savedNotifier.ID)
			} else {
				test_utils.MakePostRequest(
					t, router, "/api/v1/notifiers", "Bearer "+testUserToken,
					*newNotifier, createStatusCode,
				)
			}

			// Test UPDATE notifier
			updateStatusCode := http.StatusOK
			if !tt.canUpdate {
				updateStatusCode = http.StatusForbidden
			}
			ownerNotifier.Name = "Updated by test user"
			if tt.canUpdate {
				var updatedNotifier Notifier
				test_utils.MakePostRequestAndUnmarshal(
					t, router, "/api/v1/notifiers", "Bearer "+testUserToken,
					ownerNotifier, updateStatusCode, &updatedNotifier,
				)
				assert.Equal(t, "Updated by test user", updatedNotifier.Name)
			} else {
				test_utils.MakePostRequest(
					t, router, "/api/v1/notifiers", "Bearer "+testUserToken,
					ownerNotifier, updateStatusCode,
				)
			}

			// Test DELETE notifier
			deleteStatusCode := http.StatusOK
			if !tt.canDelete {
				deleteStatusCode = http.StatusForbidden
			}
			test_utils.MakeDeleteRequest(
				t, router,
				fmt.Sprintf("/api/v1/notifiers/%s", ownerNotifier.ID.String()),
				"Bearer "+testUserToken, deleteStatusCode,
			)

			// Cleanup
			if tt.canCreate {
				deleteNotifier(t, router, savedNotifier.ID, workspace.ID, owner.Token)
			}
			if !tt.canDelete {
				deleteNotifier(t, router, ownerNotifier.ID, workspace.ID, owner.Token)
			}
			workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
		})
	}
}

func Test_UserNotInWorkspace_CannotAccessNotifiers(t *testing.T) {
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	outsider := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	router := createRouter()
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

	notifier := createNewNotifier(workspace.ID)

	var savedNotifier Notifier
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/notifiers",
		"Bearer "+owner.Token,
		*notifier,
		http.StatusOK,
		&savedNotifier,
	)

	// Outsider cannot GET notifiers
	test_utils.MakeGetRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/notifiers?workspace_id=%s", workspace.ID.String()),
		"Bearer "+outsider.Token,
		http.StatusForbidden,
	)

	// Outsider cannot CREATE notifier
	test_utils.MakePostRequest(
		t, router, "/api/v1/notifiers", "Bearer "+outsider.Token, *notifier, http.StatusForbidden,
	)

	// Outsider cannot UPDATE notifier
	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/notifiers",
		"Bearer "+outsider.Token,
		savedNotifier,
		http.StatusForbidden,
	)

	// Outsider cannot DELETE notifier
	test_utils.MakeDeleteRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/notifiers/%s", savedNotifier.ID.String()),
		"Bearer "+outsider.Token,
		http.StatusForbidden,
	)

	deleteNotifier(t, router, savedNotifier.ID, workspace.ID, owner.Token)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
}

func Test_CrossWorkspaceSecurity_CannotAccessNotifierFromAnotherWorkspace(t *testing.T) {
	owner1 := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	owner2 := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	router := createRouter()
	workspace1 := workspaces_testing.CreateTestWorkspace(t.Context(), "Workspace 1", owner1, router)
	workspace2 := workspaces_testing.CreateTestWorkspace(t.Context(), "Workspace 2", owner2, router)

	notifier1 := createNewNotifier(workspace1.ID)

	var savedNotifier Notifier
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/notifiers",
		"Bearer "+owner1.Token,
		*notifier1,
		http.StatusOK,
		&savedNotifier,
	)

	// Try to access workspace1's notifier with owner2 from workspace2
	response := test_utils.MakeGetRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/notifiers/%s", savedNotifier.ID.String()),
		"Bearer "+owner2.Token,
		http.StatusForbidden,
	)
	assert.Contains(t, string(response.Body), "insufficient permissions")

	deleteNotifier(t, router, savedNotifier.ID, workspace1.ID, owner1.Token)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace1, router)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace2, router)
}

func Test_NotifierSensitiveDataLifecycle_AllTypes(t *testing.T) {
	testCases := []struct {
		name                string
		notifierType        NotifierType
		createNotifier      func(workspaceID uuid.UUID) *Notifier
		updateNotifier      func(workspaceID, notifierID uuid.UUID) *Notifier
		verifySensitiveData func(t *testing.T, notifier *Notifier)
		verifyHiddenData    func(t *testing.T, notifier *Notifier)
	}{
		{
			name:         "Telegram Notifier",
			notifierType: NotifierTypeTelegram,
			createNotifier: func(workspaceID uuid.UUID) *Notifier {
				return &Notifier{
					WorkspaceID:  workspaceID,
					Name:         "Test Telegram Notifier",
					NotifierType: NotifierTypeTelegram,
					TelegramNotifier: &telegram_notifier.TelegramNotifier{
						BotToken:       "original-bot-token-12345",
						TargetChatID:   "123456789",
						IsProxyEnabled: true,
						ProxyURL:       "socks5://user:password@proxy.example.com:1080",
					},
				}
			},
			updateNotifier: func(workspaceID, notifierID uuid.UUID) *Notifier {
				return &Notifier{
					ID:           notifierID,
					WorkspaceID:  workspaceID,
					Name:         "Updated Telegram Notifier",
					NotifierType: NotifierTypeTelegram,
					TelegramNotifier: &telegram_notifier.TelegramNotifier{
						BotToken:       "",
						TargetChatID:   "987654321",
						IsProxyEnabled: true,
						ProxyURL:       "",
					},
				}
			},
			verifySensitiveData: func(t *testing.T, notifier *Notifier) {
				assert.True(
					t,
					isEncrypted(notifier.TelegramNotifier.BotToken),
					"BotToken should be encrypted in DB",
				)
				decrypted := decryptField(t, notifier.TelegramNotifier.BotToken)
				assert.Equal(t, "original-bot-token-12345", decrypted)

				assert.True(
					t,
					isEncrypted(notifier.TelegramNotifier.ProxyURL),
					"ProxyURL should be encrypted in DB",
				)
				decryptedProxyURL := decryptField(t, notifier.TelegramNotifier.ProxyURL)
				assert.Equal(t, "socks5://user:password@proxy.example.com:1080", decryptedProxyURL)
				assert.True(t, notifier.TelegramNotifier.IsProxyEnabled)
			},
			verifyHiddenData: func(t *testing.T, notifier *Notifier) {
				assert.Equal(t, "", notifier.TelegramNotifier.BotToken)
				assert.Equal(t, "", notifier.TelegramNotifier.ProxyURL)
				assert.True(t, notifier.TelegramNotifier.IsProxyEnabled)
			},
		},
		{
			name:         "Email Notifier",
			notifierType: NotifierTypeEmail,
			createNotifier: func(workspaceID uuid.UUID) *Notifier {
				return &Notifier{
					WorkspaceID:  workspaceID,
					Name:         "Test Email Notifier",
					NotifierType: NotifierTypeEmail,
					EmailNotifier: &email_notifier.EmailNotifier{
						TargetEmail:  "test@example.com",
						SMTPHost:     "smtp.example.com",
						SMTPPort:     587,
						SMTPUser:     "user@example.com",
						SMTPPassword: "original-password-secret",
					},
				}
			},
			updateNotifier: func(workspaceID, notifierID uuid.UUID) *Notifier {
				return &Notifier{
					ID:           notifierID,
					WorkspaceID:  workspaceID,
					Name:         "Updated Email Notifier",
					NotifierType: NotifierTypeEmail,
					EmailNotifier: &email_notifier.EmailNotifier{
						TargetEmail:  "updated@example.com",
						SMTPHost:     "smtp.newhost.com",
						SMTPPort:     465,
						SMTPUser:     "newuser@example.com",
						SMTPPassword: "",
					},
				}
			},
			verifySensitiveData: func(t *testing.T, notifier *Notifier) {
				assert.True(
					t,
					isEncrypted(notifier.EmailNotifier.SMTPPassword),
					"SMTPPassword should be encrypted in DB",
				)
				decrypted := decryptField(t, notifier.EmailNotifier.SMTPPassword)
				assert.Equal(t, "original-password-secret", decrypted)
			},
			verifyHiddenData: func(t *testing.T, notifier *Notifier) {
				assert.Equal(t, "", notifier.EmailNotifier.SMTPPassword)
			},
		},
		{
			name:         "Slack Notifier",
			notifierType: NotifierTypeSlack,
			createNotifier: func(workspaceID uuid.UUID) *Notifier {
				return &Notifier{
					WorkspaceID:  workspaceID,
					Name:         "Test Slack Notifier",
					NotifierType: NotifierTypeSlack,
					SlackNotifier: &slack_notifier.SlackNotifier{
						BotToken:     "xoxb-original-slack-token",
						TargetChatID: "C123456",
					},
				}
			},
			updateNotifier: func(workspaceID, notifierID uuid.UUID) *Notifier {
				return &Notifier{
					ID:           notifierID,
					WorkspaceID:  workspaceID,
					Name:         "Updated Slack Notifier",
					NotifierType: NotifierTypeSlack,
					SlackNotifier: &slack_notifier.SlackNotifier{
						BotToken:     "",
						TargetChatID: "C789012",
					},
				}
			},
			verifySensitiveData: func(t *testing.T, notifier *Notifier) {
				assert.True(
					t,
					isEncrypted(notifier.SlackNotifier.BotToken),
					"BotToken should be encrypted in DB",
				)
				decrypted := decryptField(t, notifier.SlackNotifier.BotToken)
				assert.Equal(t, "xoxb-original-slack-token", decrypted)
			},
			verifyHiddenData: func(t *testing.T, notifier *Notifier) {
				assert.Equal(t, "", notifier.SlackNotifier.BotToken)
			},
		},
		{
			name:         "Discord Notifier",
			notifierType: NotifierTypeDiscord,
			createNotifier: func(workspaceID uuid.UUID) *Notifier {
				return &Notifier{
					WorkspaceID:  workspaceID,
					Name:         "Test Discord Notifier",
					NotifierType: NotifierTypeDiscord,
					DiscordNotifier: &discord_notifier.DiscordNotifier{
						ChannelWebhookURL: "https://discord.com/api/webhooks/123/original-token",
					},
				}
			},
			updateNotifier: func(workspaceID, notifierID uuid.UUID) *Notifier {
				return &Notifier{
					ID:           notifierID,
					WorkspaceID:  workspaceID,
					Name:         "Updated Discord Notifier",
					NotifierType: NotifierTypeDiscord,
					DiscordNotifier: &discord_notifier.DiscordNotifier{
						ChannelWebhookURL: "",
					},
				}
			},
			verifySensitiveData: func(t *testing.T, notifier *Notifier) {
				assert.True(
					t,
					isEncrypted(notifier.DiscordNotifier.ChannelWebhookURL),
					"WebhookURL should be encrypted in DB",
				)
				decrypted := decryptField(
					t,
					notifier.DiscordNotifier.ChannelWebhookURL,
				)
				assert.Equal(t, "https://discord.com/api/webhooks/123/original-token", decrypted)
			},
			verifyHiddenData: func(t *testing.T, notifier *Notifier) {
				assert.Equal(t, "", notifier.DiscordNotifier.ChannelWebhookURL)
			},
		},
		{
			name:         "Teams Notifier",
			notifierType: NotifierTypeTeams,
			createNotifier: func(workspaceID uuid.UUID) *Notifier {
				return &Notifier{
					WorkspaceID:  workspaceID,
					Name:         "Test Teams Notifier",
					NotifierType: NotifierTypeTeams,
					TeamsNotifier: &teams_notifier.TeamsNotifier{
						WebhookURL: "https://outlook.office.com/webhook/original-token",
					},
				}
			},
			updateNotifier: func(workspaceID, notifierID uuid.UUID) *Notifier {
				return &Notifier{
					ID:           notifierID,
					WorkspaceID:  workspaceID,
					Name:         "Updated Teams Notifier",
					NotifierType: NotifierTypeTeams,
					TeamsNotifier: &teams_notifier.TeamsNotifier{
						WebhookURL: "",
					},
				}
			},
			verifySensitiveData: func(t *testing.T, notifier *Notifier) {
				assert.True(
					t,
					isEncrypted(notifier.TeamsNotifier.WebhookURL),
					"WebhookURL should be encrypted in DB",
				)
				decrypted := decryptField(t, notifier.TeamsNotifier.WebhookURL)
				assert.Equal(
					t,
					"https://outlook.office.com/webhook/original-token",
					decrypted,
				)
			},
			verifyHiddenData: func(t *testing.T, notifier *Notifier) {
				assert.Equal(t, "", notifier.TeamsNotifier.WebhookURL)
			},
		},
		{
			name:         "Mattermost Notifier - incoming webhook mode",
			notifierType: NotifierTypeMattermost,
			createNotifier: func(workspaceID uuid.UUID) *Notifier {
				return &Notifier{
					WorkspaceID:  workspaceID,
					Name:         "Test Mattermost Webhook Notifier",
					NotifierType: NotifierTypeMattermost,
					MattermostNotifier: &mattermost_notifier.MattermostNotifier{
						DeliveryMode:      mattermost_notifier.DeliveryModeWebhook,
						WebhookURL:        "https://mattermost.example.com/hooks/original-key",
						TargetChannelName: "town-square",
					},
				}
			},
			updateNotifier: func(workspaceID, notifierID uuid.UUID) *Notifier {
				return &Notifier{
					ID:           notifierID,
					WorkspaceID:  workspaceID,
					Name:         "Updated Mattermost Webhook Notifier",
					NotifierType: NotifierTypeMattermost,
					MattermostNotifier: &mattermost_notifier.MattermostNotifier{
						DeliveryMode:      mattermost_notifier.DeliveryModeWebhook,
						WebhookURL:        "",
						TargetChannelName: "off-topic",
					},
				}
			},
			verifySensitiveData: func(t *testing.T, notifier *Notifier) {
				assert.True(
					t,
					isEncrypted(notifier.MattermostNotifier.WebhookURL),
					"WebhookURL should be encrypted in DB",
				)
				decrypted := decryptField(t, notifier.MattermostNotifier.WebhookURL)
				assert.Equal(t, "https://mattermost.example.com/hooks/original-key", decrypted)
				assert.Equal(t, "off-topic", notifier.MattermostNotifier.TargetChannelName)
			},
			verifyHiddenData: func(t *testing.T, notifier *Notifier) {
				assert.Equal(t, "", notifier.MattermostNotifier.WebhookURL)
				assert.NotEmpty(t, notifier.MattermostNotifier.TargetChannelName)
			},
		},
		{
			name:         "Mattermost Notifier - bot account mode",
			notifierType: NotifierTypeMattermost,
			createNotifier: func(workspaceID uuid.UUID) *Notifier {
				return &Notifier{
					WorkspaceID:  workspaceID,
					Name:         "Test Mattermost Bot Notifier",
					NotifierType: NotifierTypeMattermost,
					MattermostNotifier: &mattermost_notifier.MattermostNotifier{
						DeliveryMode:    mattermost_notifier.DeliveryModeBot,
						ServerURL:       "https://mattermost.example.com",
						BotToken:        "original-mattermost-bot-token",
						TargetChannelID: "abcdefghijklmnopqrstuvwxyz",
					},
				}
			},
			updateNotifier: func(workspaceID, notifierID uuid.UUID) *Notifier {
				return &Notifier{
					ID:           notifierID,
					WorkspaceID:  workspaceID,
					Name:         "Updated Mattermost Bot Notifier",
					NotifierType: NotifierTypeMattermost,
					MattermostNotifier: &mattermost_notifier.MattermostNotifier{
						DeliveryMode:    mattermost_notifier.DeliveryModeBot,
						ServerURL:       "https://mattermost.internal.example.com",
						BotToken:        "",
						TargetChannelID: "zyxwvutsrqponmlkjihgfedcba",
					},
				}
			},
			verifySensitiveData: func(t *testing.T, notifier *Notifier) {
				assert.True(
					t,
					isEncrypted(notifier.MattermostNotifier.BotToken),
					"BotToken should be encrypted in DB",
				)
				decrypted := decryptField(t, notifier.MattermostNotifier.BotToken)
				assert.Equal(t, "original-mattermost-bot-token", decrypted)
				assert.Equal(
					t,
					"https://mattermost.internal.example.com",
					notifier.MattermostNotifier.ServerURL,
				)
			},
			verifyHiddenData: func(t *testing.T, notifier *Notifier) {
				assert.Equal(t, "", notifier.MattermostNotifier.BotToken)
				assert.NotEmpty(t, notifier.MattermostNotifier.ServerURL)
			},
		},
		{
			name:         "Webhook Notifier",
			notifierType: NotifierTypeWebhook,
			createNotifier: func(workspaceID uuid.UUID) *Notifier {
				return &Notifier{
					WorkspaceID:  workspaceID,
					Name:         "Test Webhook Notifier",
					NotifierType: NotifierTypeWebhook,
					WebhookNotifier: &webhook_notifier.WebhookNotifier{
						WebhookURL:    "https://webhook.example.com/test",
						WebhookMethod: webhook_notifier.WebhookMethodPOST,
						Headers: []webhook_notifier.WebhookHeader{
							{Key: "Authorization", Value: "Bearer my-secret-token"},
							{Key: "X-Custom-Header", Value: "custom-value"},
						},
					},
				}
			},
			updateNotifier: func(workspaceID, notifierID uuid.UUID) *Notifier {
				return &Notifier{
					ID:           notifierID,
					WorkspaceID:  workspaceID,
					Name:         "Updated Webhook Notifier",
					NotifierType: NotifierTypeWebhook,
					WebhookNotifier: &webhook_notifier.WebhookNotifier{
						WebhookURL:    "https://webhook.example.com/updated",
						WebhookMethod: webhook_notifier.WebhookMethodGET,
						Headers: []webhook_notifier.WebhookHeader{
							{Key: "Authorization", Value: "Bearer updated-token"},
						},
					},
				}
			},
			verifySensitiveData: func(t *testing.T, notifier *Notifier) {
				assert.NotEmpty(
					t,
					notifier.WebhookNotifier.WebhookURL,
					"WebhookURL should be visible",
				)
				// Verify header values are encrypted in DB
				assert.True(
					t,
					isEncrypted(notifier.WebhookNotifier.Headers[0].Value),
					"Header value should be encrypted in DB",
				)
				decrypted := decryptField(
					t,
					notifier.WebhookNotifier.Headers[0].Value,
				)
				assert.Equal(t, "Bearer updated-token", decrypted)
			},
			verifyHiddenData: func(t *testing.T, notifier *Notifier) {
				assert.NotEmpty(
					t,
					notifier.WebhookNotifier.WebhookURL,
					"WebhookURL should be visible",
				)
				for _, header := range notifier.WebhookNotifier.Headers {
					assert.Empty(t, header.Value, "Header value should be hidden")
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
			router := createRouter()
			workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

			// Phase 1: Create notifier with sensitive data
			initialNotifier := tc.createNotifier(workspace.ID)
			var createdNotifier Notifier
			test_utils.MakePostRequestAndUnmarshal(
				t,
				router,
				"/api/v1/notifiers",
				"Bearer "+owner.Token,
				*initialNotifier,
				http.StatusOK,
				&createdNotifier,
			)
			assert.NotEmpty(t, createdNotifier.ID)
			assert.Equal(t, initialNotifier.Name, createdNotifier.Name)

			// Phase 2: Read via service - sensitive data should be hidden
			var retrievedNotifier Notifier
			test_utils.MakeGetRequestAndUnmarshal(
				t,
				router,
				fmt.Sprintf("/api/v1/notifiers/%s", createdNotifier.ID.String()),
				"Bearer "+owner.Token,
				http.StatusOK,
				&retrievedNotifier,
			)
			tc.verifyHiddenData(t, &retrievedNotifier)
			assert.Equal(t, initialNotifier.Name, retrievedNotifier.Name)

			// Phase 3: Update with non-sensitive changes only (sensitive fields empty)
			updatedNotifier := tc.updateNotifier(workspace.ID, createdNotifier.ID)
			var updateResponse Notifier
			test_utils.MakePostRequestAndUnmarshal(
				t,
				router,
				"/api/v1/notifiers",
				"Bearer "+owner.Token,
				*updatedNotifier,
				http.StatusOK,
				&updateResponse,
			)
			// Verify non-sensitive fields were updated
			assert.Equal(t, updatedNotifier.Name, updateResponse.Name)

			// Phase 4: Retrieve directly from repository to verify sensitive data preservation
			repository := &NotifierRepository{}
			notifierFromDB, err := repository.FindByID(t.Context(), createdNotifier.ID)
			assert.NoError(t, err)

			// Verify original sensitive data is still present in DB
			tc.verifySensitiveData(t, notifierFromDB)

			// Verify non-sensitive fields were updated in DB
			assert.Equal(t, updatedNotifier.Name, notifierFromDB.Name)

			// Phase 5: Additional verification - Check via GET that data is still hidden
			var finalRetrieved Notifier
			test_utils.MakeGetRequestAndUnmarshal(
				t,
				router,
				fmt.Sprintf("/api/v1/notifiers/%s", createdNotifier.ID.String()),
				"Bearer "+owner.Token,
				http.StatusOK,
				&finalRetrieved,
			)
			tc.verifyHiddenData(t, &finalRetrieved)

			deleteNotifier(t, router, createdNotifier.ID, workspace.ID, owner.Token)
			workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
		})
	}
}

func Test_CreateNotifier_AllSensitiveFieldsEncryptedInDB(t *testing.T) {
	testCases := []struct {
		name                      string
		createNotifier            func(workspaceID uuid.UUID) *Notifier
		verifySensitiveEncryption func(t *testing.T, notifier *Notifier)
	}{
		{
			name: "Telegram Notifier - BotToken encrypted",
			createNotifier: func(workspaceID uuid.UUID) *Notifier {
				return &Notifier{
					WorkspaceID:  workspaceID,
					Name:         "Test Telegram",
					NotifierType: NotifierTypeTelegram,
					TelegramNotifier: &telegram_notifier.TelegramNotifier{
						BotToken:       "plain-telegram-token-123",
						TargetChatID:   "123456789",
						IsProxyEnabled: true,
						ProxyURL:       "http://proxy.example.com:8080",
					},
				}
			},
			verifySensitiveEncryption: func(t *testing.T, notifier *Notifier) {
				assert.True(
					t,
					isEncrypted(notifier.TelegramNotifier.BotToken),
					"BotToken should be encrypted",
				)
				decrypted := decryptField(t, notifier.TelegramNotifier.BotToken)
				assert.Equal(t, "plain-telegram-token-123", decrypted)

				assert.True(
					t,
					isEncrypted(notifier.TelegramNotifier.ProxyURL),
					"ProxyURL should be encrypted",
				)
				decryptedProxyURL := decryptField(t, notifier.TelegramNotifier.ProxyURL)
				assert.Equal(t, "http://proxy.example.com:8080", decryptedProxyURL)
			},
		},
		{
			name: "Email Notifier - SMTPPassword encrypted",
			createNotifier: func(workspaceID uuid.UUID) *Notifier {
				return &Notifier{
					WorkspaceID:  workspaceID,
					Name:         "Test Email",
					NotifierType: NotifierTypeEmail,
					EmailNotifier: &email_notifier.EmailNotifier{
						TargetEmail:  "test@example.com",
						SMTPHost:     "smtp.example.com",
						SMTPPort:     587,
						SMTPUser:     "user@example.com",
						SMTPPassword: "plain-smtp-password-456",
						From:         "noreply@example.com",
					},
				}
			},
			verifySensitiveEncryption: func(t *testing.T, notifier *Notifier) {
				assert.True(
					t,
					isEncrypted(notifier.EmailNotifier.SMTPPassword),
					"SMTPPassword should be encrypted",
				)
				decrypted := decryptField(t, notifier.EmailNotifier.SMTPPassword)
				assert.Equal(t, "plain-smtp-password-456", decrypted)
			},
		},
		{
			name: "Slack Notifier - BotToken encrypted",
			createNotifier: func(workspaceID uuid.UUID) *Notifier {
				return &Notifier{
					WorkspaceID:  workspaceID,
					Name:         "Test Slack",
					NotifierType: NotifierTypeSlack,
					SlackNotifier: &slack_notifier.SlackNotifier{
						BotToken:     "plain-slack-token-789",
						TargetChatID: "C0123456789",
					},
				}
			},
			verifySensitiveEncryption: func(t *testing.T, notifier *Notifier) {
				assert.True(
					t,
					isEncrypted(notifier.SlackNotifier.BotToken),
					"BotToken should be encrypted",
				)
				decrypted := decryptField(t, notifier.SlackNotifier.BotToken)
				assert.Equal(t, "plain-slack-token-789", decrypted)
			},
		},
		{
			name: "Discord Notifier - WebhookURL encrypted",
			createNotifier: func(workspaceID uuid.UUID) *Notifier {
				return &Notifier{
					WorkspaceID:  workspaceID,
					Name:         "Test Discord",
					NotifierType: NotifierTypeDiscord,
					DiscordNotifier: &discord_notifier.DiscordNotifier{
						ChannelWebhookURL: "https://discord.com/api/webhooks/123/abc",
					},
				}
			},
			verifySensitiveEncryption: func(t *testing.T, notifier *Notifier) {
				assert.True(
					t,
					isEncrypted(notifier.DiscordNotifier.ChannelWebhookURL),
					"WebhookURL should be encrypted",
				)
				decrypted := decryptField(
					t,
					notifier.DiscordNotifier.ChannelWebhookURL,
				)
				assert.Equal(t, "https://discord.com/api/webhooks/123/abc", decrypted)
			},
		},
		{
			name: "Teams Notifier - WebhookURL encrypted",
			createNotifier: func(workspaceID uuid.UUID) *Notifier {
				return &Notifier{
					WorkspaceID:  workspaceID,
					Name:         "Test Teams",
					NotifierType: NotifierTypeTeams,
					TeamsNotifier: &teams_notifier.TeamsNotifier{
						WebhookURL: "https://outlook.office.com/webhook/test123",
					},
				}
			},
			verifySensitiveEncryption: func(t *testing.T, notifier *Notifier) {
				assert.True(
					t,
					isEncrypted(notifier.TeamsNotifier.WebhookURL),
					"WebhookURL should be encrypted",
				)
				decrypted := decryptField(t, notifier.TeamsNotifier.WebhookURL)
				assert.Equal(t, "https://outlook.office.com/webhook/test123", decrypted)
			},
		},
		{
			name: "Mattermost Notifier - incoming webhook URL encrypted",
			createNotifier: func(workspaceID uuid.UUID) *Notifier {
				return &Notifier{
					WorkspaceID:  workspaceID,
					Name:         "Test Mattermost Webhook",
					NotifierType: NotifierTypeMattermost,
					MattermostNotifier: &mattermost_notifier.MattermostNotifier{
						DeliveryMode: mattermost_notifier.DeliveryModeWebhook,
						WebhookURL:   "https://mattermost.example.com/hooks/plain-key-123",
					},
				}
			},
			verifySensitiveEncryption: func(t *testing.T, notifier *Notifier) {
				assert.True(
					t,
					isEncrypted(notifier.MattermostNotifier.WebhookURL),
					"WebhookURL should be encrypted",
				)
				decrypted := decryptField(t, notifier.MattermostNotifier.WebhookURL)
				assert.Equal(t, "https://mattermost.example.com/hooks/plain-key-123", decrypted)
			},
		},
		{
			name: "Mattermost Notifier - BotToken encrypted, ServerURL not encrypted",
			createNotifier: func(workspaceID uuid.UUID) *Notifier {
				return &Notifier{
					WorkspaceID:  workspaceID,
					Name:         "Test Mattermost Bot",
					NotifierType: NotifierTypeMattermost,
					MattermostNotifier: &mattermost_notifier.MattermostNotifier{
						DeliveryMode:    mattermost_notifier.DeliveryModeBot,
						ServerURL:       "https://mattermost.example.com",
						BotToken:        "plain-mattermost-token-321",
						TargetChannelID: "abcdefghijklmnopqrstuvwxyz",
					},
				}
			},
			verifySensitiveEncryption: func(t *testing.T, notifier *Notifier) {
				assert.True(
					t,
					isEncrypted(notifier.MattermostNotifier.BotToken),
					"BotToken should be encrypted",
				)
				decrypted := decryptField(t, notifier.MattermostNotifier.BotToken)
				assert.Equal(t, "plain-mattermost-token-321", decrypted)

				assert.False(
					t,
					isEncrypted(notifier.MattermostNotifier.ServerURL),
					"ServerURL should NOT be encrypted",
				)
			},
		},
		{
			name: "Webhook Notifier - Header values encrypted, URL not encrypted",
			createNotifier: func(workspaceID uuid.UUID) *Notifier {
				return &Notifier{
					WorkspaceID:  workspaceID,
					Name:         "Test Webhook",
					NotifierType: NotifierTypeWebhook,
					WebhookNotifier: &webhook_notifier.WebhookNotifier{
						WebhookURL:    "https://webhook.example.com/test456",
						WebhookMethod: webhook_notifier.WebhookMethodPOST,
						Headers: []webhook_notifier.WebhookHeader{
							{Key: "Authorization", Value: "Bearer secret-token-12345"},
							{Key: "X-API-Key", Value: "api-key-67890"},
						},
					},
				}
			},
			verifySensitiveEncryption: func(t *testing.T, notifier *Notifier) {
				assert.False(
					t,
					isEncrypted(notifier.WebhookNotifier.WebhookURL),
					"WebhookURL should NOT be encrypted",
				)
				assert.Equal(
					t,
					"https://webhook.example.com/test456",
					notifier.WebhookNotifier.WebhookURL,
				)

				assert.True(
					t,
					isEncrypted(notifier.WebhookNotifier.Headers[0].Value),
					"Header value should be encrypted",
				)
				decrypted1 := decryptField(
					t,
					notifier.WebhookNotifier.Headers[0].Value,
				)
				assert.Equal(t, "Bearer secret-token-12345", decrypted1)

				assert.True(
					t,
					isEncrypted(notifier.WebhookNotifier.Headers[1].Value),
					"Header value should be encrypted",
				)
				decrypted2 := decryptField(
					t,
					notifier.WebhookNotifier.Headers[1].Value,
				)
				assert.Equal(t, "api-key-67890", decrypted2)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
			router := createRouter()
			workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

			// Create notifier via API (plaintext credentials)
			var createdNotifier Notifier
			test_utils.MakePostRequestAndUnmarshal(
				t,
				router,
				"/api/v1/notifiers",
				"Bearer "+owner.Token,
				tc.createNotifier(workspace.ID),
				http.StatusOK,
				&createdNotifier,
			)

			// Read from DB directly (bypass service layer)
			repository := &NotifierRepository{}
			notifierFromDB, err := repository.FindByID(t.Context(), createdNotifier.ID)
			assert.NoError(t, err)

			// Verify encryption
			tc.verifySensitiveEncryption(t, notifierFromDB)

			// Cleanup
			deleteNotifier(t, router, createdNotifier.ID, workspace.ID, owner.Token)
			workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
		})
	}
}

func Test_TransferNotifier_PermissionsEnforced(t *testing.T) {
	tests := []struct {
		name               string
		sourceRole         *users_enums.WorkspaceRole
		targetRole         *users_enums.WorkspaceRole
		isGlobalAdmin      bool
		expectSuccess      bool
		expectedStatusCode int
	}{
		{
			name:               "owner in both workspaces can transfer",
			sourceRole:         func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleOwner; return &r }(),
			targetRole:         func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleOwner; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "admin in both workspaces can transfer",
			sourceRole:         func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleAdmin; return &r }(),
			targetRole:         func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleAdmin; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "member in both workspaces can transfer",
			sourceRole:         func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleMember; return &r }(),
			targetRole:         func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleMember; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "viewer in both workspaces cannot transfer",
			sourceRole:         func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleViewer; return &r }(),
			targetRole:         func() *users_enums.WorkspaceRole { r := users_enums.WorkspaceRoleViewer; return &r }(),
			isGlobalAdmin:      false,
			expectSuccess:      false,
			expectedStatusCode: http.StatusForbidden,
		},
		{
			name:               "global admin can transfer",
			sourceRole:         nil,
			targetRole:         nil,
			isGlobalAdmin:      true,
			expectSuccess:      true,
			expectedStatusCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := createRouter()
			GetNotifierService().SetNotifierDatabaseCounter(&mockNotifierDatabaseCounter{})

			sourceOwner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
			targetOwner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)

			sourceWorkspace := workspaces_testing.CreateTestWorkspace(t.Context(),
				"Source Workspace",
				sourceOwner,
				router,
			)
			targetWorkspace := workspaces_testing.CreateTestWorkspace(t.Context(),
				"Target Workspace",
				targetOwner,
				router,
			)

			notifier := createNewNotifier(sourceWorkspace.ID)
			var savedNotifier Notifier
			test_utils.MakePostRequestAndUnmarshal(
				t,
				router,
				"/api/v1/notifiers",
				"Bearer "+sourceOwner.Token,
				*notifier,
				http.StatusOK,
				&savedNotifier,
			)

			var testUserToken string
			if tt.isGlobalAdmin {
				admin := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleAdmin)
				testUserToken = admin.Token
			} else if tt.sourceRole != nil {
				testUser := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
				workspaces_testing.AddMemberToWorkspace(
					sourceWorkspace,
					testUser,
					*tt.sourceRole,
					sourceOwner.Token,
					router,
				)
				workspaces_testing.AddMemberToWorkspace(
					targetWorkspace,
					testUser,
					*tt.targetRole,
					targetOwner.Token,
					router,
				)
				testUserToken = testUser.Token
			}

			request := TransferNotifierRequest{
				TargetWorkspaceID: targetWorkspace.ID,
			}

			testResp := test_utils.MakePostRequest(
				t,
				router,
				fmt.Sprintf("/api/v1/notifiers/%s/transfer", savedNotifier.ID.String()),
				"Bearer "+testUserToken,
				request,
				tt.expectedStatusCode,
			)

			if tt.expectSuccess {
				assert.Contains(t, string(testResp.Body), "transferred successfully")

				var retrievedNotifier Notifier
				test_utils.MakeGetRequestAndUnmarshal(
					t,
					router,
					fmt.Sprintf("/api/v1/notifiers/%s", savedNotifier.ID.String()),
					"Bearer "+targetOwner.Token,
					http.StatusOK,
					&retrievedNotifier,
				)
				assert.Equal(t, targetWorkspace.ID, retrievedNotifier.WorkspaceID)

				deleteNotifier(t, router, savedNotifier.ID, targetWorkspace.ID, targetOwner.Token)
			} else {
				assert.Contains(t, string(testResp.Body), "insufficient permissions")
				deleteNotifier(t, router, savedNotifier.ID, sourceWorkspace.ID, sourceOwner.Token)
			}

			workspaces_testing.RemoveTestWorkspace(t.Context(), sourceWorkspace, router)
			workspaces_testing.RemoveTestWorkspace(t.Context(), targetWorkspace, router)
		})
	}
}

func Test_TransferNotifierNotManagableWorkspace_TransferFailed(t *testing.T) {
	router := createRouter()
	GetNotifierService().SetNotifierDatabaseCounter(&mockNotifierDatabaseCounter{})

	userA := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	userB := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)

	workspace1 := workspaces_testing.CreateTestWorkspace(t.Context(), "Workspace 1", userA, router)
	workspace2 := workspaces_testing.CreateTestWorkspace(t.Context(), "Workspace 2", userB, router)

	notifier := createNewNotifier(workspace1.ID)
	var savedNotifier Notifier
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/notifiers",
		"Bearer "+userA.Token,
		*notifier,
		http.StatusOK,
		&savedNotifier,
	)

	request := TransferNotifierRequest{
		TargetWorkspaceID: workspace2.ID,
	}

	testResp := test_utils.MakePostRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/notifiers/%s/transfer", savedNotifier.ID.String()),
		"Bearer "+userA.Token,
		request,
		http.StatusForbidden,
	)

	assert.Contains(
		t,
		string(testResp.Body),
		"insufficient permissions to manage notifier in target workspace",
	)

	deleteNotifier(t, router, savedNotifier.ID, workspace1.ID, userA.Token)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace1, router)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace2, router)
}

type mockNotifierDatabaseCounter struct{}

func (m *mockNotifierDatabaseCounter) GetNotifierAttachedDatabasesIDs(
	notifierID uuid.UUID,
) ([]uuid.UUID, error) {
	return []uuid.UUID{}, nil
}

func createRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	v1 := router.Group("/api/v1")
	protected := v1.Group("").Use(users_middleware.AuthMiddleware(users_services.GetUserService()))

	if routerGroup, ok := protected.(*gin.RouterGroup); ok {
		GetNotifierController().RegisterRoutes(routerGroup)
		workspaces_controllers.GetWorkspaceController().RegisterRoutes(routerGroup)
		workspaces_controllers.GetMembershipController().RegisterRoutes(routerGroup)
	}

	audit_logs.SetupDependencies()
	GetNotifierService().SetNotifierDatabaseCounter(&mockNotifierDatabaseCounter{})

	return router
}

func createNewNotifier(workspaceID uuid.UUID) *Notifier {
	return &Notifier{
		WorkspaceID:  workspaceID,
		Name:         "Test Notifier " + uuid.New().String(),
		NotifierType: NotifierTypeWebhook,
		WebhookNotifier: &webhook_notifier.WebhookNotifier{
			WebhookURL:    GetWebhookStub().URL() + "/test-" + uuid.New().String(),
			WebhookMethod: webhook_notifier.WebhookMethodPOST,
		},
	}
}

func createWebhookNotifier(workspaceID uuid.UUID) *Notifier {
	return &Notifier{
		WorkspaceID:  workspaceID,
		Name:         "Test Webhook Notifier " + uuid.New().String(),
		NotifierType: NotifierTypeWebhook,
		WebhookNotifier: &webhook_notifier.WebhookNotifier{
			WebhookURL:    GetWebhookStub().URL(),
			WebhookMethod: webhook_notifier.WebhookMethodGET,
		},
	}
}

func verifyNotifierData(t *testing.T, expected, actual *Notifier) {
	assert.Equal(t, expected.Name, actual.Name)
	assert.Equal(t, expected.NotifierType, actual.NotifierType)
	assert.Equal(t, expected.WorkspaceID, actual.WorkspaceID)
}

func deleteNotifier(
	t *testing.T,
	router *gin.Engine,
	notifierID, workspaceID uuid.UUID,
	token string,
) {
	test_utils.MakeDeleteRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/notifiers/%s", notifierID.String()),
		"Bearer "+token,
		http.StatusOK,
	)
}

func isEncrypted(value string) bool {
	return len(value) > 4 && value[:4] == "enc:"
}

func decryptField(t *testing.T, encryptedValue string) string {
	encryptor := GetNotifierService().fieldEncryptor
	decrypted, err := encryptor.Decrypt(encryptedValue)
	assert.NoError(t, err)
	return decrypted
}

func Test_SaveEmailNotifier_WithUnusableSettings_Refused(t *testing.T) {
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	router := createRouter()
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	testCases := []struct {
		name                 string
		applyUnusableSetting func(emailNotifier *email_notifier.EmailNotifier)
		expectedError        string
	}{
		{
			name:                 "sender without an address",
			applyUnusableSetting: func(emailNotifier *email_notifier.EmailNotifier) { emailNotifier.From = "ops at example" },
			expectedError:        mailer.ErrInvalidSender.Error(),
		},
		{
			name:                 "unknown security mode",
			applyUnusableSetting: func(emailNotifier *email_notifier.EmailNotifier) { emailNotifier.Security = "ssl" },
			expectedError:        mailer.ErrUnknownSecurity.Error(),
		},
		{
			name: "recipient without an address",
			applyUnusableSetting: func(emailNotifier *email_notifier.EmailNotifier) {
				emailNotifier.TargetEmail = "ops at example"
			},
			expectedError: mailer.ErrInvalidRecipient.Error(),
		},
		{
			name:                 "greeting name with a space",
			applyUnusableSetting: func(emailNotifier *email_notifier.EmailNotifier) { emailNotifier.HeloName = "my relay" },
			expectedError:        mailer.ErrInvalidHeloName.Error(),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			notifier := createEmailNotifier(workspace.ID)
			testCase.applyUnusableSetting(notifier.EmailNotifier)

			response := test_utils.MakePostRequest(
				t, router, "/api/v1/notifiers", "Bearer "+owner.Token, *notifier, http.StatusBadRequest,
			)

			assert.Contains(t, string(response.Body), testCase.expectedError)
		})
	}
}

func Test_SaveEmailNotifier_WithDisplayNameSenderAndConnectionOptions_ReturnedAsSaved(t *testing.T) {
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	router := createRouter()
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

	notifier := createEmailNotifier(workspace.ID)
	notifier.EmailNotifier.From = "Acme Backups <noreply@example.com>"
	notifier.EmailNotifier.Security = mailer.SecurityNone
	notifier.EmailNotifier.HeloName = "mail.example.com"

	var savedNotifier Notifier
	test_utils.MakePostRequestAndUnmarshal(
		t, router, "/api/v1/notifiers", "Bearer "+owner.Token, *notifier, http.StatusOK, &savedNotifier,
	)

	var retrievedNotifier Notifier
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		fmt.Sprintf("/api/v1/notifiers/%s", savedNotifier.ID.String()),
		"Bearer "+owner.Token,
		http.StatusOK,
		&retrievedNotifier,
	)

	assert.Equal(t, "Acme Backups <noreply@example.com>", retrievedNotifier.EmailNotifier.From)
	assert.Equal(t, mailer.SecurityNone, retrievedNotifier.EmailNotifier.Security)
	assert.Equal(t, "mail.example.com", retrievedNotifier.EmailNotifier.HeloName)

	deleteNotifier(t, router, savedNotifier.ID, workspace.ID, owner.Token)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
}

func Test_SaveEmailNotifier_WithoutSecurity_CreatesWithPortDefaultAndUpdateKeepsStoredMode(t *testing.T) {
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	router := createRouter()
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

	notifier := createEmailNotifier(workspace.ID)

	var createdNotifier Notifier
	test_utils.MakePostRequestAndUnmarshal(
		t, router, "/api/v1/notifiers", "Bearer "+owner.Token, *notifier, http.StatusOK, &createdNotifier,
	)
	assert.Equal(t, mailer.SecurityStartTLS, createdNotifier.EmailNotifier.Security)

	createdNotifier.EmailNotifier.Security = mailer.SecurityNone
	var notifierSwitchedToNone Notifier
	test_utils.MakePostRequestAndUnmarshal(
		t, router, "/api/v1/notifiers", "Bearer "+owner.Token, createdNotifier, http.StatusOK, &notifierSwitchedToNone,
	)

	notifierSwitchedToNone.EmailNotifier.Security = ""
	notifierSwitchedToNone.EmailNotifier.SMTPPort = 465
	test_utils.MakePostRequest(
		t, router, "/api/v1/notifiers", "Bearer "+owner.Token, notifierSwitchedToNone, http.StatusOK,
	)

	var retrievedNotifier Notifier
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		fmt.Sprintf("/api/v1/notifiers/%s", createdNotifier.ID.String()),
		"Bearer "+owner.Token,
		http.StatusOK,
		&retrievedNotifier,
	)

	assert.Equal(t, 465, retrievedNotifier.EmailNotifier.SMTPPort)
	assert.Equal(t, mailer.SecurityNone, retrievedNotifier.EmailNotifier.Security)

	deleteNotifier(t, router, createdNotifier.ID, workspace.ID, owner.Token)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
}

func Test_SendTestNotificationDirect_EmailWithoutSecurity_SendsWithPortDefault(t *testing.T) {
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	router := createRouter()
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

	mailpitEndpoint := containers.StartMailpit(t)

	notifier := createEmailNotifier(workspace.ID)
	notifier.EmailNotifier.SMTPHost = mailpitEndpoint.SMTP.Host
	notifier.EmailNotifier.SMTPPort = mailpitEndpoint.SMTP.Port

	response := test_utils.MakePostRequest(
		t, router, "/api/v1/notifiers/direct-test", "Bearer "+owner.Token, *notifier, http.StatusBadRequest,
	)

	assert.Contains(t, string(response.Body), mailer.ErrStartTLSNotOffered.Error())

	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
}

func createEmailNotifier(workspaceID uuid.UUID) *Notifier {
	return &Notifier{
		WorkspaceID:  workspaceID,
		Name:         "Test Email Notifier " + uuid.New().String(),
		NotifierType: NotifierTypeEmail,
		EmailNotifier: &email_notifier.EmailNotifier{
			TargetEmail: "ops@example.com",
			SMTPHost:    "smtp.example.com",
			SMTPPort:    587,
		},
	}
}

func Test_SaveTelegramNotifier_WithBotAPIURL_ReturnedInClearAndClearedByUpdate(t *testing.T) {
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	router := createRouter()
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

	notifier := createTelegramNotifier(workspace.ID, "https://gateway.example.com/telegram")
	notifier.TelegramNotifier.IsProxyEnabled = true
	notifier.TelegramNotifier.ProxyURL = "socks5://user:password@proxy.example.com:1080"

	var savedNotifier Notifier
	test_utils.MakePostRequestAndUnmarshal(
		t, router, "/api/v1/notifiers", "Bearer "+owner.Token, *notifier, http.StatusOK, &savedNotifier,
	)

	retrievedNotifier := getNotifier(t, router, savedNotifier.ID, owner.Token)
	assert.Equal(t, "https://gateway.example.com/telegram", retrievedNotifier.TelegramNotifier.BotAPIURL)
	assert.Equal(t, "", retrievedNotifier.TelegramNotifier.BotToken)
	assert.Equal(t, "", retrievedNotifier.TelegramNotifier.ProxyURL)

	retrievedNotifier.TelegramNotifier.BotAPIURL = ""
	test_utils.MakePostRequest(
		t, router, "/api/v1/notifiers", "Bearer "+owner.Token, *retrievedNotifier, http.StatusOK,
	)

	updatedNotifier := getNotifier(t, router, savedNotifier.ID, owner.Token)
	assert.Equal(t, "", updatedNotifier.TelegramNotifier.BotAPIURL)

	deleteNotifier(t, router, savedNotifier.ID, workspace.ID, owner.Token)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
}

func createTelegramNotifier(workspaceID uuid.UUID, botAPIURL string) *Notifier {
	return &Notifier{
		WorkspaceID:  workspaceID,
		Name:         "Test Telegram Notifier " + uuid.New().String(),
		NotifierType: NotifierTypeTelegram,
		TelegramNotifier: &telegram_notifier.TelegramNotifier{
			BotToken:     testTelegramBotToken,
			TargetChatID: "-1001234567890",
			BotAPIURL:    botAPIURL,
		},
	}
}

func getNotifier(t *testing.T, router *gin.Engine, notifierID uuid.UUID, token string) *Notifier {
	var retrievedNotifier Notifier
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		fmt.Sprintf("/api/v1/notifiers/%s", notifierID.String()),
		"Bearer "+token,
		http.StatusOK,
		&retrievedNotifier,
	)

	return &retrievedNotifier
}

const (
	testTelegramBotToken       = "123456:AAH-TESTBOTTOKEN"
	testTelegramBotTokenSecret = "AAH-TESTBOTTOKEN"
)

func Test_SaveTelegramNotifier_WithUnusableBotAPIURL_RefusedAndStoredNotifierUnchanged(t *testing.T) {
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	router := createRouter()
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

	var savedNotifier Notifier
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/notifiers",
		"Bearer "+owner.Token,
		*createTelegramNotifier(workspace.ID, "https://gateway.example.com/telegram"),
		http.StatusOK,
		&savedNotifier,
	)

	for _, unusableBotAPIURL := range []string{
		"ftp://telegram.example.com",
		"https://user:password@gateway.example.com",
		"https://gateway.example.com/telegram?key=1",
	} {
		t.Run(unusableBotAPIURL, func(t *testing.T) {
			newNotifier := createTelegramNotifier(workspace.ID, unusableBotAPIURL)
			test_utils.MakePostRequest(
				t, router, "/api/v1/notifiers", "Bearer "+owner.Token, *newNotifier, http.StatusBadRequest,
			)

			updatedNotifier := getNotifier(t, router, savedNotifier.ID, owner.Token)
			updatedNotifier.TelegramNotifier.BotAPIURL = unusableBotAPIURL
			test_utils.MakePostRequest(
				t, router, "/api/v1/notifiers", "Bearer "+owner.Token, *updatedNotifier, http.StatusBadRequest,
			)

			storedNotifier := getNotifier(t, router, savedNotifier.ID, owner.Token)
			assert.Equal(t, "https://gateway.example.com/telegram", storedNotifier.TelegramNotifier.BotAPIURL)
		})
	}

	var notifiers []Notifier
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		fmt.Sprintf("/api/v1/notifiers?workspace_id=%s", workspace.ID.String()),
		"Bearer "+owner.Token,
		http.StatusOK,
		&notifiers,
	)
	assert.Len(t, notifiers, 1)

	deleteNotifier(t, router, savedNotifier.ID, workspace.ID, owner.Token)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
}

func Test_SendTestNotificationExisting_TelegramWithBotAPIURL_DeliveredUnderPathPrefix(t *testing.T) {
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	router := createRouter()
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)

	botAPIServer := startTelegramBotAPIStub(t, http.StatusOK, `{"ok":true}`)

	var savedNotifier Notifier
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/notifiers",
		"Bearer "+owner.Token,
		*createTelegramNotifier(workspace.ID, botAPIServer.URL+"/telegram/"),
		http.StatusOK,
		&savedNotifier,
	)

	test_utils.MakePostRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/notifiers/%s/test", savedNotifier.ID.String()),
		"Bearer "+owner.Token,
		nil,
		http.StatusOK,
	)

	receivedRequests := botAPIServer.GetReceivedRequests()
	assert.Len(t, receivedRequests, 1)
	assert.Equal(t, "/telegram/bot"+testTelegramBotToken+"/sendMessage", receivedRequests[0].Path)
	assert.Equal(t, "-1001234567890", receivedRequests[0].ChatID)

	deleteNotifier(t, router, savedNotifier.ID, workspace.ID, owner.Token)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
}

func Test_SendTestNotificationDirect_TelegramWithBotAPIURL_DeliveredUnderPathPrefix(t *testing.T) {
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	router := createRouter()
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	botAPIServer := startTelegramBotAPIStub(t, http.StatusOK, `{"ok":true}`)
	notifier := createTelegramNotifier(workspace.ID, botAPIServer.URL+"/telegram")

	test_utils.MakePostRequest(
		t, router, "/api/v1/notifiers/direct-test", "Bearer "+owner.Token, *notifier, http.StatusOK,
	)

	receivedRequests := botAPIServer.GetReceivedRequests()
	assert.Len(t, receivedRequests, 1)
	assert.Equal(t, "/telegram/bot"+testTelegramBotToken+"/sendMessage", receivedRequests[0].Path)
	assert.Equal(t, "-1001234567890", receivedRequests[0].ChatID)
}

func Test_SendTestNotificationDirect_TelegramWithUnusableBotAPIURL_RefusedWithoutRequest(t *testing.T) {
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	router := createRouter()
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	botAPIServer := startTelegramBotAPIStub(t, http.StatusOK, `{"ok":true}`)
	botAPIHost := strings.TrimPrefix(botAPIServer.URL, "http://")

	for _, unusableBotAPIURL := range []string{
		"ftp://" + botAPIHost + "/telegram",
		"http://user:password@" + botAPIHost + "/telegram",
	} {
		t.Run(unusableBotAPIURL, func(t *testing.T) {
			notifier := createTelegramNotifier(workspace.ID, unusableBotAPIURL)

			test_utils.MakePostRequest(
				t, router, "/api/v1/notifiers/direct-test", "Bearer "+owner.Token, *notifier, http.StatusBadRequest,
			)
		})
	}

	assert.Empty(t, botAPIServer.GetReceivedRequests())
}

func Test_SendTestNotificationDirect_TelegramWithBotAPIURLAndProxy_SentThroughProxy(t *testing.T) {
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	router := createRouter()
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	proxyServer := startTelegramBotAPIStub(t, http.StatusOK, `{"ok":true}`)

	notifier := createTelegramNotifier(workspace.ID, "http://telegram-bot-api.invalid:8081/telegram")
	notifier.TelegramNotifier.IsProxyEnabled = true
	notifier.TelegramNotifier.ProxyURL = proxyServer.URL

	test_utils.MakePostRequest(
		t, router, "/api/v1/notifiers/direct-test", "Bearer "+owner.Token, *notifier, http.StatusOK,
	)

	receivedRequests := proxyServer.GetReceivedRequests()
	assert.Len(t, receivedRequests, 1)
	assert.Equal(
		t,
		"http://telegram-bot-api.invalid:8081/telegram/bot"+testTelegramBotToken+"/sendMessage",
		receivedRequests[0].RequestURI,
	)
}

func Test_SendTestNotificationDirect_TelegramBotAPIUnreachable_ErrorNamesHostWithoutBotToken(t *testing.T) {
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	router := createRouter()
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	closedServer := httptest.NewServer(http.NotFoundHandler())
	closedServer.Close()

	notifier := createTelegramNotifier(workspace.ID, closedServer.URL+"/telegram")

	response := test_utils.MakePostRequest(
		t, router, "/api/v1/notifiers/direct-test", "Bearer "+owner.Token, *notifier, http.StatusBadRequest,
	)

	assert.Contains(t, string(response.Body), strings.TrimPrefix(closedServer.URL, "http://"))
	assert.NotContains(t, string(response.Body), testTelegramBotTokenSecret)
}

type telegramBotAPIRequest struct {
	Path       string
	RequestURI string
	ChatID     string
}

type telegramBotAPIStub struct {
	URL string

	mu               sync.Mutex
	receivedRequests []telegramBotAPIRequest
}

func (s *telegramBotAPIStub) GetReceivedRequests() []telegramBotAPIRequest {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]telegramBotAPIRequest(nil), s.receivedRequests...)
}

// Also serves as a forward proxy: a proxied request arrives with the target's absolute URL as its
// RequestURI.
func startTelegramBotAPIStub(t *testing.T, statusCode int, responseBody string) *telegramBotAPIStub {
	stub := &telegramBotAPIStub{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestBody, _ := io.ReadAll(r.Body)
		formValues, _ := url.ParseQuery(string(requestBody))

		stub.mu.Lock()
		stub.receivedRequests = append(stub.receivedRequests, telegramBotAPIRequest{
			Path:       r.URL.Path,
			RequestURI: r.RequestURI,
			ChatID:     formValues.Get("chat_id"),
		})
		stub.mu.Unlock()

		w.WriteHeader(statusCode)
		_, _ = io.WriteString(w, strings.ReplaceAll(responseBody, "{path}", r.URL.Path))
	}))
	t.Cleanup(server.Close)

	stub.URL = server.URL

	return stub
}

func Test_SendTestNotificationDirect_TelegramBotAPIEchoesRequestPath_ErrorKeepsBodyWithoutBotToken(t *testing.T) {
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	router := createRouter()
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	botAPIServer := startTelegramBotAPIStub(t, http.StatusNotFound, "Cannot POST {path}")
	notifier := createTelegramNotifier(workspace.ID, botAPIServer.URL+"/telegram")

	response := test_utils.MakePostRequest(
		t, router, "/api/v1/notifiers/direct-test", "Bearer "+owner.Token, *notifier, http.StatusBadRequest,
	)

	assert.Contains(t, string(response.Body), "Cannot POST /telegram/botredacted/sendMessage")
	assert.NotContains(t, string(response.Body), testTelegramBotTokenSecret)
}

func Test_ListNotifiers_WithUndecryptableWebhookURL_ListsNotifierAndRefusesToSend(t *testing.T) {
	owner := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleMember)
	router := createRouter()
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
	defer workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)

	var savedNotifier Notifier
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/notifiers",
		"Bearer "+owner.Token,
		*createNewNotifier(workspace.ID),
		http.StatusOK,
		&savedNotifier,
	)
	defer deleteNotifier(t, router, savedNotifier.ID, workspace.ID, owner.Token)

	// The API stores only values it can decrypt, so a ciphertext under a lost or rotated key is
	// written directly.
	err := storage.GetDb().
		Exec(
			"UPDATE webhook_notifiers SET webhook_url = ? WHERE notifier_id = ?",
			"enc:not-a-valid-ciphertext",
			savedNotifier.ID,
		).Error
	require.NoError(t, err)

	var listedNotifiers []Notifier
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		fmt.Sprintf("/api/v1/notifiers?workspace_id=%s", workspace.ID.String()),
		"Bearer "+owner.Token,
		http.StatusOK,
		&listedNotifiers,
	)

	require.Len(t, listedNotifiers, 1)
	assert.Equal(t, savedNotifier.ID, listedNotifiers[0].ID)

	GetWebhookStub().ResetCalls()
	testResponse := test_utils.MakePostRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/notifiers/%s/test", savedNotifier.ID.String()),
		"Bearer "+owner.Token,
		nil,
		http.StatusBadRequest,
	)

	assert.Contains(t, string(testResponse.Body), "failed to decrypt webhook URL")
	assert.Equal(t, 0, GetWebhookStub().CallCount())
}
