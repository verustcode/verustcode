package llm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ====================
// Tests for NewSchemaGenerator
// ====================

func TestNewSchemaGenerator(t *testing.T) {
	gen := NewSchemaGenerator()
	assert.NotNil(t, gen)
}

// ====================
// Tests for SchemaGenerator.Generate
// ====================

func TestSchemaGenerator_Generate(t *testing.T) {
	gen := NewSchemaGenerator()

	t.Run("nil value", func(t *testing.T) {
		_, err := gen.Generate(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nil")
	})

	t.Run("string type", func(t *testing.T) {
		schema, err := gen.Generate("")
		require.NoError(t, err)
		assert.Equal(t, "string", schema["type"])
	})

	t.Run("int type", func(t *testing.T) {
		schema, err := gen.Generate(0)
		require.NoError(t, err)
		assert.Equal(t, "integer", schema["type"])
	})

	t.Run("float type", func(t *testing.T) {
		schema, err := gen.Generate(0.0)
		require.NoError(t, err)
		assert.Equal(t, "number", schema["type"])
	})

	t.Run("bool type", func(t *testing.T) {
		schema, err := gen.Generate(false)
		require.NoError(t, err)
		assert.Equal(t, "boolean", schema["type"])
	})

	t.Run("slice type", func(t *testing.T) {
		schema, err := gen.Generate([]string{})
		require.NoError(t, err)
		assert.Equal(t, "array", schema["type"])
		assert.NotNil(t, schema["items"])
	})

	t.Run("map type", func(t *testing.T) {
		schema, err := gen.Generate(map[string]interface{}{})
		require.NoError(t, err)
		assert.Equal(t, "object", schema["type"])
	})

	t.Run("struct type", func(t *testing.T) {
		type TestStruct struct {
			Name  string `json:"name"`
			Value int    `json:"value"`
		}
		schema, err := gen.Generate(TestStruct{})
		require.NoError(t, err)
		assert.Equal(t, "object", schema["type"])
		assert.NotNil(t, schema["properties"])
		props := schema["properties"].(map[string]interface{})
		assert.Equal(t, "string", props["name"].(map[string]interface{})["type"])
		assert.Equal(t, "integer", props["value"].(map[string]interface{})["type"])
	})

	t.Run("pointer type", func(t *testing.T) {
		type TestStruct struct {
			Name string `json:"name"`
		}
		schema, err := gen.Generate((*TestStruct)(nil))
		require.NoError(t, err)
		assert.Equal(t, "object", schema["type"])
	})

	t.Run("struct with omitempty", func(t *testing.T) {
		type TestStruct struct {
			Name  string `json:"name"`
			Value int    `json:"value,omitempty"`
		}
		schema, err := gen.Generate(TestStruct{})
		require.NoError(t, err)
		required := schema["required"].([]string)
		assert.Contains(t, required, "name")
		assert.NotContains(t, required, "value")
	})

	t.Run("struct with json tag -", func(t *testing.T) {
		type TestStruct struct {
			Name  string `json:"name"`
			Hidden string `json:"-"`
		}
		schema, err := gen.Generate(TestStruct{})
		require.NoError(t, err)
		props := schema["properties"].(map[string]interface{})
		assert.Contains(t, props, "name")
		assert.NotContains(t, props, "Hidden")
	})
}

// ====================
// Tests for ToJSONString
// ====================

func TestSchemaGenerator_ToJSONString(t *testing.T) {
	gen := NewSchemaGenerator()

	t.Run("valid schema", func(t *testing.T) {
		schema := map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type": "string",
				},
			},
		}
		jsonStr, err := gen.ToJSONString(schema)
		require.NoError(t, err)
		assert.Contains(t, jsonStr, "type")
		assert.Contains(t, jsonStr, "object")
		assert.Contains(t, jsonStr, "properties")
		assert.Contains(t, jsonStr, "name")
		assert.Contains(t, jsonStr, "string")
	})

	t.Run("invalid schema", func(t *testing.T) {
		// Create a schema with a channel (cannot be marshaled)
		schema := map[string]interface{}{
			"invalid": make(chan int),
		}
		_, err := gen.ToJSONString(schema)
		assert.Error(t, err)
	})
}

// ====================
// Tests for BuildSchemaPrompt
// ====================

