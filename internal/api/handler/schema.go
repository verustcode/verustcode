// Package handler provides HTTP handlers for the API.
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/verustcode/verustcode/internal/dsl"
)

// SchemaHandler handles schema-related API endpoints.
type SchemaHandler struct{}

// NewSchemaHandler creates a new SchemaHandler.
func NewSchemaHandler() *SchemaHandler {
	return &SchemaHandler{}
}

// GetSchema returns the default JSON schema.
// GET /api/v1/schemas/:name
// Note: Only "default" schema is supported since schemas are now embedded in code.
func (h *SchemaHandler) GetSchema(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "schema name is required"})
		return
	}

	// Only "default" schema is supported (embedded in code)
	if name != "default" && name != "default.json" {
		c.JSON(http.StatusNotFound, gin.H{"error": "schema not found, only 'default' schema is available"})
		return
	}

	schema := dsl.GetDefaultJSONSchema()
	c.JSON(http.StatusOK, schema)
}
