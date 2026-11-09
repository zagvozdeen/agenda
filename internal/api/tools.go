package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"uuid"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"gitverse.ru/zagvozdeen/agenda/internal/models/enums"
)

func registerTool[In any](a *API, name, description string, readOnly, public bool, handler func(context.Context, In) (any, error)) {
	schema, err := jsonschema.For[In](&jsonschema.ForOptions{
		TypeSchemas: map[reflect.Type]*jsonschema.Schema{
			reflect.TypeFor[uuid.UUID]():          {Type: "string", Format: "uuid"},
			reflect.TypeFor[enums.BoardRole]():    {Type: "string", Enum: []any{"owner", "member"}},
			reflect.TypeFor[enums.TaskType]():     {Type: "string", Enum: []any{"feature", "bug", "chore"}},
			reflect.TypeFor[enums.TaskStatus]():   {Type: "string", Enum: []any{"todo", "in_progress", "review", "done", "cancelled"}},
			reflect.TypeFor[enums.TaskPriority](): {Type: "string", Enum: []any{"low", "medium", "high", "critical"}},
			reflect.TypeFor[enums.SprintStatus](): {Type: "string", Enum: []any{"planned", "active", "closed"}},
			reflect.TypeFor[enums.LinkType]():     {Type: "string", Enum: []any{"blocks", "relates_to", "duplicates"}},
			reflect.TypeFor[enums.EventType]():    {Type: "string", Enum: []any{"created", "updated", "deleted", "commented", "linked", "unlinked"}},
		},
	})
	if err != nil {
		panic(fmt.Errorf("failed to infer schema for %s: %w", name, err))
	}
	mcp.AddTool(a.mcp, &mcp.Tool{
		Name:        name,
		Description: description + " This tool is not implemented yet.",
		InputSchema: schema,
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: readOnly},
	}, func(ctx context.Context, req *mcp.CallToolRequest, input In) (*mcp.CallToolResult, any, error) {
		if !public && actor(ctx) == nil {
			return toolError("unauthorized", ErrUnauthorized.Error()), nil, nil
		}
		output, err := handler(ctx, input)
		if err != nil {
			if errors.Is(err, ErrNotImplemented) {
				return toolError("not_implemented", name+" is not implemented yet"), nil, nil
			}
			a.log.ErrorContext(ctx, "Failed to call MCP tool", slog.String("tool", name), slog.Any("error", err))
			return toolError("internal_error", "internal server error"), nil, nil
		}
		return nil, output, nil
	})
}

func toolError(code, message string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError:           true,
		Content:           []mcp.Content{&mcp.TextContent{Text: message}},
		StructuredContent: map[string]any{"error": map[string]string{"code": code, "message": message}},
	}
}
