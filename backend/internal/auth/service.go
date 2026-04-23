package auth

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"backend/internal/config"

	"github.com/go-jose/go-jose/v4"
	josejwt "github.com/go-jose/go-jose/v4/jwt"
	"github.com/google/uuid"
)

type ctxKey string

const principalContextKey ctxKey = "auth.principal"

type Principal struct {
	Authenticated      bool     `json:"authenticated"`
	UserID             string   `json:"userId"`
	Email              string   `json:"email"`
	DisplayName        string   `json:"displayName"`
	Status             string   `json:"status"`
	Roles              []string `json:"roles"`
	Provider           string   `json:"provider"`
	Subject            string   `json:"subject"`
	EmailVerified      bool     `json:"emailVerified"`
	RegistrationNeeded bool     `json:"registrationNeeded"`
	RejectionReason    string   `json:"rejectionReason"`
}

type SessionResponse struct {
	Authenticated bool      `json:"authenticated"`
	AuthMode      string    `json:"authMode"`
	AuthProvider  string    `json:"authProvider"`
	RBACMode      string    `json:"rbacMode"`
	User          Principal `json:"user"`
}

type RegistrationInput struct {
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
}

type UserSummary struct {
	ID              string   `json:"id"`
	Email           string   `json:"email"`
	DisplayName     string   `json:"displayName"`
	Status          string   `json:"status"`
	Roles           []string `json:"roles"`
	Provider        string   `json:"provider"`
	LastLoginAt     string   `json:"lastLoginAt"`
	UpdatedAt       string   `json:"updatedAt"`
	RejectionReason string   `json:"rejectionReason"`
}

type RoleSummary struct {
	Key         string `json:"key"`
	Description string `json:"description"`
}

type ApproveUserInput struct {
	Roles []string `json:"roles"`
}

type RejectUserInput struct {
	Reason string `json:"reason"`
}

type Claims struct {
	Provider      string
	Subject       string
	Email         string
	DisplayName   string
	EmailVerified bool
	Roles         []string
}

type TokenVerifier interface {
	Verify(ctx context.Context, rawToken string) (Claims, error)
}

type Service struct {
	cfg      config.AuthConfig
	repo     *Repository
	verifier TokenVerifier
}

func NewService(cfg config.AuthConfig, repo *Repository) (*Service, error) {
	verifier, err := newTokenVerifier(cfg)
	if err != nil {
		return nil, err
	}
	return &Service{cfg: cfg, repo: repo, verifier: verifier}, nil
}

func (s *Service) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}

		principal, err := s.CurrentPrincipal(r.Context(), bearerToken(r.Header.Get("Authorization")))
		if err != nil {
			if s.cfg.Mode == "dry_run" {
				next.ServeHTTP(w, r.WithContext(WithPrincipal(r.Context(), Principal{Authenticated: false, Status: "anonymous"})))
				return
			}
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		if s.cfg.Mode == "enforced" && requiresAuthenticatedUser(r.URL.Path) {
			if !principal.Authenticated {
				http.Error(w, "authentication required", http.StatusUnauthorized)
				return
			}
			if s.cfg.RequireEmailVerified && !principal.EmailVerified && requiresVerifiedEmail(r.URL.Path) {
				http.Error(w, "email verification is required", http.StatusUnauthorized)
				return
			}
		}

		next.ServeHTTP(w, r.WithContext(WithPrincipal(r.Context(), principal)))
	})
}

