package api

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"time"
	"uuid"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"gitverse.ru/zagvozdeen/agenda/internal/models/enums"
	"gitverse.ru/zagvozdeen/agenda/internal/store"
)

func registerTool[In any](a *API, name, description string, readOnly, public bool, handler func(context.Context, In) (any, error)) {
	uuidSchema := &jsonschema.Schema{Type: "string", Format: "uuid"}
	nullable := func(schema *jsonschema.Schema) *jsonschema.Schema {
		return &jsonschema.Schema{AnyOf: []*jsonschema.Schema{schema, {Type: "null"}}}
	}
	schema, err := jsonschema.For[In](&jsonschema.ForOptions{
		TypeSchemas: map[reflect.Type]*jsonschema.Schema{
			reflect.TypeFor[uuid.UUID]():           uuidSchema,
			reflect.TypeFor[Optional[uuid.UUID]](): nullable(uuidSchema),
			reflect.TypeFor[Optional[string]]():    nullable(&jsonschema.Schema{Type: "string"}),
			reflect.TypeFor[Optional[int]]():       nullable(&jsonschema.Schema{Type: "integer"}),
			reflect.TypeFor[Optional[time.Time]](): nullable(&jsonschema.Schema{Type: "string", Format: "date-time"}),
			reflect.TypeFor[enums.CodeLinkType]():  {Type: "string", Enum: []any{"commit", "pull_request"}},
			reflect.TypeFor[enums.BoardRole]():     {Type: "string", Enum: []any{"owner", "member"}},
			reflect.TypeFor[enums.TaskType]():      {Type: "string", Enum: []any{"feature", "bug", "chore"}},
			reflect.TypeFor[enums.TaskStatus]():    {Type: "string", Enum: []any{"todo", "in_progress", "review", "done", "cancelled"}},
			reflect.TypeFor[enums.TaskPriority]():  {Type: "string", Enum: []any{"low", "medium", "high", "critical"}},
			reflect.TypeFor[enums.SprintStatus]():  {Type: "string", Enum: []any{"planned", "active", "closed"}},
			reflect.TypeFor[enums.LinkType]():      {Type: "string", Enum: []any{"blocks", "relates_to", "duplicates"}},
			reflect.TypeFor[enums.EventType]():     {Type: "string", Enum: []any{"created", "updated", "deleted", "commented", "linked", "unlinked"}},
		},
	})
	if err != nil {
		panic(fmt.Errorf("failed to infer schema for %s: %w", name, err))
	}
	resolved, err := schema.Resolve(nil)
	if err != nil {
		panic(fmt.Errorf("failed to resolve schema for %s: %w", name, err))
	}
	a.mcp.AddTool(&mcp.Tool{
		Name: name, Description: description, InputSchema: schema,
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: readOnly},
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if !public && actor(ctx) == nil {
			return toolError("unauthorized", ErrUnauthorized.Error()), nil
		}
		data := []byte(req.Params.Arguments)
		if len(data) == 0 {
			data = []byte("{}")
		}
		var value any
		if err := json.Unmarshal(data, &value); err != nil {
			return toolError("invalid_arguments", "arguments must be valid JSON"), nil
		}
		if err := resolved.Validate(value); err != nil {
			return toolError("invalid_arguments", err.Error()), nil
		}
		var input In
		if err := json.Unmarshal(data, &input, json.RejectUnknownMembers(true)); err != nil {
			return toolError("invalid_arguments", err.Error()), nil
		}
		if !readOnly {
			txCtx, err := a.store.Begin(ctx)
			if err != nil {
				return a.toolFailure(ctx, name, err), nil
			}
			ctx = txCtx
			defer a.store.Rollback(ctx)
		}
		output, err := handler(ctx, input)
		if err != nil {
			return a.toolFailure(ctx, name, err), nil
		}
		encoded, err := json.Marshal(output)
		if err != nil {
			return a.toolFailure(ctx, name, err), nil
		}
		var structured any
		if err = json.Unmarshal(encoded, &structured); err != nil {
			return a.toolFailure(ctx, name, err), nil
		}
		if !readOnly {
			if err = a.store.Commit(ctx); err != nil {
				return a.toolFailure(ctx, name, err), nil
			}
		}
		return &mcp.CallToolResult{
			Content:           []mcp.Content{&mcp.TextContent{Text: string(encoded)}},
			StructuredContent: structured,
		}, nil
	})
}

func (a *API) toolFailure(ctx context.Context, name string, err error) *mcp.CallToolResult {
	if failure, ok := errors.AsType[*Error](err); ok {
		return toolError(failure.Code, failure.Message)
	}
	if errors.Is(err, store.ErrBoardNotFound) ||
		errors.Is(err, store.ErrBoardMemberNotFound) ||
		errors.Is(err, store.ErrSprintNotFound) ||
		errors.Is(err, store.ErrTaskNotFound) ||
		errors.Is(err, store.ErrTaskLinkNotFound) ||
		errors.Is(err, store.ErrCodeLinkNotFound) ||
		errors.Is(err, store.ErrCommentNotFound) ||
		errors.Is(err, store.ErrUserNotFound) {
		return toolError("not_found", err.Error())
	}
	if failure, ok := errors.AsType[*pgconn.PgError](err); ok {
		switch failure.Code {
		case "23505":
			return toolError("conflict", "resource already exists or conflicts with the current state")
		case "23503":
			return toolError("invalid_reference", "referenced resource does not exist")
		case "23514", "22001", "22003":
			return toolError("invalid_arguments", "value violates a resource constraint")
		}
	}
	a.log.ErrorContext(ctx, "Failed to call MCP tool", slog.String("tool", name), slog.Any("error", err))
	return toolError("internal_error", "internal server error")
}

func toolError(code, message string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError:           true,
		Content:           []mcp.Content{&mcp.TextContent{Text: message}},
		StructuredContent: map[string]any{"error": map[string]string{"code": code, "message": message}},
	}
}
