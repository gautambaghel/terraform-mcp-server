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

// CreateRunComment creates run comment in Terraform.
func CreateRunComment(logger *log.Logger) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool("create_run_comment",
			mcp.WithDescription(`Creates a new Terraform run comment in the specified workspace run.`),
			mcp.WithTitleAnnotation("Create a new Terraform run comment"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithString("run_id",
				mcp.Required(),
				mcp.Description("The Terraform Cloud/Enterprise run id to which the comment will be added."),
			),
			mcp.WithString("run_comment_text",
				mcp.Required(),
				mcp.Description("The text content of the run comment."),
			),
		),
		Handler: func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return createRunCommentHandler(ctx, req, logger)
		},
	}
}

func createRunCommentHandler(ctx context.Context, request mcp.CallToolRequest, logger *log.Logger) (*mcp.CallToolResult, error) {
	runId, err := request.RequireString("run_id")
	if err != nil {
		return ToolError(logger, "missing required input: run_id", err)
	}
	runId = strings.TrimSpace(runId)

	runComment, err := request.RequireString("run_comment_text")
	if err != nil {
		return ToolError(logger, "missing required input: runComment", err)
	}
	runComment = strings.TrimSpace(runComment)

	tfeClient, err := client.GetTfeClientFromContext(ctx, logger)
	if err != nil {
		return ToolError(logger, "failed to get Terraform client", err)
	}

	options := &tfe.CommentCreateOptions{
		Type: "comment",
		Body: runComment,
	}

	runCommentResult, err := tfeClient.Comments.Create(ctx, runId, *options)
	if err != nil {
		return ToolErrorf(logger, "created comment in run '%v': %v", runId, err)
	}

	var buf bytes.Buffer
	if err := jsonapi.MarshalPayload(&buf, runCommentResult); err != nil {
		return ToolError(logger, "failed to marshal comment response", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(buf.String()),
		},
	}, nil
}
