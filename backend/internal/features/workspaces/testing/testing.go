//go:build testhelpers

package workspaces_testing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	users_testing "databasus-backend/internal/features/users/testing"
	"databasus-backend/internal/modules/workspaces"
	"databasus-backend/internal/storage"
)

// TestWorkspace is a workspace the helpers inserted.
type TestWorkspace struct {
	ID   uuid.UUID
	Name string
}

// setGinTestMode runs once because gin.SetMode writes a process-wide variable, and parallel tests build
// their routers at the same time.
var setGinTestMode = sync.OnceFunc(func() { gin.SetMode(gin.TestMode) })

func CreateTestRouter(controllers ...ControllerInterface) *gin.Engine {
	setGinTestMode()
	router := gin.New()

	v1 := router.Group("/api/v1")
	protected := v1.Group("").Use(users_testing.GetSessionMiddleware())

	for _, controller := range controllers {
		if routerGroup, ok := protected.(*gin.RouterGroup); ok {
			controller.RegisterRoutes(routerGroup)
		}
	}

	return router
}

// CreateTestWorkspace inserts a workspace owned by the user, as the workspaces module creates one.
func CreateTestWorkspace(ctx context.Context, name string, owner *users_testing.TestUser) *TestWorkspace {
	workspace, err := insertWorkspace(ctx, name, owner.UserID)
	if err != nil {
		panic(err)
	}

	return workspace
}

// AddMemberToWorkspace inserts a member of the workspace with the role. A workspace has one owner,
// so a new owner takes the role over and the previous owner becomes an admin, as in an ownership
// transfer.
func AddMemberToWorkspace(workspace *TestWorkspace, member *users_testing.TestUser, role workspaces.Role) {
	err := storage.GetDb().Transaction(func(tx *gorm.DB) error {
		if role == workspaces.RoleOwner {
			err := tx.Exec("UPDATE workspaces_members SET role = ? WHERE workspace_id = ? AND role = ?",
				workspaces.RoleAdmin, workspace.ID, workspaces.RoleOwner).Error
			if err != nil {
				return err
			}
		}

		return tx.Exec(
			"INSERT INTO workspaces_members (id, user_id, workspace_id, role, created_at) VALUES (?, ?, ?, ?, ?)",
			uuid.New(), member.UserID, workspace.ID, role, time.Now().UTC(),
		).Error
	})
	if err != nil {
		panic(fmt.Errorf("add a member to a test workspace: %w", err))
	}
}

// RemoveTestWorkspace deletes the workspace's databases, notifiers and storages before the
// workspace, standing in for the deletion the workspaces module runs through its subscribers, which
// only internal/app subscribes. The context is stripped of cancellation because callers pass t.Context() from a t.Cleanup, and the
// test context is already cancelled by the time cleanup runs.
func RemoveTestWorkspace(ctx context.Context, workspace *TestWorkspace) {
	err := removeWorkspace(context.WithoutCancel(ctx), workspace.ID)
	if err != nil {
		panic(err)
	}
}

func MakeAPIRequest(
	router *gin.Engine,
	method, url, authToken string,
	body any,
) *httptest.ResponseRecorder {
	var requestBody *bytes.Buffer
	if body != nil {
		bodyJSON, err := json.Marshal(body)
		if err != nil {
			panic(err)
		}
		requestBody = bytes.NewBuffer(bodyJSON)
	} else {
		requestBody = bytes.NewBuffer(nil)
	}

	req, err := http.NewRequestWithContext(context.Background(), method, url, requestBody)
	if err != nil {
		panic(err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if authToken != "" {
		req.Header.Set("Authorization", authToken)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// CreateTestWorkspaceDirect serves a test that has the owner's ID but no test user.
func CreateTestWorkspaceDirect(name string, ownerID uuid.UUID) (*TestWorkspace, error) {
	return insertWorkspace(context.Background(), name, ownerID)
}

func RemoveTestWorkspaceDirect(workspaceID uuid.UUID) error {
	return removeWorkspace(context.Background(), workspaceID)
}

// insertWorkspace inserts a workspace with its owner's member in one transaction, as the workspaces
// module creates one.
func insertWorkspace(ctx context.Context, name string, ownerID uuid.UUID) (*TestWorkspace, error) {
	workspace := &TestWorkspace{ID: uuid.New(), Name: name}
	now := time.Now().UTC()

	err := storage.GetDb().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Exec("INSERT INTO workspaces (id, name, created_at) VALUES (?, ?, ?)",
			workspace.ID, workspace.Name, now).Error
		if err != nil {
			return err
		}

		return tx.Exec(
			"INSERT INTO workspaces_members (id, user_id, workspace_id, role, created_at) VALUES (?, ?, ?, ?, ?)",
			uuid.New(), ownerID, workspace.ID, workspaces.RoleOwner, now,
		).Error
	})
	if err != nil {
		return nil, fmt.Errorf("insert a test workspace: %w", err)
	}

	return workspace, nil
}

// A workspace's verification and physical records must be removed before its databases and storages.
func removeWorkspace(ctx context.Context, workspaceID uuid.UUID) error {
	return storage.GetDb().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, statement := range []string{
			"DELETE FROM verification_runs WHERE database_id IN (SELECT id FROM databases WHERE workspace_id = ?)",
			"DELETE FROM verification_configs WHERE database_id IN (SELECT id FROM databases WHERE workspace_id = ?)",
			"DELETE FROM backups_physical_wal_stream_samples WHERE database_id IN (SELECT id FROM databases WHERE workspace_id = ?)",
			"DELETE FROM backups_physical_wal_streamers WHERE database_id IN (SELECT id FROM databases WHERE workspace_id = ?)",
			"DELETE FROM backups_physical_wal_segments WHERE database_id IN (SELECT id FROM databases WHERE workspace_id = ?)",
			"DELETE FROM backups_physical_wal_history_files WHERE database_id IN (SELECT id FROM databases WHERE workspace_id = ?)",
			"DELETE FROM backups_physical WHERE database_id IN (SELECT id FROM databases WHERE workspace_id = ?)",
			"DELETE FROM healthcheck_attempts WHERE database_id IN (SELECT id FROM databases WHERE workspace_id = ?)",
			"DELETE FROM healthcheck_configs WHERE database_id IN (SELECT id FROM databases WHERE workspace_id = ?)",
			"DELETE FROM databases WHERE workspace_id = ?",
			"DELETE FROM notifiers WHERE workspace_id = ?",
			"DELETE FROM storages WHERE workspace_id = ?",
			"DELETE FROM workspaces WHERE id = ?",
		} {
			err := tx.Exec(statement, workspaceID).Error
			if err != nil {
				return fmt.Errorf("remove a test workspace: %w", err)
			}
		}

		return nil
	})
}
