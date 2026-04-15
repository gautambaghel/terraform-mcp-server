// Copyright IBM Corp. 2025
// SPDX-License-Identifier: MPL-2.0

package tools

import (
	"bytes"
	"context"
	"strings"

	"github.com/hashicorp/go-tfe"
	"github.com/hashicorp/jsonapi"
	"github.com/hashicorp/terraform-mcp-server/pkg/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	log "github.com/sirupsen/logrus"
)

// AttachRunTaskToWorkspaces attaches run tasks to workspaces in Terraform.
func AttachRunTaskToWorkspaces(logger *log.Logger) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool("attach_run_task_to_workspaces",
			mcp.WithDescription(`Attaches a Terraform run task to the specified workspaces.`),
			mcp.WithTitleAnnotation("Attach a Terraform run task to workspaces"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithString("workspace_id",
				mcp.Required(),
				mcp.Description("The ID of the workspace to attach the run task to."),
			),
			mcp.WithString("run_task_id",
				mcp.Required(),
				mcp.Description("The ID of the run task to attach."),
			),
			mcp.WithString("run_task_enforcement_level",
				mcp.Description("The enforcement level of the run task."),
				mcp.Enum("advisory", "mandatory"),
				mcp.DefaultString("advisory"),
			),
		),
		Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return AttachRunTaskToWorkspaceHandler(ctx, req, logger)
		},
	}
}

func AttachRunTaskToWorkspaceHandler(ctx context.Context, request mcp.CallToolRequest, logger *log.Logger) (*mcp.CallToolResult, error) {
	workspaceID, err := request.RequireString("workspace_id")
	if err != nil {
		return ToolError(logger, "missing required input: workspace_id", err)
	}
	workspaceID = strings.TrimSpace(workspaceID)

	runTaskId, err := request.RequireString("run_task_id")
	if err != nil {
		return ToolError(logger, "missing required input: run_task_id", err)
	}
	runTaskId = strings.TrimSpace(runTaskId)
	enforcementLevel := request.GetString("run_task_enforcement_level", "advisory")

	tfeClient, err := client.GetTfeClientFromContext(ctx, logger)
	if err != nil {
		return ToolError(logger, "failed to get Terraform client", err)
	}

	runTask, err := tfeClient.RunTasks.AttachToWorkspace(ctx, workspaceID, runTaskId, tfe.TaskEnforcementLevel(enforcementLevel))
	if err != nil {
		return ToolErrorf(logger, "error attaching run task '%s' to workspace '%s': %v", runTaskId, workspaceID, err)
	}

	var buf bytes.Buffer
	if err := jsonapi.MarshalPayload(&buf, runTask); err != nil {
		return ToolError(logger, "failed to marshal run task response", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(buf.String()),
		},
	}, nil
}