func (s *Service) CurrentPrincipal(ctx context.Context, rawToken string) (Principal, error) {
	if s.cfg.Mode == "none" {
		return Principal{Authenticated: false, Status: "anonymous"}, nil
	}

	if strings.TrimSpace(rawToken) == "" {
		if s.cfg.Mode == "dry_run" {
			return Principal{Authenticated: false, Status: "anonymous"}, nil
		}
		return Principal{}, fmt.Errorf("missing bearer token")
	}

	claims, err := s.verifier.Verify(ctx, rawToken)
	if err != nil {
		if s.cfg.Mode == "dry_run" {
			return Principal{Authenticated: false, Status: "anonymous"}, nil
		}
		return Principal{}, err
	}
	user, err := s.repo.FindUser(ctx, claims.Provider, claims.Subject, claims.Email)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return Principal{}, err
	}
	if errors.Is(err, sql.ErrNoRows) {
		return Principal{
			Authenticated:      true,
			Email:              claims.Email,
			DisplayName:        claims.DisplayName,
			Status:             "unregistered",
			Roles:              claims.Roles,
			Provider:           claims.Provider,
			Subject:            claims.Subject,
			EmailVerified:      claims.EmailVerified,
			RegistrationNeeded: true,
		}, nil
	}

	_ = s.repo.TouchLastLogin(ctx, user.ID)
	return Principal{
		Authenticated:      true,
		UserID:             user.ID,
		Email:              user.Email,
		DisplayName:        user.DisplayName,
		Status:             user.Status,
		Roles:              user.Roles,
		Provider:           defaultString(user.Provider, claims.Provider),
		Subject:            claims.Subject,
		EmailVerified:      claims.EmailVerified,
		RegistrationNeeded: user.Status == "unregistered",
		RejectionReason:    user.RejectionReason,
	}, nil
}

func (s *Service) Session(ctx context.Context, principal Principal) SessionResponse {
	return SessionResponse{
		Authenticated: principal.Authenticated,
		AuthMode:      s.cfg.Mode,
		AuthProvider:  s.cfg.Verifier,
		RBACMode:      s.cfg.RBAC,
		User:          principal,
	}
}

func (s *Service) Register(ctx context.Context, principal Principal, input RegistrationInput) (UserSummary, error) {
	email := strings.TrimSpace(input.Email)
	if principal.Email != "" {
		email = principal.Email
	}
	if email == "" {
		return UserSummary{}, fmt.Errorf("email is required")
	}
	displayName := strings.TrimSpace(input.DisplayName)
	if displayName == "" {
		displayName = defaultString(principal.DisplayName, email)
	}
	user, err := s.repo.UpsertPendingRegistration(ctx, principal.Provider, principal.Subject, email, displayName)
	if err != nil {
		return UserSummary{}, err
	}
	hasActiveUsers, err := s.repo.HasActiveUsers(ctx)
	if err != nil {
		return UserSummary{}, err
	}
	if !hasActiveUsers {
		return s.repo.UpdateUserStatus(ctx, user.ID, "active", "", []string{"admin"})
	}
	return user, nil
}

func (s *Service) ListUsers(ctx context.Context) ([]UserSummary, error) {
	return s.repo.ListUsers(ctx)
}

func (s *Service) ListRoles(ctx context.Context) ([]RoleSummary, error) {
	return s.repo.ListRoles(ctx)
}

func (s *Service) ApproveUser(ctx context.Context, id string, roles []string) (UserSummary, error) {
	normalized := normalizeRoles(roles)
	if len(normalized) == 0 {
		return UserSummary{}, fmt.Errorf("at least one role is required")
	}
	return s.repo.UpdateUserStatus(ctx, id, "active", "", normalized)
}

func (s *Service) RejectUser(ctx context.Context, id, reason string) (UserSummary, error) {
	if strings.TrimSpace(reason) == "" {
		reason = "No reason provided"
	}
	return s.repo.UpdateUserStatus(ctx, id, "rejected", reason, nil)
}

func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, principalContextKey, principal)
}

func PrincipalFromContext(ctx context.Context) Principal {
	principal, _ := ctx.Value(principalContextKey).(Principal)
	return principal
}

func Allowed(principal Principal, role string) bool {
	if slices.Contains(principal.Roles, "admin") {
		return true
	}
	return slices.Contains(principal.Roles, role)
}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

type userRow struct {
	ID              string
	Email           string
	DisplayName     string
	Status          string
	Roles           []string
	Provider        string
	Subject         string
	LastLoginAt     sql.NullTime
	UpdatedAt       time.Time
	RejectionReason string
}

