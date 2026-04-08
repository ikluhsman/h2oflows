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
)

// WithUser stores the authenticated user's Supabase ID and email in ctx.
func WithUser(ctx context.Context, userID, email string) context.Context {
	ctx = context.WithValue(ctx, userIDKey, userID)
	ctx = context.WithValue(ctx, emailKey, email)
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
