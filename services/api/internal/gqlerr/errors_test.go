package gqlerr_test

import (
	"context"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"testing"

	"github.com/allure-framework/allure-go/testify/require"

	"github.com/mdg-labs/escalite/services/api/internal/gqlerr"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
)

func TestPresentMapsCodedError(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		err := gqlerr.New(handlers.CodeForbidden, "forbidden")
		presented := gqlerr.Present(context.Background(), err)

		require.Equal(a, "forbidden", presented.Message)
		require.Equal(a, handlers.CodeForbidden, presented.Extensions["code"])
	})
}

func TestPresentDefaultsToInternal(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		presented := gqlerr.Present(context.Background(), context.Canceled)

		require.Equal(a, handlers.CodeInternal, presented.Extensions["code"])
	})
}