func (r *Repository) FindUser(ctx context.Context, provider, subject, email string) (UserSummary, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT
			u.id::text,
			u.email,
			u.display_name,
			u.status,
			COALESCE(array_agg(ur.role_key ORDER BY ur.role_key) FILTER (WHERE ur.role_key IS NOT NULL), ARRAY[]::text[]),
			COALESCE(u.identity_provider, ''),
			COALESCE(u.identity_subject, ''),
			u.last_login_at,
			u.updated_at,
			COALESCE(u.rejection_reason, '')
		FROM app_users u
		LEFT JOIN user_roles ur ON ur.user_id = u.id
		WHERE (u.identity_provider = $1 AND u.identity_subject = $2 AND $2 <> '')
		   OR (LOWER(u.email) = LOWER($3) AND $3 <> '')
		GROUP BY u.id
		ORDER BY CASE WHEN u.identity_subject = $2 AND $2 <> '' THEN 0 ELSE 1 END
		LIMIT 1
	`, provider, subject, email)

	var record userRow
	if err := row.Scan(
		&record.ID,
		&record.Email,
		&record.DisplayName,
		&record.Status,
		&record.Roles,
		&record.Provider,
		&record.Subject,
		&record.LastLoginAt,
		&record.UpdatedAt,
		&record.RejectionReason,
	); err != nil {
		return UserSummary{}, err
	}
	return toUserSummary(record), nil
}

func (r *Repository) TouchLastLogin(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE app_users SET last_login_at = NOW(), updated_at = NOW() WHERE id = $1::uuid`, id)
	return err
}

func (r *Repository) UpsertPendingRegistration(ctx context.Context, provider, subject, email, displayName string) (UserSummary, error) {
	id := uuid.New().String()
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO app_users (id, email, display_name, status, identity_provider, identity_subject, updated_at, rejection_reason)
		VALUES ($1::uuid, $2, $3, 'pending', NULLIF($4, ''), NULLIF($5, ''), NOW(), '')
		ON CONFLICT (email) DO UPDATE
		SET display_name = EXCLUDED.display_name,
		    status = 'pending',
		    identity_provider = COALESCE(NULLIF(EXCLUDED.identity_provider, ''), app_users.identity_provider),
		    identity_subject = COALESCE(NULLIF(EXCLUDED.identity_subject, ''), app_users.identity_subject),
		    rejection_reason = '',
		    updated_at = NOW()
		RETURNING
			id::text,
			email,
			display_name,
			status,
			COALESCE(identity_provider, ''),
			COALESCE(identity_subject, ''),
			last_login_at,
			updated_at,
			COALESCE(rejection_reason, '')
	`, id, email, displayName, provider, subject)

	var record userRow
	if err := row.Scan(
		&record.ID,
		&record.Email,
		&record.DisplayName,
		&record.Status,
		&record.Provider,
		&record.Subject,
		&record.LastLoginAt,
		&record.UpdatedAt,
		&record.RejectionReason,
	); err != nil {
		return UserSummary{}, fmt.Errorf("upsert registration: %w", err)
	}
	record.Roles = []string{}
	return toUserSummary(record), nil
}

func (r *Repository) HasActiveUsers(ctx context.Context) (bool, error) {
	var exists bool
	if err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM app_users WHERE status = 'active')`).Scan(&exists); err != nil {
		return false, fmt.Errorf("check active users: %w", err)
	}
	return exists, nil
}

