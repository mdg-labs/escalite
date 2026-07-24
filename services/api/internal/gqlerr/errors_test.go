package gqlerr_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/api/internal/gqlerr"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
)

func TestPresentMapsCodedError(t *testing.T) {
	err := gqlerr.New(handlers.CodeForbidden, "forbidden")
	presented := gqlerr.Present(context.Background(), err)

	require.Equal(t, "forbidden", presented.Message)
	require.Equal(t, handlers.CodeForbidden, presented.Extensions["code"])
}

func TestPresentDefaultsToInternal(t *testing.T) {
	presented := gqlerr.Present(context.Background(), context.Canceled)

	require.Equal(t, handlers.CodeInternal, presented.Extensions["code"])
}
