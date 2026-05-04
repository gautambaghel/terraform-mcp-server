// Copyright IBM Corp. 2026
// SPDX-License-Identifier: MPL-2.0

package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateChangeRequest(t *testing.T) {
	logger := log.New()
	logger.SetLevel(log.ErrorLevel)

	t.Run("tool creation", func(t *testing.T) {
		tool := CreateChangeRequest(logger)

		assert.Equal(t, "create_change_request", tool.Tool.Name)
		assert.Contains(t, tool.Tool.Description, "Creates a new Terraform change request")
		assert.NotNil(t, tool.Handler)

		assert.NotNil(t, tool.Tool.Annotations.ReadOnlyHint)
		assert.False(t, *tool.Tool.Annotations.ReadOnlyHint)
		assert.NotNil(t, tool.Tool.Annotations.DestructiveHint)
		assert.False(t, *tool.Tool.Annotations.DestructiveHint)

		assert.Contains(t, tool.Tool.InputSchema.Required, "terraform_org_name")
		assert.Contains(t, tool.Tool.InputSchema.Required, "subject")
		assert.Contains(t, tool.Tool.InputSchema.Required, "message")

		assert.NotNil(t, tool.Tool.InputSchema.Properties["target_workspace_ids"])
		assert.NotNil(t, tool.Tool.InputSchema.Properties["workspace_name_contains"])
	})

	t.Run("handler parameter validation", func(t *testing.T) {
		tests := []struct {
			name     string
			params   map[string]interface{}
			errorMsg string
		}{
			{
				name: "missing org",
				params: map[string]interface{}{
					"subject":                 "subj",
					"message":                 "msg",
					"workspace_name_contains": []interface{}{"dev"},
				},
				errorMsg: "terraform_org_name",
			},
			{
				name: "missing subject",
				params: map[string]interface{}{
					"terraform_org_name":      "org",
					"message":                 "msg",
					"workspace_name_contains": []interface{}{"dev"},
				},
				errorMsg: "subject",
			},
			{
				name: "missing message",
				params: map[string]interface{}{
					"terraform_org_name":      "org",
					"subject":                 "subj",
					"workspace_name_contains": []interface{}{"dev"},
				},
				errorMsg: "message",
			},
			{
				name: "missing targeting arguments",
				params: map[string]interface{}{
					"terraform_org_name": "org",
					"subject":            "subj",
					"message":            "msg",
				},
				errorMsg: "either target_workspace_ids or workspace_name_contains is required",
			},
			{
				name: "valid shape but missing tfe client",
				params: map[string]interface{}{
					"terraform_org_name":      "org",
					"subject":                 "subj",
					"message":                 "msg",
					"workspace_name_contains": []interface{}{"dev"},
				},
				errorMsg: "failed to get Terraform client",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				req := mcp.CallToolRequest{
					Params: mcp.CallToolParams{Arguments: tt.params},
				}

				res, err := createChangeRequestHandler(context.Background(), req, logger)
				assert.NoError(t, err)
				require.NotNil(t, res)
				assert.True(t, res.IsError)

				require.NotEmpty(t, res.Content)
				contentText, ok := res.Content[0].(mcp.TextContent)
				require.True(t, ok)
				assert.Contains(t, contentText.Text, tt.errorMsg)
			})
		}
	})
}

func TestBuildChangeRequestPayload(t *testing.T) {
	t.Run("with target workspace ids", func(t *testing.T) {
		payload, err := buildChangeRequestPayload(
			"  Subject  ",
			"  Message  ",
			[]string{" ws-1 ", "", "ws-2"},
			nil,
		)
		require.NoError(t, err)
		require.NotNil(t, payload)

		assert.Equal(t, "bulk_actions", payload.Data.Type)
		assert.Equal(t, "change_request", payload.Data.Attributes.ActionType)
		assert.Equal(t, "Subject", payload.Data.Attributes.ActionInput.Subject)
		assert.Equal(t, "Message", payload.Data.Attributes.ActionInput.Message)
		assert.Equal(t, []string{"ws-1", "ws-2"}, payload.Data.Attributes.TargetIDs)
		assert.Nil(t, payload.Data.Attributes.Query)
	})

	t.Run("with workspace name contains query", func(t *testing.T) {
		payload, err := buildChangeRequestPayload(
			"Subject",
			"Message",
			nil,
			[]string{" dev ", "staging", ""},
		)
		require.NoError(t, err)
		require.NotNil(t, payload)

		assert.Nil(t, payload.Data.Attributes.TargetIDs)
		require.NotNil(t, payload.Data.Attributes.Query)
		assert.Equal(t, "workspaces", payload.Data.Attributes.Query.Type)
		require.Len(t, payload.Data.Attributes.Query.Filter, 1)
		assert.Equal(t, []string{"dev", "staging"}, payload.Data.Attributes.Query.Filter[0].WorkspaceName.Contains)
	})

	t.Run("invalid when both targeting methods are set", func(t *testing.T) {
		payload, err := buildChangeRequestPayload(
			"Subject",
			"Message",
			[]string{"ws-1"},
			[]string{"dev"},
		)
		assert.Nil(t, payload)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be used together")
	})

	t.Run("invalid when no targeting method is set", func(t *testing.T) {
		payload, err := buildChangeRequestPayload("Subject", "Message", nil, nil)
		assert.Nil(t, payload)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "either target_workspace_ids or workspace_name_contains is required")
	})

	t.Run("invalid when subject or message empty", func(t *testing.T) {
		_, err := buildChangeRequestPayload(" ", "Message", []string{"ws-1"}, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "subject cannot be empty")

		_, err = buildChangeRequestPayload("Subject", " ", []string{"ws-1"}, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "message cannot be empty")
	})

	t.Run("payload serializes to expected json keys", func(t *testing.T) {
		payload, err := buildChangeRequestPayload("Subject", "Message", []string{"ws-1"}, nil)
		require.NoError(t, err)

		b, err := json.Marshal(payload)
		require.NoError(t, err)
		jsonText := string(b)

		for _, key := range []string{"\"action_type\"", "\"action_inputs\"", "\"target_ids\"", "\"subject\"", "\"message\""} {
			assert.True(t, strings.Contains(jsonText, key), "expected key %s in %s", key, jsonText)
		}
	})
}
