// Package auth provides Supabase JWT verification and HTTP middleware.
//
// The middleware is *optional* by default — requests without a valid token
// proceed anonymously, and handlers can check for a user via UserIDFromContext.
// This lets device_id-based anonymous flows coexist with authenticated ones
// during the migration to full accounts.
package auth

import "context"

// contextKey is unexported so only this package can write to it.
type contextKey int

const (
	userIDKey contextKey = iota
	emailKey
	roleKey
	appRolesKey // slice of application roles from user_roles table
)

// WithUser stores the authenticated user's Supabase ID, email, and role in ctx.
func WithUser(ctx context.Context, userID, email, role string) context.Context {
	ctx = context.WithValue(ctx, userIDKey, userID)
	ctx = context.WithValue(ctx, emailKey, email)
	ctx = context.WithValue(ctx, roleKey, role)
	return ctx
}

// WithAppRoles stores the user's application-level roles (from user_roles table) in ctx.
func WithAppRoles(ctx context.Context, roles []string) context.Context {
	return context.WithValue(ctx, appRolesKey, roles)
}

// AppRolesFromContext returns the user's application roles loaded from the DB.
func AppRolesFromContext(ctx context.Context) []string {
	v, _ := ctx.Value(appRolesKey).([]string)
	return v
}

// UserIDFromContext returns the authenticated user's Supabase UUID, if any.
// The second return value is false when the request is anonymous.
func UserIDFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(userIDKey).(string)
	if !ok || v == "" {
		return "", false
	}
	return v, true
}

// EmailFromContext returns the authenticated user's email, if any.
func EmailFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(emailKey).(string)
	if !ok || v == "" {
		return "", false
	}
	return v, true
}

// IsSiteAdminFromContext returns true when the user is a site admin —
// either via Supabase app_metadata.role="admin" or via user_roles table.
func IsSiteAdminFromContext(ctx context.Context) bool {
	v, _ := ctx.Value(roleKey).(string)
	if v == "admin" {
		return true
	}
	for _, r := range AppRolesFromContext(ctx) {
		if r == "site_admin" {
			return true
		}
	}
	return false
}

// IsAdminFromContext is an alias for IsSiteAdminFromContext for backwards compatibility.
func IsAdminFromContext(ctx context.Context) bool {
	return IsSiteAdminFromContext(ctx)
}

// IsDataAdminFromContext returns true when the user has at least data_admin rights
// (either site_admin or data_admin role).
func IsDataAdminFromContext(ctx context.Context) bool {
	if IsSiteAdminFromContext(ctx) {
		return true
	}
	for _, r := range AppRolesFromContext(ctx) {
		if r == "data_admin" {
			return true
		}
	}
	return false
}
