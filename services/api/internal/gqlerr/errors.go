package gqlerr

import (
	"context"
	"errors"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/v2/gqlerror"

	"github.com/mdg-labs/escalite/services/api/internal/handlers"
)

const complexityLimitCode = "COMPLEXITY_LIMIT_EXCEEDED"

// CodedError is a GraphQL resolver error with a stable extensions.code value.
type CodedError struct {
	Code    string
	Message string
}

func (e *CodedError) Error() string {
	return e.Message
}

// New returns an error that maps to extensions.code in GraphQL responses.
func New(code, message string) error {
	return &CodedError{Code: code, Message: message}
}

// Present maps resolver and protocol errors to GraphQL extensions.code per doc 08.
func Present(ctx context.Context, err error) *gqlerror.Error {
	gqlErr := graphql.DefaultErrorPresenter(ctx, err)

	if gqlErr.Extensions == nil {
		gqlErr.Extensions = map[string]any{}
	}

	var coded *CodedError
	if errors.As(err, &coded) {
		gqlErr.Extensions["code"] = coded.Code
		gqlErr.Message = coded.Message
		return gqlErr
	}

	if code, ok := gqlErr.Extensions["code"].(string); ok && code != "" {
		if code == complexityLimitCode {
			gqlErr.Extensions["code"] = handlers.CodeValidation
		}
		return gqlErr
	}

	gqlErr.Extensions["code"] = handlers.CodeInternal
	return gqlErr
}
