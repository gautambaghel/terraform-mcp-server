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

func TestCreateRunTask(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.ErrorLevel) // Reduce noise in tests

	t.Run("tool creation", func(t *testing.T) {
		tool := CreateRunTask(logger)

		assert.Equal(t, "create_run_task", tool.Tool.Name)
		assert.Contains(t, tool.Tool.Description, "Creates a new Terraform run task")
		assert.NotNil(t, tool.Handler)

		assert.NotNil(t, tool.Tool.Annotations.DestructiveHint)
		assert.False(t, *tool.Tool.Annotations.DestructiveHint)
		assert.NotNil(t, tool.Tool.Annotations.ReadOnlyHint)
		assert.False(t, *tool.Tool.Annotations.ReadOnlyHint)

		// Check that required parameters are defined
		assert.Contains(t, tool.Tool.InputSchema.Required, "terraform_org_name")
		assert.Contains(t, tool.Tool.InputSchema.Required, "run_task_name")
		assert.Contains(t, tool.Tool.InputSchema.Required, "run_task_endpoint_url")
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
					"terraform_org_name":    "test-org",
					"run_task_name":         "test-task",
					"run_task_endpoint_url": "https://example.com/run-task",
				},
				expectError: true, // Will error on missing client after validations
				errorMsg:    "failed to get Terraform client",
			},
			{
				name: "missing org name",
				params: map[string]interface{}{
					"run_task_name":         "test-task",
					"run_task_endpoint_url": "https://example.com/run-task",
				},
				expectError: true,
				errorMsg:    "terraform_org_name",
			},
			{
				name: "missing task name",
				params: map[string]interface{}{
					"terraform_org_name":    "test-org",
					"run_task_endpoint_url": "https://example.com/run-task",
				},
				expectError: true,
				errorMsg:    "run_task_name",
			},
			{
				name: "missing endpoint url",
				params: map[string]interface{}{
					"terraform_org_name": "test-org",
					"run_task_name":      "test-task",
				},
				expectError: true,
				errorMsg:    "run_task_endpoint_url",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				req := mcp.CallToolRequest{
					Params: mcp.CallToolParams{
						Arguments: tt.params,
					},
				}
				res, err := createRunTaskHandler(context.Background(), req, logger)
				
				if tt.expectError {
					// We expect either a go error or a ToolError wrapped inside res
					if err != nil {
						assert.True(t, strings.Contains(err.Error(), tt.errorMsg),
							"expected error containing '%s', got: %v", tt.errorMsg, err)
					} else {
						assert.NotNil(t, res)
						assert.True(t, res.IsError, "expected ToolResultError, got successful result")
						
						// Verify error message in the content text
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
