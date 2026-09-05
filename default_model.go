package arboreal

import (
	"context"
	"log"
	"sync"

	"github.com/zenful-ai/arboreal/llm"
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

// fallbackModelURI is what a state or planner uses when neither it nor the
// context names a model. It is transitional: a later release removes it and
// makes a missing model an error.
const fallbackModelURI = llm.GPT4oMini

// fallbackWarning guards the deprecation warning so it is emitted once per
// process rather than on every turn.
var fallbackWarning sync.Once

// resolveModelURI picks the model URI to use: explicit if set, else ctxDefault
// if set, else fallbackModelURI. usedFallback reports that neither an explicit
// choice nor a context default was available.
func resolveModelURI(explicit, ctxDefault string) (uri string, usedFallback bool) {
	if explicit != "" {
		return explicit, false
	}
	if ctxDefault != "" {
		return ctxDefault, false
	}
	return fallbackModelURI, true
}

// modelURIFor resolves the model URI for one call site. explicit is the
// caller's own choice (an options field or executive field); when it is empty
// the context default from WithDefaultModel is used; when that is absent too,
// the transitional fallback is used and a deprecation warning is logged once.
func modelURIFor(ctx context.Context, explicit string) string {
	ctxDefault, _ := DefaultModelFromContext(ctx)
	uri, usedFallback := resolveModelURI(explicit, ctxDefault)
	if usedFallback {
		fallbackWarning.Do(func() {
			log.Printf("arboreal: no default model configured; falling back to %s — call arboreal.WithDefaultModel(ctx, \"provider:model\"). This fallback will be removed.", fallbackModelURI)
		})
	}
	return uri
}
