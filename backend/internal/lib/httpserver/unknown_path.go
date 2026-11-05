package httpserver

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"databasus-backend/internal/lib/apierror"
)

// HandleUnknownPath answers an unknown API path with a Problem 404 and serves the frontend for every
// other path: a file when one exists, the single-page app's index.html otherwise.
func HandleUnknownPath(staticDirectory string) gin.HandlerFunc {
	return func(ginContext *gin.Context) {
		if strings.HasPrefix(ginContext.Request.URL.Path, "/api/") {
			apierror.Write(ginContext, &apierror.Problem{
				Code:   apierror.CodeNotFound,
				Detail: "no API operation matches the path",
				Errors: nil,
			})

			return
		}

		filePath := filepath.Join(staticDirectory, filepath.Clean("/"+ginContext.Request.URL.Path))

		if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
			ginContext.File(filePath)

			return
		}

		ginContext.File(filepath.Join(staticDirectory, "index.html"))
	}
}