func (r *Repository) ListUsers(ctx context.Context) ([]UserSummary, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			u.id::text,
			u.email,
			u.display_name,
			u.status,
			COALESCE(array_agg(ur.role_key ORDER BY ur.role_key) FILTER (WHERE ur.role_key IS NOT NULL), ARRAY[]::text[]),
			COALESCE(u.identity_provider, ''),
			COALESCE(u.identity_subject, ''),
			u.last_login_at,
			u.updated_at,
			COALESCE(u.rejection_reason, '')
		FROM app_users u
		LEFT JOIN user_roles ur ON ur.user_id = u.id
		GROUP BY u.id
		ORDER BY CASE u.status WHEN 'pending' THEN 0 WHEN 'rejected' THEN 1 ELSE 2 END, u.display_name
	`)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	result := []UserSummary{}
	for rows.Next() {
		var record userRow
		if err := rows.Scan(
			&record.ID,
			&record.Email,
			&record.DisplayName,
			&record.Status,
			&record.Roles,
			&record.Provider,
			&record.Subject,
			&record.LastLoginAt,
			&record.UpdatedAt,
			&record.RejectionReason,
		); err != nil {
			return nil, fmt.Errorf("scan users: %w", err)
		}
		result = append(result, toUserSummary(record))
	}
	return result, rows.Err()
}

func (r *Repository) ListRoles(ctx context.Context) ([]RoleSummary, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT key, description FROM roles ORDER BY key`)
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}
	defer rows.Close()

	result := []RoleSummary{}
	for rows.Next() {
		var row RoleSummary
		if err := rows.Scan(&row.Key, &row.Description); err != nil {
			return nil, fmt.Errorf("scan roles: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func (r *Repository) UpdateUserStatus(ctx context.Context, id, status, rejectionReason string, roles []string) (UserSummary, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return UserSummary{}, fmt.Errorf("begin user status tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		UPDATE app_users
		SET status = $2,
		    rejection_reason = $3,
		    updated_at = NOW()
		WHERE id = $1::uuid
	`, id, status, rejectionReason); err != nil {
		return UserSummary{}, fmt.Errorf("update user status: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM user_roles WHERE user_id = $1::uuid`, id); err != nil {
		return UserSummary{}, fmt.Errorf("clear user roles: %w", err)
	}

	if status == "active" {
		for _, role := range roles {
			if _, err := tx.ExecContext(ctx, `INSERT INTO user_roles (user_id, role_key) VALUES ($1::uuid, $2)`, id, role); err != nil {
				return UserSummary{}, fmt.Errorf("assign user role: %w", err)
			}
		}
	}

	row := tx.QueryRowContext(ctx, `
		SELECT
			u.id::text,
			u.email,
			u.display_name,
			u.status,
			COALESCE(array_agg(ur.role_key ORDER BY ur.role_key) FILTER (WHERE ur.role_key IS NOT NULL), ARRAY[]::text[]),
			COALESCE(u.identity_provider, ''),
			COALESCE(u.identity_subject, ''),
			u.last_login_at,
			u.updated_at,
			COALESCE(u.rejection_reason, '')
		FROM app_users u
		LEFT JOIN user_roles ur ON ur.user_id = u.id
		WHERE u.id = $1::uuid
		GROUP BY u.id
	`, id)

	var record userRow
	if err := row.Scan(
		&record.ID,
		&record.Email,
		&record.DisplayName,
		&record.Status,
		&record.Roles,
		&record.Provider,
		&record.Subject,
		&record.LastLoginAt,
		&record.UpdatedAt,
		&record.RejectionReason,
	); err != nil {
		return UserSummary{}, fmt.Errorf("reload user status: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return UserSummary{}, fmt.Errorf("commit user status: %w", err)
	}
	return toUserSummary(record), nil
}

func toUserSummary(record userRow) UserSummary {
	lastLoginAt := ""
	if record.LastLoginAt.Valid {
		lastLoginAt = record.LastLoginAt.Time.UTC().Format(time.RFC3339)
	}
	return UserSummary{
		ID:              record.ID,
		Email:           record.Email,
		DisplayName:     record.DisplayName,
		Status:          record.Status,
		Roles:           record.Roles,
		Provider:        record.Provider,
		LastLoginAt:     lastLoginAt,
		UpdatedAt:       record.UpdatedAt.UTC().Format(time.RFC3339),
		RejectionReason: record.RejectionReason,
	}
}

type localTokenVerifier struct {
	tokens map[string]Claims
}

func (v *localTokenVerifier) Verify(_ context.Context, rawToken string) (Claims, error) {
	if claims, ok := v.tokens[rawToken]; ok {
		return claims, nil
	}
	if strings.HasPrefix(rawToken, "local:") {
		payload := strings.TrimPrefix(rawToken, "local:")
		parts := strings.Split(payload, "|")
		if len(parts) < 2 {
			return Claims{}, fmt.Errorf("invalid local token format")
		}
		roles := []string{}
		if len(parts) >= 3 {
			roles = normalizeRoles(strings.Split(parts[2], ","))
		}
		email := strings.TrimSpace(parts[0])
		return Claims{
			Provider:      "local",
			Subject:       "local:" + strings.ToLower(email),
			Email:         email,
			DisplayName:   strings.TrimSpace(parts[1]),
			EmailVerified: true,
			Roles:         roles,
		}, nil
	}
	return Claims{}, fmt.Errorf("unknown local token")
}

type jwksTokenVerifier struct {
	issuer            string
	audience          string
	jwksURL           string
	allowedAlgorithms []jose.SignatureAlgorithm
	client            *http.Client

	mu         sync.RWMutex
	cachedKeys *jose.JSONWebKeySet
	expiresAt  time.Time
}

func (v *jwksTokenVerifier) Verify(ctx context.Context, rawToken string) (Claims, error) {
	token, err := josejwt.ParseSigned(rawToken, v.allowedAlgorithms)
	if err != nil {
		return Claims{}, fmt.Errorf("parse jwt: %w", err)
	}
	keys, err := v.fetchKeySet(ctx)
	if err != nil {
		return Claims{}, err
	}
	header := token.Headers[0]
	var key any
	for _, candidate := range keys.Keys {
		if header.KeyID == "" || candidate.KeyID == header.KeyID {
			key = candidate.Key
			break
		}
	}
	if key == nil {
		return Claims{}, fmt.Errorf("signing key not found in JWKS")
	}

	var std josejwt.Claims
	custom := struct {
		Email         string   `json:"email"`
		Name          string   `json:"name"`
		EmailVerified bool     `json:"email_verified"`
		Roles         []string `json:"roles"`
		Groups        []string `json:"groups"`
	}{}
	if err := token.Claims(key, &std, &custom); err != nil {
		return Claims{}, fmt.Errorf("verify jwt claims: %w", err)
	}

	expected := josejwt.Expected{Issuer: v.issuer, Time: time.Now()}
	if v.audience != "" {
		expected.AnyAudience = []string{v.audience}
	}
	if err := std.Validate(expected); err != nil {
		return Claims{}, fmt.Errorf("validate jwt claims: %w", err)
	}

	roles := custom.Roles
	if len(roles) == 0 {
		roles = custom.Groups
	}
	return Claims{
		Provider:      "oidc",
		Subject:       std.Subject,
		Email:         custom.Email,
		DisplayName:   defaultString(custom.Name, custom.Email),
		EmailVerified: custom.EmailVerified,
		Roles:         normalizeRoles(roles),
	}, nil
}

func (v *jwksTokenVerifier) fetchKeySet(ctx context.Context) (*jose.JSONWebKeySet, error) {
	v.mu.RLock()
	if v.cachedKeys != nil && time.Now().Before(v.expiresAt) {
		keys := v.cachedKeys
		v.mu.RUnlock()
		return keys, nil
	}
	v.mu.RUnlock()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.jwksURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build jwks request: %w", err)
	}
	resp, err := v.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch jwks: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch jwks: unexpected status %d", resp.StatusCode)
	}

	var keys jose.JSONWebKeySet
	if err := json.NewDecoder(resp.Body).Decode(&keys); err != nil {
		return nil, fmt.Errorf("decode jwks: %w", err)
	}

	v.mu.Lock()
	v.cachedKeys = &keys
	v.expiresAt = time.Now().Add(5 * time.Minute)
	v.mu.Unlock()
	return &keys, nil
}

