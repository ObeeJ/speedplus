package observability

import (
	"context"
	"log/slog"

	"github.com/getsentry/sentry-go"
)

// CaptureError logs the error via slog and sends it to Sentry.
// Call this at every site where an unexpected error is handled — not for
// expected domain errors (ErrNotFound, ErrInvalidCredentials, etc.).
//
// attrs are additional slog key-value pairs, e.g. "order_id", id.String().
func CaptureError(ctx context.Context, err error, msg string, attrs ...any) {
	if err == nil {
		return
	}

	// Structured log — request_id propagated if set on context
	args := append([]any{"error", err}, attrs...)
	slog.ErrorContext(ctx, msg, args...)

	// Sentry — attach request context if hub is on context
	hub := sentry.GetHubFromContext(ctx)
	if hub == nil {
		hub = sentry.CurrentHub()
	}
	hub.WithScope(func(scope *sentry.Scope) {
		// Attach all attrs as Sentry extras
		for i := 0; i+1 < len(attrs); i += 2 {
			if key, ok := attrs[i].(string); ok {
				scope.SetExtra(key, attrs[i+1])
			}
		}
		scope.SetExtra("msg", msg)
		hub.CaptureException(err)
	})
}

// CaptureMessage sends an informational message to Sentry (no error object).
// Use for unexpected-but-non-fatal conditions worth tracking.
func CaptureMessage(ctx context.Context, msg string, attrs ...any) {
	slog.WarnContext(ctx, msg, attrs...)

	hub := sentry.GetHubFromContext(ctx)
	if hub == nil {
		hub = sentry.CurrentHub()
	}
	hub.CaptureMessage(msg)
}

// SetRequestContext attaches Sentry hub to the context so downstream
// CaptureError calls carry the request's transaction/user context.
// Call this once per request in middleware.
func SetRequestContext(ctx context.Context) context.Context {
	hub := sentry.CurrentHub().Clone()
	return sentry.SetHubOnContext(ctx, hub)
}
