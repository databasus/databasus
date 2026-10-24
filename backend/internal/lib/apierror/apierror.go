// Package apierror answers the errors of contract operations with the Problem body that
// backend/api/openapi.yaml declares.
package apierror

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Code mirrors the ProblemCode enum of the contract; a test keeps the two equal.
type Code string

const (
	CodeInvalidRequest Code = "invalid_request"
	CodeNotFound       Code = "not_found"
	CodeInternal       Code = "internal"
)

const contentType = "application/problem+json"

type problemBody struct {
	Type   string       `json:"type"`
	Title  string       `json:"title"`
	Status int          `json:"status"`
	Detail string       `json:"detail"`
	Code   Code         `json:"code"`
	Errors []FieldError `json:"errors,omitzero"`
}

// The detail of every internal error is the same, so a failure never reveals SQL, paths or secrets.
const internalErrorDetail = "an internal error occurred"

// NewInternalProblem is the problem of every failure the client did not cause.
func NewInternalProblem() *Problem {
	return &Problem{Code: CodeInternal, Detail: internalErrorDetail, Errors: nil}
}

// Write answers the problem and aborts the request, so no later handler writes a second body.
func Write(ctx *gin.Context, problem *Problem) {
	status, title := describeCode(problem.Code)

	ctx.Render(status, problemRender{body: problemBody{
		Type:   "urn:databasus:problem:" + string(problem.Code),
		Title:  title,
		Status: status,
		Detail: problem.Detail,
		Code:   problem.Code,
		Errors: problem.Errors,
	}})
	ctx.Abort()
}

func describeCode(code Code) (int, string) {
	switch code {
	case CodeInvalidRequest:
		return http.StatusBadRequest, "Invalid request"
	case CodeNotFound:
		return http.StatusNotFound, "Not found"
	case CodeInternal:
		return http.StatusInternalServerError, "Internal error"
	}

	return http.StatusInternalServerError, "Internal error"
}

type problemRender struct {
	body problemBody
}

func (r problemRender) Render(writer http.ResponseWriter) error {
	r.WriteContentType(writer)

	return json.NewEncoder(writer).Encode(r.body)
}

func (r problemRender) WriteContentType(writer http.ResponseWriter) {
	writer.Header().Set("Content-Type", contentType)
}