func newTokenVerifier(cfg config.AuthConfig) (TokenVerifier, error) {
	switch cfg.Verifier {
	case "", "local":
		return &localTokenVerifier{tokens: parseLocalTokenSpec(cfg.LocalTokenSpec)}, nil
	case "oidc", "jwks":
		if strings.TrimSpace(cfg.JWKSURL) == "" || strings.TrimSpace(cfg.ExpectedIssuer) == "" {
			return nil, fmt.Errorf("OIDC_JWKS_URL and OIDC_EXPECTED_ISSUER are required for JWKS auth")
		}
		return &jwksTokenVerifier{
			issuer:            cfg.ExpectedIssuer,
			audience:          cfg.ExpectedAudience,
			jwksURL:           cfg.JWKSURL,
			allowedAlgorithms: parseSigningAlgorithms(cfg.SigningAlgorithms),
			client:            &http.Client{Timeout: 10 * time.Second},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported jwt verifier: %s", cfg.Verifier)
	}
}

func parseSigningAlgorithms(raw []string) []jose.SignatureAlgorithm {
	allowed := make([]jose.SignatureAlgorithm, 0, len(raw))
	for _, item := range raw {
		switch strings.TrimSpace(strings.ToUpper(item)) {
		case "RS256":
			allowed = append(allowed, jose.RS256)
		case "RS384":
			allowed = append(allowed, jose.RS384)
		case "RS512":
			allowed = append(allowed, jose.RS512)
		case "ES256":
			allowed = append(allowed, jose.ES256)
		case "ES384":
			allowed = append(allowed, jose.ES384)
		case "ES512":
			allowed = append(allowed, jose.ES512)
		}
	}
	if len(allowed) == 0 {
		return []jose.SignatureAlgorithm{jose.RS256}
	}
	return allowed
}

func parseLocalTokenSpec(spec string) map[string]Claims {
	result := map[string]Claims{
		"local-admin-token": {
			Provider: "local", Subject: "seed-admin", Email: "admin@example.local", DisplayName: "Local Admin", EmailVerified: true,
		},
		"local-operator-token": {
			Provider: "local", Subject: "seed-operator", Email: "operator@example.local", DisplayName: "Local Operator", EmailVerified: true,
		},
		"local-inventory-token": {
			Provider: "local", Subject: "seed-inventory", Email: "inventory@example.local", DisplayName: "Local Inventory", EmailVerified: true,
		},
		"local-procurement-token": {
			Provider: "local", Subject: "seed-procurement", Email: "procurement@example.local", DisplayName: "Local Procurement", EmailVerified: true,
		},
		"local-inspector-token": {
			Provider: "local", Subject: "seed-inspector", Email: "inspector@example.local", DisplayName: "Local Inspector", EmailVerified: true,
		},
	}
	for token, claims := range result {
		result[token] = Claims{
			Provider:      claims.Provider,
			Subject:       claims.Subject,
			Email:         claims.Email,
			DisplayName:   claims.DisplayName,
			EmailVerified: true,
		}
	}
	for _, entry := range strings.Split(spec, ";") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 {
			continue
		}
		token := strings.TrimSpace(parts[0])
		fields := strings.Split(parts[1], "|")
		if token == "" || len(fields) < 2 {
			continue
		}
		roles := []string{}
		if len(fields) >= 3 {
			roles = normalizeRoles(strings.Split(fields[2], ","))
		}
		email := strings.TrimSpace(fields[0])
		result[token] = Claims{
			Provider:      "local",
			Subject:       "local:" + strings.ToLower(email),
			Email:         email,
			DisplayName:   strings.TrimSpace(fields[1]),
			EmailVerified: true,
			Roles:         roles,
		}
	}
	return result
}

func normalizeRoles(roles []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(roles))
	for _, role := range roles {
		normalized := strings.TrimSpace(strings.ToLower(role))
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	slices.Sort(result)
	return result
}

func requiresAuthenticatedUser(path string) bool {
	return strings.HasPrefix(path, "/api/v1/auth/me") ||
		strings.HasPrefix(path, "/api/v1/admin/users") ||
		strings.HasPrefix(path, "/api/v1/admin/roles")
}

func requiresVerifiedEmail(path string) bool {
	return !strings.HasPrefix(path, "/api/v1/auth/me") &&
		!strings.HasPrefix(path, "/api/v1/auth/register")
}

func bearerToken(header string) string {
	if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(header)), "bearer ") {
		return ""
	}
	return strings.TrimSpace(header[7:])
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func DecodeJWTUnsafe(rawToken string) map[string]any {
	parts := strings.Split(rawToken, ".")
	if len(parts) < 2 {
		return map[string]any{}
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return map[string]any{}
	}
	out := map[string]any{}
	_ = json.Unmarshal(payload, &out)
	return out
}
