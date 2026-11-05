package httpserver

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// AllowAnyOrigin lets the frontend dev server on another port call the API. It belongs to
// development only.
func AllowAnyOrigin(exposedHeaders []string) gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders: []string{
			"Origin",
			"Content-Length",
			"Content-Type",
			"Authorization",
			"Accept",
			"Accept-Language",
			"Accept-Encoding",
			"Access-Control-Request-Method",
			"Access-Control-Request-Headers",
			"Access-Control-Allow-Methods",
			"Access-Control-Allow-Headers",
			"Access-Control-Allow-Origin",
		},
		ExposeHeaders:    exposedHeaders,
		AllowCredentials: true,
	})
}
