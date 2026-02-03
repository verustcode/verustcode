// Package admin provides embedded frontend assets for the admin console.
package admin

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed dist/*
var distFS embed.FS

// RegisterRoutes registers the admin console static file routes
func RegisterRoutes(r *gin.Engine) error {
	// Get the dist subdirectory
	subFS, err := fs.Sub(distFS, "dist")
	if err != nil {
		return err
	}

	httpFS := http.FS(subFS)

	// Read index.html content to serve it directly
	// This avoids any potential redirect issues with FileFromFS/http.FileServer
	indexContent, err := fs.ReadFile(subFS, "index.html")
	if err != nil {
		return err
	}

	// Serve static assets (js, css, images, etc.) under /admin/assets
	assetsFS, err := fs.Sub(distFS, "dist/assets")
	if err == nil {
		r.StaticFS("/admin/assets", http.FS(assetsFS))
	}

	// Serve favicon and other root static files explicitly
	r.GET("/admin/logo.svg", func(c *gin.Context) {
		c.FileFromFS("logo.svg", httpFS)
	})
	r.GET("/admin/vite.svg", func(c *gin.Context) {
		c.FileFromFS("vite.svg", httpFS)
	})

	// Serve index.html for the root admin path
	r.GET("/admin", func(c *gin.Context) {
		c.Data(200, "text/html; charset=utf-8", indexContent)
	})

	// Handle SPA routing - all /admin/* routes serve index.html
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		// Check if request is for admin console
		if strings.HasPrefix(path, "/admin") {
			// Skip if it's a request for static assets (already handled)
			if strings.HasPrefix(path, "/admin/assets/") {
				c.JSON(http.StatusNotFound, gin.H{
					"code":    "not_found",
					"message": "Asset not found",
				})
				return
			}
			// Serve index.html content directly for SPA routing
			c.Data(200, "text/html; charset=utf-8", indexContent)
			return
		}

		// Otherwise, return 404
		c.JSON(http.StatusNotFound, gin.H{
			"code":    "not_found",
			"message": "Resource not found",
		})
	})

	return nil
}

// HasEmbeddedAssets returns true if the dist directory has embedded assets
func HasEmbeddedAssets() bool {
	entries, err := distFS.ReadDir("dist")
	if err != nil {
		return false
	}
	return len(entries) > 0
}
