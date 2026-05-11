// Copyright IBM Corp. 2026
// SPDX-License-Identifier: MPL-2.0

package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateAnalyzerSummary(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.ErrorLevel)

	t.Run("tool creation", func(t *testing.T) {
		tool := CreateAnalyzerSummary(logger)

		assert.Equal(t, "create_analyzer_summary", tool.Tool.Name)
		assert.Contains(t, tool.Tool.Description, "Creates an analyzer summary")
		assert.NotNil(t, tool.Handler)

		assert.NotNil(t, tool.Tool.Annotations.ReadOnlyHint)
		assert.False(t, *tool.Tool.Annotations.ReadOnlyHint)
		assert.NotNil(t, tool.Tool.Annotations.DestructiveHint)
		assert.False(t, *tool.Tool.Annotations.DestructiveHint)

		assert.Contains(t, tool.Tool.InputSchema.Required, "run_id")
		assert.Contains(t, tool.Tool.InputSchema.Required, "status")
		assert.Contains(t, tool.Tool.InputSchema.Required, "summary")
	})

	t.Run("handler parameter validation", func(t *testing.T) {
		tests := []struct {
			name     string
			params   map[string]interface{}
			errorMsg string
		}{
			{
				name: "missing run_id",
				params: map[string]interface{}{
					"status":  "succeeded",
					"summary": "example summary",
				},
				errorMsg: "run_id",
			},
			{
				name: "missing status",
				params: map[string]interface{}{
					"run_id":  "run-123",
					"summary": "example summary",
				},
				errorMsg: "status",
			},
			{
				name: "missing summary",
				params: map[string]interface{}{
					"run_id": "run-123",
					"status": "succeeded",
				},
				errorMsg: "summary",
			},
			{
				name: "valid shape but missing tfe client",
				params: map[string]interface{}{
					"run_id":  "run-123",
					"status":  "succeeded",
					"summary": "example summary",
				},
				errorMsg: "failed to get Terraform client",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				req := mcp.CallToolRequest{
					Params: mcp.CallToolParams{Arguments: tt.params},
				}

				res, err := createAnalyzerSummaryHandler(context.Background(), req, logger)
				assert.NoError(t, err)
				require.NotNil(t, res)
				assert.True(t, res.IsError)

				require.NotEmpty(t, res.Content)
				contentText, ok := res.Content[0].(mcp.TextContent)
				require.True(t, ok)
				assert.True(t, strings.Contains(contentText.Text, tt.errorMsg),
					"expected error containing '%s', got: %s", tt.errorMsg, contentText.Text)
			})
		}
	})
}
