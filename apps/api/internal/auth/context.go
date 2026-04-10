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
)

// WithUser stores the authenticated user's Supabase ID, email, and role in ctx.
func WithUser(ctx context.Context, userID, email, role string) context.Context {
	ctx = context.WithValue(ctx, userIDKey, userID)
	ctx = context.WithValue(ctx, emailKey, email)
	ctx = context.WithValue(ctx, roleKey, role)
	return ctx
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

// IsAdminFromContext returns true when the authenticated user has the "admin" role.
func IsAdminFromContext(ctx context.Context) bool {
	v, _ := ctx.Value(roleKey).(string)
	return v == "admin"
}
