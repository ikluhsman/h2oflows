package auth

import (
	"encoding/json"
	"net/http"
	"strings"
)

// Optional returns middleware that attaches user claims to the request
// context when a valid Authorization: Bearer <token> header is present,
// but lets anonymous requests through untouched.
//
// Use this during the device_id → authenticated migration so handlers can
// progressively layer on "if you're signed in, associate the write to
// your user_id; otherwise fall back to the device_id flow".
func Optional(v *Verifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := bearerToken(r)
			if token != "" && v != nil {
				if claims, err := v.Verify(r.Context(), token); err == nil {
					r = r.WithContext(WithUser(r.Context(), claims.UserID, claims.Email, claims.Role))
				}
				// Invalid tokens are silently ignored in Optional mode —
				// the request continues as anonymous. Endpoints that must
				// refuse bad tokens should use Required.
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Required returns middleware that rejects any request without a valid
// Supabase JWT. Use on endpoints that should never be reachable anonymously
// (account settings, personal profile writes, etc.).
func Required(v *Verifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if v == nil {
				writeJSONError(w, http.StatusServiceUnavailable, "auth not configured")
				return
			}
			token := bearerToken(r)
			if token == "" {
				writeJSONError(w, http.StatusUnauthorized, "missing bearer token")
				return
			}
			claims, err := v.Verify(r.Context(), token)
			if err != nil {
				writeJSONError(w, http.StatusUnauthorized, "invalid token")
				return
			}
			ctx := WithUser(r.Context(), claims.UserID, claims.Email, claims.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAdmin returns middleware that rejects requests from non-admin users.
// It relies on the Optional (or Required) middleware having already run and
// populated the role in context. Returns 401 for unauthenticated requests and
// 403 for authenticated non-admin users.
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := UserIDFromContext(r.Context()); !ok {
			writeJSONError(w, http.StatusUnauthorized, "authentication required")
			return
		}
		if !IsAdminFromContext(r.Context()) {
			writeJSONError(w, http.StatusForbidden, "admin role required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// LoadAppRoles returns middleware that loads the authenticated user's application
// roles from the DB (via querier) and stores them in context. Must run after
// Optional or Required. The querier receives the user ID and returns role strings.
// Failures are non-fatal — the request proceeds with no app roles.
func LoadAppRoles(querier func(r *http.Request, userID string) ([]string, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if userID, ok := UserIDFromContext(r.Context()); ok {
				if roles, err := querier(r, userID); err == nil {
					r = r.WithContext(WithAppRoles(r.Context(), roles))
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireDataAdmin rejects requests from users without at least data_admin rights.
// Returns 401 for unauthenticated requests and 403 for authenticated non-admin users.
func RequireDataAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := UserIDFromContext(r.Context()); !ok {
			writeJSONError(w, http.StatusUnauthorized, "authentication required")
			return
		}
		if !IsDataAdminFromContext(r.Context()) {
			writeJSONError(w, http.StatusForbidden, "data admin role required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// bearerToken extracts the token from the Authorization header, if present.
// Returns an empty string when no Bearer token is attached.
func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if h == "" {
		return ""
	}
	const prefix = "Bearer "
	if len(h) < len(prefix) || !strings.EqualFold(h[:len(prefix)], prefix) {
		return ""
	}
	return strings.TrimSpace(h[len(prefix):])
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
