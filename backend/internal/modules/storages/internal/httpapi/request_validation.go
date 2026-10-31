package httpapi

import (
	"fmt"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers/gorillamux"
	"github.com/gin-gonic/gin"

	"databasus-backend/internal/lib/apierror"
)

func NewRequestValidator(hooks *apierror.Hooks) MiddlewareFunc {
	contract, err := GetSwagger()
	if err != nil {
		panic(fmt.Errorf("load generated storages contract: %w", err))
	}

	router, err := gorillamux.NewRouter(contract)
	if err != nil {
		panic(fmt.Errorf("build storages contract router: %w", err))
	}

	return func(ctx *gin.Context) {
		route, pathParams, err := router.FindRoute(ctx.Request)
		if err != nil {
			hooks.HandleHandlerError(ctx, err)
			return
		}

		input := &openapi3filter.RequestValidationInput{
			Request:    ctx.Request,
			PathParams: pathParams,
			Route:      route,
			Options:    &openapi3filter.Options{AuthenticationFunc: openapi3filter.NoopAuthenticationFunc},
		}

		if err := openapi3filter.ValidateRequest(ctx.Request.Context(), input); err != nil {
			apierror.Write(
				ctx,
				apierror.NewInvalidFieldsProblem(
					[]apierror.FieldError{{Field: "request", Detail: "must match the API contract"}},
				),
			)
			return
		}

		ctx.Next()
	}
}
