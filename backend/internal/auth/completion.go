package auth

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type RolePermissionUpdateInput struct {
	Permissions []string `json:"permissions"`
}

func (s *Service) BootstrapRegister(ctx context.Context, input RegistrationInput) (UserSummary, error) {
	hasActiveUsers, err := s.hasBlockingActiveUsers(ctx)
	if err != nil {
		return UserSummary{}, err
	}
	if hasActiveUsers {
		return UserSummary{}, fmt.Errorf("bootstrap registration is only allowed when zero active users exist")
	}

	email := defaultString(input.Email, "admin@example.local")
	displayName := defaultString(input.DisplayName, email)
	username := defaultString(input.Username, "admin")
	user, err := s.repo.UpsertPendingRegistration(
		ctx,
		"bootstrap",
		"bootstrap:"+email,
		email,
		displayName,
		username,
		"admin",
		input.Memo,
		input.HostedDomain,
	)
	if err != nil {
		return UserSummary{}, err
	}
	return s.repo.UpdateUserStatus(ctx, user.ID, "active", "", []string{"admin"})
}

func (s *Service) Permissions(ctx context.Context) ([]PermissionSummary, error) {
	return s.repo.ListPermissions(ctx)
}

func (s *Service) UpdateRolePermissions(ctx context.Context, roleKey string, permissions []string) (RoleSummary, error) {
	if roleKey == "" {
		return RoleSummary{}, fmt.Errorf("role key is required")
	}
	return s.repo.UpdateRolePermissions(ctx, roleKey, normalizeRoles(permissions))
}

func (s *Service) UserStatusHistory(ctx context.Context, userID string) ([]UserStatusHistoryEntry, error) {
	if userID == "" {
		return nil, fmt.Errorf("user id is required")
	}
	return s.repo.UserStatusHistory(ctx, userID)
}

func (r *Repository) ListPermissions(ctx context.Context) ([]PermissionSummary, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT key, description FROM permissions ORDER BY key`)
	if err != nil {
		return nil, fmt.Errorf("list permissions: %w", err)
	}
	defer rows.Close()

	result := []PermissionSummary{}
	for rows.Next() {
		var row PermissionSummary
		if err := rows.Scan(&row.Key, &row.Description); err != nil {
			return nil, fmt.Errorf("scan permissions: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func (r *Repository) UpdateRolePermissions(ctx context.Context, roleKey string, permissions []string) (RoleSummary, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return RoleSummary{}, fmt.Errorf("begin role permission tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM role_permissions WHERE role_key = $1`, roleKey); err != nil {
		return RoleSummary{}, fmt.Errorf("clear role permissions: %w", err)
	}
	for _, permission := range permissions {
		if _, err := tx.ExecContext(ctx, `INSERT INTO role_permissions (role_key, permission_key) VALUES ($1, $2)`, roleKey, permission); err != nil {
			return RoleSummary{}, fmt.Errorf("insert role permission: %w", err)
		}
	}

	var result RoleSummary
	if err := tx.QueryRowContext(ctx, `
		SELECT
			r.key,
			r.description,
			COALESCE(array_agg(rp.permission_key ORDER BY rp.permission_key) FILTER (WHERE rp.permission_key IS NOT NULL), ARRAY[]::text[])
		FROM roles r
		LEFT JOIN role_permissions rp ON rp.role_key = r.key
		WHERE r.key = $1
		GROUP BY r.key, r.description
	`, roleKey).Scan(&result.Key, &result.Description, &result.Permissions); err != nil {
		return RoleSummary{}, fmt.Errorf("reload role permissions: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return RoleSummary{}, fmt.Errorf("commit role permission tx: %w", err)
	}
	return result, nil
}

func (r *Repository) UserStatusHistory(ctx context.Context, userID string) ([]UserStatusHistoryEntry, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, status, reason, changed_by, created_at
		FROM user_status_history
		WHERE user_id = $1::uuid
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list user status history: %w", err)
	}
	defer rows.Close()

	result := []UserStatusHistoryEntry{}
	for rows.Next() {
		var row UserStatusHistoryEntry
		var createdAt time.Time
		if err := rows.Scan(&row.ID, &row.Status, &row.Reason, &row.ChangedBy, &createdAt); err != nil {
			return nil, fmt.Errorf("scan user status history: %w", err)
		}
		row.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		result = append(result, row)
	}
	return result, rows.Err()
}

func (r *Repository) recordUserStatusHistory(ctx context.Context, tx queryExecutor, userID, status, reason, changedBy string) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO user_status_history (id, user_id, status, reason, changed_by, created_at)
		VALUES ($1, $2::uuid, $3, $4, $5, NOW())
	`, uuid.New().String(), userID, status, reason, changedBy)
	if err != nil {
		return fmt.Errorf("insert user status history: %w", err)
	}
	return nil
}

type queryExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}
