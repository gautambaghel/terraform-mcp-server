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

// CreateRunTask creates run tasks in Terraform.
func CreateRunTask(logger *log.Logger) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool("create_run_task",
			mcp.WithDescription(`Creates a new Terraform run task in the specified organization.`),
			mcp.WithTitleAnnotation("Create a new Terraform run task"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithString("terraform_org_name",
				mcp.Required(),
				mcp.Description("The Terraform Cloud/Enterprise organization name"),
			),
			mcp.WithString("run_task_name",
				mcp.Required(),
				mcp.Description("The name of the run task to create. Can only contain letters, numbers, dashes and underscores."),
			),
			mcp.WithString("run_task_endpoint_url",
				mcp.Required(),
				mcp.Description("Run Tasks will POST to this URL. It must be a valid URL that the Terraform service can reach."),
			),
			mcp.WithString("run_task_hmac_key",
				mcp.Description("A secret key that may be required by the service to verify request authenticity."),
			),
			mcp.WithString("run_task_description",
				mcp.Description("A description for the run task."),
			),
			mcp.WithBoolean("run_task_global_scope",
				mcp.Description("Apply run task to all workspaces in the organization."),
				mcp.DefaultBool(false),
			),
			mcp.WithArray("run_task_global_run_stages",
				mcp.Description("Apply run task to all run stages in the organization. (Multiple values allowed)"),
				mcp.WithStringItems(mcp.Enum("pre_plan", "post_plan", "pre_apply", "post_apply")),
			),
			mcp.WithString("run_task_enforcement_level",
				mcp.Description("The enforcement level for the run task"),
				mcp.Enum("advisory", "mandatory"),
				mcp.DefaultString("advisory"),
			),
		),
		Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return createRunTaskHandler(ctx, req, logger)
		},
	}
}

func createRunTaskHandler(ctx context.Context, request mcp.CallToolRequest, logger *log.Logger) (*mcp.CallToolResult, error) {
	terraformOrgName, err := request.RequireString("terraform_org_name")
	if err != nil {
		return ToolError(logger, "missing required input: terraform_org_name", err)
	}
	terraformOrgName = strings.TrimSpace(terraformOrgName)

	runTaskName, err := request.RequireString("run_task_name")
	if err != nil {
		return ToolError(logger, "missing required input: run_task_name", err)
	}
	runTaskName = strings.TrimSpace(runTaskName)

	runTaskEndpointURL, err := request.RequireString("run_task_endpoint_url")
	if err != nil {
		return ToolError(logger, "missing required input: run_task_endpoint_url", err)
	}
	runTaskEndpointURL = strings.TrimSpace(runTaskEndpointURL)

	runTaskHMACKey := request.GetString("run_task_hmac_key", "")
	runTaskDescription := request.GetString("run_task_description", "")
	runTaskGlobalScope := request.GetBool("run_task_global_scope", false)
	runTaskGlobalRunStages := request.GetStringSlice("run_task_global_run_stages", nil)
	runTaskEnforcementLevel := request.GetString("run_task_enforcement_level", "")

	tfeClient, err := client.GetTfeClientFromContext(ctx, logger)
	if err != nil {
		return ToolError(logger, "failed to get Terraform client", err)
	}

	var stages *[]tfe.Stage
	if len(runTaskGlobalRunStages) > 0 {
		stgs := []tfe.Stage{}
		for _, s := range runTaskGlobalRunStages {
			stgs = append(stgs, tfe.Stage(strings.TrimSpace(s)))
		}
		stages = &stgs
	}

	var enforcementLevel *tfe.TaskEnforcementLevel
	if runTaskEnforcementLevel != "" {
		el := tfe.TaskEnforcementLevel(runTaskEnforcementLevel)
		enforcementLevel = &el
	}

	options := &tfe.RunTaskCreateOptions{
		Name:        runTaskName,
		URL:         runTaskEndpointURL,
		Category:    "task",
		Description: tfe.String(runTaskDescription),
		HMACKey:     tfe.String(runTaskHMACKey),
		Enabled:     tfe.Bool(true),
		Global: &tfe.GlobalRunTaskOptions{
			Enabled:          tfe.Bool(runTaskGlobalScope),
			Stages:           stages,
			EnforcementLevel: enforcementLevel,
		},
	}

	runTask, err := tfeClient.RunTasks.Create(ctx, terraformOrgName, *options)
	if err != nil {
		return ToolErrorf(logger, "created run task '%s' not found in org '%s': %v", runTaskName, terraformOrgName, err)
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
