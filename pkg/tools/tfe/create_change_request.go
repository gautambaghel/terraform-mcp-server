// Copyright IBM Corp. 2026
// SPDX-License-Identifier: MPL-2.0

package tools

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-mcp-server/pkg/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	log "github.com/sirupsen/logrus"
)

type createChangeRequestPayload struct {
	Data createChangeRequestData `json:"data"`
}

type createChangeRequestData struct {
	Type       string                        `json:"type"`
	Attributes createChangeRequestAttributes `json:"attributes"`
}

type createChangeRequestAttributes struct {
	ActionType  string                          `json:"action_type"`
	ActionInput createChangeRequestActionInputs `json:"action_inputs"`
	TargetIDs   []string                        `json:"target_ids,omitempty"`
	Query       *createChangeRequestQuery       `json:"query,omitempty"`
}

type createChangeRequestActionInputs struct {
	Subject string `json:"subject"`
	Message string `json:"message"`
}

type createChangeRequestQuery struct {
	Type   string                      `json:"type"`
	Filter []createChangeRequestFilter `json:"filter"`
}

type createChangeRequestFilter struct {
	WorkspaceName createChangeRequestWorkspaceNameFilter `json:"workspace_name"`
}

type createChangeRequestWorkspaceNameFilter struct {
	Contains []string `json:"contains"`
}

// CreateChangeRequest creates Terraform workspace change requests through the Explorer bulk-actions API.
func CreateChangeRequest(logger *log.Logger) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool("create_change_request",
			mcp.WithDescription(`Creates a new Terraform change request in the specified organization using the Explorer bulk-actions API. You must provide either target_workspace_ids or workspace_name_contains.`),
			mcp.WithTitleAnnotation("Create a new Terraform change request"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithString("terraform_org_name",
				mcp.Required(),
				mcp.Description("The Terraform Cloud/Enterprise organization name"),
			),
			mcp.WithString("subject",
				mcp.Required(),
				mcp.Description("The change request subject line"),
			),
			mcp.WithString("message",
				mcp.Required(),
				mcp.Description("The change request message body. Terraform treats this as markdown."),
			),
			mcp.WithArray("target_workspace_ids",
				mcp.Description("Optional list of workspace IDs to target directly (e.g., ['ws-abc123']). Mutually exclusive with workspace_name_contains."),
				mcp.WithStringItems(),
			),
			mcp.WithArray("workspace_name_contains",
				mcp.Description("Optional list of substrings used to match workspace names via Explorer query filter (e.g., ['dev', 'staging']). Mutually exclusive with target_workspace_ids."),
				mcp.WithStringItems(),
			),
		),
		Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return createChangeRequestHandler(ctx, req, logger)
		},
	}
}

func createChangeRequestHandler(ctx context.Context, request mcp.CallToolRequest, logger *log.Logger) (*mcp.CallToolResult, error) {
	terraformOrgName, err := request.RequireString("terraform_org_name")
	if err != nil {
		return ToolError(logger, "missing required input: terraform_org_name", err)
	}
	terraformOrgName = strings.TrimSpace(terraformOrgName)

	subject, err := request.RequireString("subject")
	if err != nil {
		return ToolError(logger, "missing required input: subject", err)
	}
	subject = strings.TrimSpace(subject)

	message, err := request.RequireString("message")
	if err != nil {
		return ToolError(logger, "missing required input: message", err)
	}
	message = strings.TrimSpace(message)

	targetWorkspaceIDs := request.GetStringSlice("target_workspace_ids", nil)
	workspaceNameContains := request.GetStringSlice("workspace_name_contains", nil)

	payload, err := buildChangeRequestPayload(subject, message, targetWorkspaceIDs, workspaceNameContains)
	if err != nil {
		return ToolError(logger, "invalid change request inputs", err)
	}

	tfeClient, err := client.GetTfeClientFromContext(ctx, logger)
	if err != nil {
		return ToolError(logger, "failed to get Terraform client", err)
	}

	req, err := tfeClient.NewRequest("POST", fmt.Sprintf("organizations/%s/explorer/bulk-actions", terraformOrgName), payload)
	if err != nil {
		return ToolError(logger, "failed to prepare change request API call", err)
	}

	// explorer/bulk-actions is a plain JSON endpoint, not JSON:API;
	// override the Content-Type that NewRequest sets by default.
	req.Header.Set("Content-Type", "application/vnd.api+json")

	var buf bytes.Buffer
	if err := req.DoJSON(ctx, &buf); err != nil {
		return ToolError(logger, "failed to create change request", err)
	}

	return mcp.NewToolResultText(buf.String()), nil
}

func buildChangeRequestPayload(subject, message string, targetWorkspaceIDs, workspaceNameContains []string) (*createChangeRequestPayload, error) {
	trimmedSubject := strings.TrimSpace(subject)
	if trimmedSubject == "" {
		return nil, fmt.Errorf("subject cannot be empty")
	}

	trimmedMessage := strings.TrimSpace(message)
	if trimmedMessage == "" {
		return nil, fmt.Errorf("message cannot be empty")
	}

	trimmedTargetIDs := trimAndFilterEmpty(targetWorkspaceIDs)
	trimmedWorkspaceContains := trimAndFilterEmpty(workspaceNameContains)

	if len(trimmedTargetIDs) == 0 && len(trimmedWorkspaceContains) == 0 {
		return nil, fmt.Errorf("either target_workspace_ids or workspace_name_contains is required")
	}

	if len(trimmedTargetIDs) > 0 && len(trimmedWorkspaceContains) > 0 {
		return nil, fmt.Errorf("target_workspace_ids and workspace_name_contains cannot be used together")
	}

	payload := &createChangeRequestPayload{
		Data: createChangeRequestData{
			Type: "bulk_actions",
			Attributes: createChangeRequestAttributes{
				ActionType: "change_request",
				ActionInput: createChangeRequestActionInputs{
					Subject: trimmedSubject,
					Message: trimmedMessage,
				},
			},
		},
	}

	if len(trimmedTargetIDs) > 0 {
		payload.Data.Attributes.TargetIDs = trimmedTargetIDs
	} else {
		payload.Data.Attributes.Query = &createChangeRequestQuery{
			Type: "workspaces",
			Filter: []createChangeRequestFilter{
				{
					WorkspaceName: createChangeRequestWorkspaceNameFilter{
						Contains: trimmedWorkspaceContains,
					},
				},
			},
		}
	}

	log.Infof("Built change request payload: %+v", payload)
	return payload, nil
}

func trimAndFilterEmpty(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	trimmed := make([]string, 0, len(values))
	for _, value := range values {
		v := strings.TrimSpace(value)
		if v != "" {
			trimmed = append(trimmed, v)
		}
	}

	if len(trimmed) == 0 {
		return nil
	}

	return trimmed
}
