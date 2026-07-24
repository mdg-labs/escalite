package handlers

import (
	"context"
	"net/http"
)

type gqlHTTPContextKey int

const (
	gqlResponseWriterKey gqlHTTPContextKey = iota
	gqlRequestKey
)

// WithGraphQLHTTP stores the current HTTP request and response writer on the context.
func WithGraphQLHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) context.Context {
	ctx = context.WithValue(ctx, gqlResponseWriterKey, w)
	ctx = context.WithValue(ctx, gqlRequestKey, r)
	return ctx
}

// GraphQLResponseWriterFromContext returns the response writer for the active GraphQL request.
func GraphQLResponseWriterFromContext(ctx context.Context) (http.ResponseWriter, bool) {
	w, ok := ctx.Value(gqlResponseWriterKey).(http.ResponseWriter)
	return w, ok
}

// GraphQLRequestFromContext returns the HTTP request for the active GraphQL request.
func GraphQLRequestFromContext(ctx context.Context) (*http.Request, bool) {
	r, ok := ctx.Value(gqlRequestKey).(*http.Request)
	return r, ok
}

// GraphQLHTTPContextMiddleware attaches the request and response writer to the request context.
func GraphQLHTTPContextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := WithGraphQLHTTP(r.Context(), w, r)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
