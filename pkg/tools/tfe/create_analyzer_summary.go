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

type createAnalyzerSummaryRequest struct {
	Status  string `json:"status"`
	Summary string `json:"summary"`
}

// CreateAnalyzerSummary creates an analyzer summary for a Terraform run.
func CreateAnalyzerSummary(logger *log.Logger) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool("create_analyzer_summary",
			mcp.WithDescription(`Creates an analyzer summary for the specified Terraform run.`),
			mcp.WithTitleAnnotation("Create analyzer summary for a run"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithString("run_id",
				mcp.Required(),
				mcp.Description("The Terraform Cloud/Enterprise run id to attach the analyzer summary to."),
			),
			mcp.WithString("status",
				mcp.Required(),
				mcp.Description("Analyzer result status (for example: succeeded, failed)."),
			),
			mcp.WithString("summary",
				mcp.Required(),
				mcp.Description("Human-readable analyzer summary for the run."),
			),
		),
		Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return createAnalyzerSummaryHandler(ctx, req, logger)
		},
	}
}

func createAnalyzerSummaryHandler(ctx context.Context, request mcp.CallToolRequest, logger *log.Logger) (*mcp.CallToolResult, error) {
	runID, err := request.RequireString("run_id")
	if err != nil {
		return ToolError(logger, "missing required input: run_id", err)
	}
	runID = strings.TrimSpace(runID)

	status, err := request.RequireString("status")
	if err != nil {
		return ToolError(logger, "missing required input: status", err)
	}
	status = strings.TrimSpace(status)

	summary, err := request.RequireString("summary")
	if err != nil {
		return ToolError(logger, "missing required input: summary", err)
	}
	summary = strings.TrimSpace(summary)

	tfeClient, err := client.GetTfeClientFromContext(ctx, logger)
	if err != nil {
		return ToolError(logger, "failed to get Terraform client", err)
	}

	payload := createAnalyzerSummaryRequest{
		Status:  status,
		Summary: summary,
	}

	req, err := tfeClient.NewRequest("POST", fmt.Sprintf("runs/%s/analyzer-summary", runID), &payload)
	if err != nil {
		return ToolError(logger, "failed to prepare analyzer summary API call", err)
	}

	// runs/:id/analyzer-summary expects a plain JSON body.
	req.Header.Set("Content-Type", "application/json")

	var buf bytes.Buffer
	if err := req.DoJSON(ctx, &buf); err != nil {
		return ToolErrorf(logger, "failed to create analyzer summary for run '%s': %v", runID, err)
	}

	if strings.TrimSpace(buf.String()) == "" {
		return mcp.NewToolResultText(fmt.Sprintf("{\"success\":true,\"run_id\":\"%s\"}", runID)), nil
	}

	return mcp.NewToolResultText(buf.String()), nil
}
