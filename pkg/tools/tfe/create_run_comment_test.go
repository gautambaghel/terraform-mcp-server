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
)

func TestCreateRunComment(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.ErrorLevel)

	t.Run("tool creation", func(t *testing.T) {
		tool := CreateRunComment(logger)

		assert.Equal(t, "create_run_task", tool.Tool.Name)
		assert.Contains(t, tool.Tool.Description, "Creates a new Terraform run comment")
		assert.NotNil(t, tool.Handler)

		assert.NotNil(t, tool.Tool.Annotations.ReadOnlyHint)
		assert.False(t, *tool.Tool.Annotations.ReadOnlyHint)
		assert.NotNil(t, tool.Tool.Annotations.DestructiveHint)
		assert.False(t, *tool.Tool.Annotations.DestructiveHint)

		assert.Contains(t, tool.Tool.InputSchema.Required, "run_id")
		assert.Contains(t, tool.Tool.InputSchema.Required, "run_comment_text")
	})

	t.Run("parameter validation via handler", func(t *testing.T) {
		tests := []struct {
			name     string
			params   map[string]interface{}
			errorMsg string
		}{
			{
				name: "missing run_id",
				params: map[string]interface{}{
					"run_comment_text": "hello",
				},
				errorMsg: "run_id",
			},
			{
				name: "missing run_comment_text",
				params: map[string]interface{}{
					"run_id": "run-123",
				},
				errorMsg: "runComment",
			},
			{
				name: "valid parameters but missing client in context",
				params: map[string]interface{}{
					"run_id":           "run-123",
					"run_comment_text": "comment text",
				},
				errorMsg: "failed to get Terraform client",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				req := mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Arguments: tt.params,
					},
				}

				res, err := createRunCommentHandler(context.Background(), req, logger)
				assert.NoError(t, err)
				assert.NotNil(t, res)
				assert.True(t, res.IsError)

				if assert.NotEmpty(t, res.Content) {
					contentText, ok := res.Content[0].(mcp.TextContent)
					if assert.True(t, ok) {
						assert.True(t, strings.Contains(contentText.Text, tt.errorMsg),
							"expected error containing '%s', got: %s", tt.errorMsg, contentText.Text)
					}
				}
			})
		}
	})
}
