package s3

import (
	"errors"
	"net/http"

	"github.com/minio/minio-go/v7"
)

// An error without an S3 answer is a network failure before the provider answered, which is as
// temporary as a busy node.
func isRetryableS3Error(err error) bool {
	if errorResponse, isResponseError := errors.AsType[minio.ErrorResponse](err); isResponseError {
		return errorResponse.StatusCode == http.StatusRequestTimeout ||
			errorResponse.StatusCode == http.StatusTooManyRequests ||
			errorResponse.StatusCode >= http.StatusInternalServerError
	}

	return true
}
