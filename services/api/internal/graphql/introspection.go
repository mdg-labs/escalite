package graphql

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/vektah/gqlparser/v2/gqlerror"

	"github.com/mdg-labs/escalite/services/api/internal/auth"
)

// ConditionalIntrospection enables schema introspection in development and for authenticated
// production requests only.
type ConditionalIntrospection struct {
	Production bool
}

var _ interface {
	graphql.OperationContextMutator
	graphql.HandlerExtension
} = ConditionalIntrospection{}

func (c ConditionalIntrospection) ExtensionName() string {
	return "ConditionalIntrospection"
}

func (c ConditionalIntrospection) Validate(schema graphql.ExecutableSchema) error {
	return nil
}

func (c ConditionalIntrospection) MutateOperationContext(
	ctx context.Context,
	opCtx *graphql.OperationContext,
) *gqlerror.Error {
	if !c.Production {
		return extension.Introspection{}.MutateOperationContext(ctx, opCtx)
	}

	if _, ok := auth.SessionFromContext(ctx); ok {
		return extension.Introspection{}.MutateOperationContext(ctx, opCtx)
	}

	opCtx.DisableIntrospection = true
	return nil
}
