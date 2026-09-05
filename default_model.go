package arboreal

import (
	"context"
)

const defaultModelContextKey contextKey = "arboreal_default_model"

// WithDefaultModel returns a copy of ctx carrying the default model URI, ready
// to pass to RunLoop / Execute. States and planners that do not name a model
// of their own use this one.
//
// The URI should carry a provider prefix, e.g.
// "anthropic:claude-sonnet-4-20250514". A URI whose prefix is not a
// recognized provider — including a bare model name — is routed to the
// OpenAI provider.
func WithDefaultModel(ctx context.Context, uri string) context.Context {
	return context.WithValue(ctx, defaultModelContextKey, uri)
}

// DefaultModelFromContext returns the model URI stored by WithDefaultModel, if any.
// The bool reports only that WithDefaultModel was called; an empty URI is
// treated as unset everywhere in the framework.
func DefaultModelFromContext(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	uri, ok := ctx.Value(defaultModelContextKey).(string)
	return uri, ok
}
