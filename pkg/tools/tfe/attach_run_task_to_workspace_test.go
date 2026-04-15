// Copyright IBM Corp. 2025
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

func TestAttachRunTaskToWorkspaces(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.ErrorLevel) // Reduce noise in tests

	t.Run("tool creation", func(t *testing.T) {
		tool := AttachRunTaskToWorkspaces(logger)

		assert.Equal(t, "attach_run_task_to_workspaces", tool.Tool.Name)
		assert.Contains(t, tool.Tool.Description, "Attaches a Terraform run task to the specified workspaces")
		assert.NotNil(t, tool.Handler)

		assert.NotNil(t, tool.Tool.Annotations.DestructiveHint)
		assert.False(t, *tool.Tool.Annotations.DestructiveHint)
		assert.NotNil(t, tool.Tool.Annotations.ReadOnlyHint)
		assert.False(t, *tool.Tool.Annotations.ReadOnlyHint)

		// Check that required parameters are defined
		assert.Contains(t, tool.Tool.InputSchema.Required, "workspace_id")
		assert.Contains(t, tool.Tool.InputSchema.Required, "run_task_id")
	})

	t.Run("parameter validation via handler", func(t *testing.T) {
		tests := []struct {
			name        string
			params      map[string]interface{}
			expectError bool
			errorMsg    string
		}{
			{
				name: "valid minimal parameters",
				params: map[string]interface{}{
					"workspace_id": "ws-123456",
					"run_task_id":  "task-7890",
				},
				expectError: true, // Will error on missing client after validations
				errorMsg:    "failed to get Terraform client",
			},
			{
				name: "missing workspace id",
				params: map[string]interface{}{
					"run_task_id":  "task-7890",
				},
				expectError: true,
				errorMsg:    "workspace_id",
			},
			{
				name: "missing task id",
				params: map[string]interface{}{
					"workspace_id": "ws-123456",
				},
				expectError: true,
				errorMsg:    "run_task_id",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				req := mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Arguments: tt.params,
					},
				}
				res, err := AttachRunTaskToWorkspaceHandler(context.Background(), req, logger)
				
				if tt.expectError {
					if err != nil {
						assert.True(t, strings.Contains(err.Error(), tt.errorMsg),
							"expected error containing '%s', got: %v", tt.errorMsg, err)
					} else {
						assert.NotNil(t, res)
						assert.True(t, res.IsError, "expected ToolResultError, got successful result")
						
						if len(res.Content) > 0 {
							contentTxt, ok := res.Content[0].(mcp.TextContent)
							if ok {
								assert.True(t, strings.Contains(contentTxt.Text, tt.errorMsg),
								"expected error containing '%s', got: %v", tt.errorMsg, contentTxt.Text)
							}
						}
					}
				} else {
					assert.NoError(t, err)
					if res != nil {
						assert.False(t, res.IsError)
					}
				}
			})
		}
	})
}