func TestBuildSchemaPrompt(t *testing.T) {
	t.Run("nil schema", func(t *testing.T) {
		result := BuildSchemaPrompt(nil)
		assert.Empty(t, result)
	})

	t.Run("with schema", func(t *testing.T) {
		schema := &ResponseSchema{
			Name:        "test",
			Description: "test description",
			Schema: map[string]interface{}{
				"type": "object",
			},
		}
		result := BuildSchemaPrompt(schema)
		assert.Contains(t, result, "Output Format")
		assert.Contains(t, result, "test description")
		assert.Contains(t, result, "JSON format")
		assert.Contains(t, result, "type")
		assert.Contains(t, result, "object")
	})

	t.Run("strict schema", func(t *testing.T) {
		schema := &ResponseSchema{
			Name:   "test",
			Schema: map[string]interface{}{"type": "object"},
			Strict: true,
		}
		result := BuildSchemaPrompt(schema)
		assert.Contains(t, result, "MUST be valid JSON")
		assert.Contains(t, result, "strictly follows")
	})

	t.Run("non-strict schema", func(t *testing.T) {
		schema := &ResponseSchema{
			Name:   "test",
			Schema: map[string]interface{}{"type": "object"},
			Strict: false,
		}
		result := BuildSchemaPrompt(schema)
		assert.Contains(t, result, "ensure your response")
		assert.NotContains(t, result, "MUST")
	})
}

// ====================
// Tests for ExtractJSON
// ====================

func TestExtractJSON(t *testing.T) {
	t.Run("valid JSON object", func(t *testing.T) {
		content := `some text before {"name": "test", "value": 42} some text after`
		jsonStr, err := ExtractJSON(content)
		require.NoError(t, err)
		assert.Equal(t, `{"name": "test", "value": 42}`, jsonStr)
	})

	t.Run("valid JSON array", func(t *testing.T) {
		content := `before [1, 2, 3] after`
		jsonStr, err := ExtractJSON(content)
		require.NoError(t, err)
		assert.Equal(t, `[1, 2, 3]`, jsonStr)
	})

	t.Run("no JSON found", func(t *testing.T) {
		content := "no json here"
		_, err := ExtractJSON(content)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no valid JSON")
	})

	t.Run("only opening brace", func(t *testing.T) {
		content := "only { here"
		_, err := ExtractJSON(content)
		assert.Error(t, err)
	})

	t.Run("only closing brace", func(t *testing.T) {
		content := "only } here"
		_, err := ExtractJSON(content)
		assert.Error(t, err)
	})

	t.Run("nested JSON", func(t *testing.T) {
		content := `text {"outer": {"inner": "value"}} more text`
		jsonStr, err := ExtractJSON(content)
		require.NoError(t, err)
		assert.Contains(t, jsonStr, "outer")
		assert.Contains(t, jsonStr, "inner")
	})
}

// ====================
// Tests for ParseResponseJSON
// ====================

func TestParseResponseJSON(t *testing.T) {
	t.Run("valid JSON object", func(t *testing.T) {
		type TestStruct struct {
			Name  string `json:"name"`
			Value int    `json:"value"`
		}
		var result TestStruct

		content := `some text {"name": "test", "value": 42} more text`
		err := ParseResponseJSON(content, &result)
		require.NoError(t, err)
		assert.Equal(t, "test", result.Name)
		assert.Equal(t, 42, result.Value)
	})

	t.Run("valid JSON array", func(t *testing.T) {
		var result []int
		content := `before [1, 2, 3] after`
		err := ParseResponseJSON(content, &result)
		require.NoError(t, err)
		assert.Equal(t, []int{1, 2, 3}, result)
	})

	t.Run("no JSON found", func(t *testing.T) {
		var result map[string]interface{}
		content := "no json"
		err := ParseResponseJSON(content, &result)
		assert.Error(t, err)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		var result map[string]interface{}
		content := `text {invalid json} text`
		err := ParseResponseJSON(content, &result)
		assert.Error(t, err)
	})

	t.Run("type mismatch", func(t *testing.T) {
		var result int
		content := `{"name": "test"}`
		err := ParseResponseJSON(content, &result)
		assert.Error(t, err)
	})
}

// ====================
// Tests for MarkdownOutputPrompt
// ====================

func TestMarkdownOutputPrompt(t *testing.T) {
	prompt := MarkdownOutputPrompt()
	assert.Contains(t, prompt, "Output Format")
	assert.Contains(t, prompt, "Markdown format")
	assert.Contains(t, prompt, "Markdown")
}

