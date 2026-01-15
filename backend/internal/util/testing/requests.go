package testing

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

type RequestOptions struct {
	Method         string
	URL            string
	Body           interface{}
	Headers        map[string]string
	AuthToken      string
	ExpectedStatus int
}

type TestResponse struct {
	StatusCode int
	Body       []byte
	Headers    http.Header
}

func MakeGetRequest(
	t *testing.T,
	router *gin.Engine,
	url, authToken string,
	expectedStatus int,
) *TestResponse {
	return makeAuthenticatedRequest(t, router, "GET", url, authToken, nil, expectedStatus)
}

func MakeGetRequestAndUnmarshal(
	t *testing.T,
	router *gin.Engine,
	url, authToken string,
	expectedStatus int,
	responseStruct interface{},
) *TestResponse {
	return makeAuthenticatedRequestAndUnmarshal(
		t,
		router,
		"GET",
		url,
		authToken,
		nil,
		expectedStatus,
		responseStruct,
	)
}

func MakePostRequest(
	t *testing.T,
	router *gin.Engine,
	url, authToken string,
	body interface{},
	expectedStatus int,
) *TestResponse {
	return makeAuthenticatedRequest(t, router, "POST", url, authToken, body, expectedStatus)
}

func MakePostRequestAndUnmarshal(
	t *testing.T,
	router *gin.Engine,
	url, authToken string,
	body interface{},
	expectedStatus int,
	responseStruct interface{},
) *TestResponse {
	return makeAuthenticatedRequestAndUnmarshal(
		t,
		router,
		"POST",
		url,
		authToken,
		body,
		expectedStatus,
		responseStruct,
	)
}

func MakePutRequest(
	t *testing.T,
	router *gin.Engine,
	url, authToken string,
	body interface{},
	expectedStatus int,
) *TestResponse {
	return makeAuthenticatedRequest(t, router, "PUT", url, authToken, body, expectedStatus)
}

func MakePutRequestAndUnmarshal(
	t *testing.T,
	router *gin.Engine,
	url, authToken string,
	body interface{},
	expectedStatus int,
	responseStruct interface{},
) *TestResponse {
	return makeAuthenticatedRequestAndUnmarshal(
		t,
		router,
		"PUT",
		url,
		authToken,
		body,
		expectedStatus,
		responseStruct,
	)
}

func MakeDeleteRequest(
	t *testing.T,
	router *gin.Engine,
	url, authToken string,
	expectedStatus int,
) *TestResponse {
	return makeAuthenticatedRequest(t, router, "DELETE", url, authToken, nil, expectedStatus)
}

func MakeRequest(t *testing.T, router *gin.Engine, options RequestOptions) *TestResponse {
	var requestBody *bytes.Buffer

	if options.Body != nil {
		bodyJSON, err := json.Marshal(options.Body)
		assert.NoError(t, err, "Failed to marshal request body")
		requestBody = bytes.NewBuffer(bodyJSON)
	} else {
		requestBody = bytes.NewBuffer(nil)
	}

	req, err := http.NewRequest(options.Method, options.URL, requestBody)
	assert.NoError(t, err, "Failed to create HTTP request")

	if options.Body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if options.AuthToken != "" {
		req.Header.Set("Authorization", options.AuthToken)
	}

	for key, value := range options.Headers {
		req.Header.Set(key, value)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if options.ExpectedStatus != 0 {
		if !assert.Equal(t, options.ExpectedStatus, w.Code, "Unexpected status code") {
			log.Printf("Unexpected status code with response: %s", w.Body.String())
		}
	}

	return &TestResponse{
		StatusCode: w.Code,
		Body:       w.Body.Bytes(),
		Headers:    w.Header(),
	}
}

func makeRequestAndUnmarshal(
	t *testing.T,
	router *gin.Engine,
	options RequestOptions,
	responseStruct interface{},
) *TestResponse {
	response := MakeRequest(t, router, options)

	if responseStruct != nil && len(response.Body) > 0 {
		err := json.Unmarshal(response.Body, responseStruct)
		assert.NoError(t, err, "Failed to unmarshal response body")
	}

	return response
}

func makeAuthenticatedRequest(
	t *testing.T,
	router *gin.Engine,
	method, url, authToken string,
	body interface{},
	expectedStatus int,
) *TestResponse {
	return MakeRequest(t, router, RequestOptions{
		Method:         method,
		URL:            url,
		Body:           body,
		AuthToken:      authToken,
		ExpectedStatus: expectedStatus,
	})
}

func makeAuthenticatedRequestAndUnmarshal(
	t *testing.T,
	router *gin.Engine,
	method, url, authToken string,
	body interface{},
	expectedStatus int,
	responseStruct interface{},
) *TestResponse {
	return makeRequestAndUnmarshal(t, router, RequestOptions{
		Method:         method,
		URL:            url,
		Body:           body,
		AuthToken:      authToken,
		ExpectedStatus: expectedStatus,
	}, responseStruct)
}
