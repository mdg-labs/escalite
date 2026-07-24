package graphql

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"

	"github.com/mdg-labs/escalite/services/api/internal/handlers"
)

// DepthLimit rejects queries that exceed a maximum selection depth.
type DepthLimit struct {
	MaxDepth int
}

var _ interface {
	graphql.OperationContextMutator
	graphql.HandlerExtension
} = DepthLimit{}

func (d DepthLimit) ExtensionName() string {
	return "DepthLimit"
}

func (d DepthLimit) Validate(schema graphql.ExecutableSchema) error {
	return nil
}

func (d DepthLimit) MutateOperationContext(
	ctx context.Context,
	opCtx *graphql.OperationContext,
) *gqlerror.Error {
	if d.MaxDepth <= 0 {
		return nil
	}

	op := opCtx.Doc.Operations.ForName(opCtx.OperationName)
	depth := maxQueryDepth(opCtx.Doc, op)
	if depth > d.MaxDepth {
		err := gqlerror.Errorf("query depth %d exceeds the limit of %d", depth, d.MaxDepth)
		err.Extensions = map[string]any{"code": handlers.CodeValidation}
		return err
	}

	return nil
}

func maxQueryDepth(doc *ast.QueryDocument, op *ast.OperationDefinition) int {
	if doc == nil || op == nil {
		return 0
	}

	fragments := make(map[string]*ast.FragmentDefinition, len(doc.Fragments))
	for _, fragment := range doc.Fragments {
		fragments[fragment.Name] = fragment
	}

	return selectionSetDepth(op.SelectionSet, fragments, 1)
}

func selectionSetDepth(
	set ast.SelectionSet,
	fragments map[string]*ast.FragmentDefinition,
	depth int,
) int {
	max := depth
	for _, sel := range set {
		switch selection := sel.(type) {
		case *ast.Field:
			if len(selection.SelectionSet) == 0 {
				continue
			}
			if childDepth := selectionSetDepth(selection.SelectionSet, fragments, depth+1); childDepth > max {
				max = childDepth
			}
		case *ast.InlineFragment:
			if childDepth := selectionSetDepth(selection.SelectionSet, fragments, depth); childDepth > max {
				max = childDepth
			}
		case *ast.FragmentSpread:
			fragment, ok := fragments[selection.Name]
			if !ok {
				continue
			}
			if childDepth := selectionSetDepth(fragment.SelectionSet, fragments, depth); childDepth > max {
				max = childDepth
			}
		}
	}

	return max
}
