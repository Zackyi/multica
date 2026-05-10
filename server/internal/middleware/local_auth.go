package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

const (
	// DefaultLocalUser is the user for local mode
	DefaultLocalUserEmail = "local@multica.local"
	DefaultLocalUserName  = "Local User"
)

// IsLocalMode returns true if MULTICA_LOCAL_MODE is set to "true"
func IsLocalMode() bool {
	return os.Getenv("MULTICA_LOCAL_MODE") == "true"
}

// LocalAuth creates a default user and workspace on first request in local mode,
// then skips authentication for all requests.
func LocalAuth(queries *db.Queries) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !IsLocalMode() {
				// Not in local mode, use normal auth
				Auth(queries)(next).ServeHTTP(w, r)
				return
			}

			// Local mode: set default user headers
			ctx := r.Context()

			// Try to get existing local user first
			user, err := queries.GetUserByEmail(ctx, DefaultLocalUserEmail)
			if err != nil {
				// Create default user (ID is auto-generated)
				user, err = queries.CreateUser(ctx, db.CreateUserParams{
					Name:  DefaultLocalUserName,
					Email: DefaultLocalUserEmail,
				})
				if err != nil {
					slog.Error("local mode: failed to create default user", "error", err)
					http.Error(w, `{"error":"failed to initialize local mode"}`, http.StatusInternalServerError)
					return
				}
				slog.Info("local mode: created default user", "user_id", uuidToString(user.ID))
			}

			// Ensure workspace exists (check by getting user's workspaces)
			workspaces, err := queries.ListWorkspaces(ctx, user.ID)
			if err != nil || len(workspaces) == 0 {
				// Create default workspace (ID is auto-generated)
				workspace, err := queries.CreateWorkspace(ctx, db.CreateWorkspaceParams{
					Name: "Local Workspace",
					Slug: "local-workspace",
				})
				if err != nil {
					slog.Error("local mode: failed to create default workspace", "error", err)
					http.Error(w, `{"error":"failed to initialize local mode"}`, http.StatusInternalServerError)
					return
				}

				// Add user as owner
				_, err = queries.CreateMember(ctx, db.CreateMemberParams{
					WorkspaceID: workspace.ID,
					UserID:      user.ID,
					Role:        "owner",
				})
				if err != nil {
					slog.Error("local mode: failed to add user to workspace", "error", err)
				}
				slog.Info("local mode: created default workspace", "workspace_id", uuidToString(workspace.ID))
				// Refresh workspaces list
				workspaces, _ = queries.ListWorkspaces(ctx, user.ID)
			}

			// Set headers for downstream handlers
			r.Header.Set("X-User-ID", uuidToString(user.ID))
			r.Header.Set("X-User-Email", user.Email)
			r.Header.Set("X-Local-Mode", "true")

			// Set workspace in context if not already set
			if WorkspaceIDFromContext(ctx) == "" && len(workspaces) > 0 {
				ws := workspaces[0]
				ctx = context.WithValue(ctx, ctxKeyWorkspaceID, uuidToString(ws.ID))
				member, _ := queries.GetMemberByUserAndWorkspace(ctx, db.GetMemberByUserAndWorkspaceParams{
					UserID:      user.ID,
					WorkspaceID: ws.ID,
				})
				ctx = context.WithValue(ctx, ctxKeyMember, member)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// LocalDaemonAuth is the daemon auth variant that also supports local mode
func LocalDaemonAuth(queries *db.Queries) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !IsLocalMode() {
				// Not in local mode, use normal daemon auth
				DaemonAuth(queries)(next).ServeHTTP(w, r)
				return
			}

			// Local mode: set daemon context headers
			ctx := r.Context()

			// Get the local user by email (more reliable than hardcoded UUID)
			user, err := queries.GetUserByEmail(ctx, DefaultLocalUserEmail)
			if err != nil {
				slog.Error("local mode: failed to get local user", "error", err)
				http.Error(w, `{"error":"local mode not initialized"}`, http.StatusInternalServerError)
				return
			}

			workspaces, err := queries.ListWorkspaces(ctx, user.ID)
			if err != nil || len(workspaces) == 0 {
				slog.Error("local mode: failed to get workspace", "error", err)
				http.Error(w, `{"error":"local mode not initialized"}`, http.StatusInternalServerError)
				return
			}

			ws := workspaces[0]
			ctx = context.WithValue(ctx, ctxKeyDaemonWorkspaceID, uuidToString(ws.ID))
			ctx = context.WithValue(ctx, ctxKeyDaemonID, "local-daemon")
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// LocalModeInfo returns information about local mode for health checks
func LocalModeInfo() map[string]string {
	if IsLocalMode() {
		return map[string]string{
			"local_mode":   "enabled",
			"default_user": DefaultLocalUserEmail,
		}
	}
	return map[string]string{
		"local_mode": "disabled",
	}
}
