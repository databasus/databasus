package httpserver

import (
	"fmt"
	"log/slog"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"
	"github.com/gin-gonic/gin"

	"databasus-backend/internal/lib/apierror"
)

// RequireContractOperation guards the routes served from the contract: a request whose method and
// path the contract does not describe gets 500 and never reaches its handler. A route mounted by
// mistake therefore cannot expose an operation that skips the contract's access mark. The lookup
// runs per request and puts the found operation's mark into the request context, where
// GetAccessMark reads it for the access check. An unknown mark fails here, at startup.
func RequireContractOperation(contract []byte, logger *slog.Logger) (gin.HandlerFunc, error) {
	contractRouter, err := NewContractRouter(contract)
	if err != nil {
		return nil, err
	}

	err = checkAccessMarks(contract)
	if err != nil {
		return nil, err
	}

	return func(ginContext *gin.Context) {
		route, _, err := contractRouter.FindRoute(ginContext.Request)
		if err == nil {
			mark, _ := readAccessMark(route.Operation)
			ginContext.Request = ginContext.Request.WithContext(withAccessMark(ginContext.Request.Context(), mark))

			ginContext.Next()

			return
		}

		logger.ErrorContext(
			ginContext.Request.Context(),
			"route is mounted for the contract but has no operation in it",
			slog.String("method", ginContext.Request.Method),
			slog.String("path", ginContext.Request.URL.Path),
			slog.Any("error", err),
		)

		apierror.Write(ginContext, apierror.NewInternalProblem())
	}, nil
}

func checkAccessMarks(contract []byte) error {
	loadedContract, err := openapi3.NewLoader().LoadFromData(contract)
	if err != nil {
		return fmt.Errorf("parse the API contract: %w", err)
	}

	for _, pathItem := range loadedContract.Paths.Map() {
		for _, operation := range pathItem.Operations() {
			_, err := readAccessMark(operation)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// NewContractRouter finds the operation of the contract that a request calls.
func NewContractRouter(contract []byte) (routers.Router, error) {
	loader := openapi3.NewLoader()

	loadedContract, err := loader.LoadFromData(contract)
	if err != nil {
		return nil, fmt.Errorf("parse the API contract: %w", err)
	}

	err = loadedContract.Validate(loader.Context)
	if err != nil {
		return nil, fmt.Errorf("validate the API contract: %w", err)
	}

	contractRouter, err := gorillamux.NewRouter(loadedContract)
	if err != nil {
		return nil, fmt.Errorf("build the router of the API contract: %w", err)
	}

	return contractRouter, nil
}
