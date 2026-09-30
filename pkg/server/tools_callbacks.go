package server

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/nbaertsch/mythic-sdk-go/pkg/mythic/types"
)

// registerCallbacksTools registers callback management MCP tools
func (s *Server) registerCallbacksTools() {
	// mythic_get_all_callbacks - List all callbacks
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "mythic_get_all_callbacks",
		Description: "Get a list of all callbacks (active agent connections) in Mythic",
	}, s.handleGetAllCallbacks)

	// mythic_get_active_callbacks - List active callbacks
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "mythic_get_active_callbacks",
		Description: "Get a list of all active callbacks",
	}, s.handleGetActiveCallbacks)

	// mythic_get_callback - Get specific callback
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "mythic_get_callback",
		Description: "Get details of a specific callback by its display_id (the number shown in the Mythic UI)",
	}, s.handleGetCallback)

	// mythic_update_callback - Update callback
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "mythic_update_callback",
		Description: "Update a callback's properties (description, active status, etc.)",
	}, s.handleUpdateCallback)

	// mythic_delete_callback - Delete callback
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "mythic_delete_callback",
		Description: "Delete one or more callbacks from Mythic. Admin users perform a hard delete (removes callbacks and tasks from the database). Non-admin users perform a soft delete (sets active=false, dead=true), since hard deletion requires the mythic_admin role.",
	}, s.handleDeleteCallback)

	// mythic_get_loaded_commands - Get loaded commands
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name: "mythic_get_loaded_commands",
		Description: "Get all commands loaded in a callback, including both the agent's built-in commands " +
			"and any dynamically loaded commands (e.g. forge script_only commands). Each command " +
			"includes its payload_type_name (which agent it belongs to) and script_only flag " +
			"(whether it runs server-side or on the agent). Use mythic_get_payload_type_commands " +
			"to see all commands for a payload type without needing a callback.",
	}, s.handleGetLoadedCommands)

	// mythic_export_callback_config - Export callback config
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name: "mythic_export_callback_config",
		Description: "Export a callback's configuration as JSON. " +
			"IMPORTANT: The agent_callback_id parameter requires the callback's UUID string " +
			"(e.g. 'eebbb756-3e16-42b4-88d8-1a01f3e5e320'), NOT the display_id number. " +
			"Use mythic_get_callbacks to find the agent_callback_id (UUID) for a callback.",
	}, s.handleExportCallbackConfig)

	// mythic_import_callback_config - Import callback config
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "mythic_import_callback_config",
		Description: "Import a callback configuration from JSON",
	}, s.handleImportCallbackConfig)

	// mythic_get_callback_tokens - Get callback tokens
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "mythic_get_callback_tokens",
		Description: "Get list of tokens associated with a callback",
	}, s.handleGetCallbackTokens)

	// mythic_add_callback_edge - Add P2P callback edge
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "mythic_add_callback_edge",
		Description: "Add a P2P connection between two callbacks in the callback graph",
	}, s.handleAddCallbackEdge)

	// mythic_remove_callback_edge - Remove P2P callback edge
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "mythic_remove_callback_edge",
		Description: "Remove a P2P connection between callbacks",
	}, s.handleRemoveCallbackEdge)
}

// Tool argument types for callbacks tools

type getAllCallbacksArgs struct{}

type getActiveCallbacksArgs struct{}

type getCallbackArgs struct {
	CallbackID int `json:"callback_id" jsonschema:"Callback display_id (the number shown in the Mythic UI, not the internal database id)"`
}

type updateCallbackArgs struct {
	CallbackID  int      `json:"callback_id" jsonschema:"Callback display_id (the number shown in the Mythic UI, not the internal database id)"`
	Active      *bool    `json:"active,omitempty" jsonschema:"Set callback active/inactive status"`
	Locked      *bool    `json:"locked,omitempty" jsonschema:"Lock/unlock callback for tasking"`
	Description *string  `json:"description,omitempty" jsonschema:"Set callback description"`
	IPs         []string `json:"ips,omitempty" jsonschema:"Update IP addresses"`
	User        *string  `json:"user,omitempty" jsonschema:"Update username"`
	Host        *string  `json:"host,omitempty" jsonschema:"Update hostname"`
}

type deleteCallbackArgs struct {
	CallbackIDs []int `json:"callback_ids" jsonschema:"Array of callback display_ids (the numbers shown in the Mythic UI, not internal database ids)"`
}

type getLoadedCommandsArgs struct {
	CallbackID int `json:"callback_id" jsonschema:"Callback display_id (the number shown in the Mythic UI, not the internal database id)"`
}

type exportCallbackConfigArgs struct {
	AgentCallbackID string `json:"agent_callback_id" jsonschema:"Callback UUID string (NOT display_id number). Get this from mythic_get_callbacks agent_callback_id field."`
}

type importCallbackConfigArgs struct {
	Config string `json:"config" jsonschema:"JSON configuration string to import"`
}

type getCallbackTokensArgs struct {
	CallbackID int `json:"callback_id" jsonschema:"Callback display_id (the number shown in the Mythic UI, not the internal database id)"`
}

type addCallbackEdgeArgs struct {
	SourceID      int    `json:"source_id" jsonschema:"Source callback ID"`
	DestinationID int    `json:"destination_id" jsonschema:"Destination callback ID"`
	C2ProfileName string `json:"c2_profile_name" jsonschema:"C2 profile name for the connection"`
}

type removeCallbackEdgeArgs struct {
	EdgeID int `json:"edge_id" jsonschema:"ID of the callback graph edge to remove"`
}

// Tool handlers

// handleGetAllCallbacks retrieves all callbacks
func (s *Server) handleGetAllCallbacks(ctx context.Context, req *mcp.CallToolRequest, args getAllCallbacksArgs) (*mcp.CallToolResult, any, error) {
	callbacks, err := s.mythicClient.GetAllCallbacks(ctx)
	if err != nil {
		return nil, nil, translateError(err)
	}

	data, err := json.MarshalIndent(callbacks, "", "  ")
	if err != nil {
		return nil, nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: fmt.Sprintf("All callbacks (%d total):\n\n%s", len(callbacks), string(data)),
			},
		},
	}, wrapList(callbacks), nil
}

// handleGetActiveCallbacks retrieves active callbacks
func (s *Server) handleGetActiveCallbacks(ctx context.Context, req *mcp.CallToolRequest, args getActiveCallbacksArgs) (*mcp.CallToolResult, any, error) {
	callbacks, err := s.mythicClient.GetAllActiveCallbacks(ctx)
	if err != nil {
		return nil, nil, translateError(err)
	}

	data, err := json.MarshalIndent(callbacks, "", "  ")
	if err != nil {
		return nil, nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: fmt.Sprintf("Active callbacks (%d total):\n\n%s", len(callbacks), string(data)),
			},
		},
	}, wrapList(callbacks), nil
}

// handleGetCallback retrieves a specific callback by ID
func (s *Server) handleGetCallback(ctx context.Context, req *mcp.CallToolRequest, args getCallbackArgs) (*mcp.CallToolResult, any, error) {
	callback, err := s.mythicClient.GetCallbackByID(ctx, args.CallbackID)
	if err != nil {
		return nil, nil, translateError(err)
	}

	data, err := json.MarshalIndent(callback, "", "  ")
	if err != nil {
		return nil, nil, err
	}

	status := "Inactive"
	if callback.Active {
		status = "Active"
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: fmt.Sprintf("Callback %d (%s): %s@%s (%s)\n\n%s",
					callback.DisplayID, status, callback.User, callback.Host, callback.OS, string(data)),
			},
		},
	}, callback, nil
}

// handleUpdateCallback updates a callback's properties
func (s *Server) handleUpdateCallback(ctx context.Context, req *mcp.CallToolRequest, args updateCallbackArgs) (*mcp.CallToolResult, any, error) {
	updateReq := &types.CallbackUpdateRequest{
		CallbackDisplayID: args.CallbackID,
		Active:            args.Active,
		Locked:            args.Locked,
		Description:       args.Description,
		IPs:               args.IPs,
		User:              args.User,
		Host:              args.Host,
	}

	err := s.mythicClient.UpdateCallback(ctx, updateReq)
	if err != nil {
		return nil, nil, translateError(err)
	}

	return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: fmt.Sprintf("Successfully updated callback %d", args.CallbackID),
				},
			},
		}, map[string]interface{}{
			"callback_id": args.CallbackID,
			"success":     true,
		}, nil
}

// handleDeleteCallback deletes one or more callbacks
func (s *Server) handleDeleteCallback(ctx context.Context, req *mcp.CallToolRequest, args deleteCallbackArgs) (*mcp.CallToolResult, any, error) {
	if len(args.CallbackIDs) == 0 {
		return nil, nil, fmt.Errorf("at least one callback ID is required")
	}

	err := s.mythicClient.DeleteCallback(ctx, args.CallbackIDs)
	if err != nil {
		return nil, nil, translateError(err)
	}

	return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: fmt.Sprintf("Successfully deleted %d callback(s)", len(args.CallbackIDs)),
				},
			},
		}, map[string]interface{}{
			"callback_ids": args.CallbackIDs,
			"count":        len(args.CallbackIDs),
			"success":      true,
		}, nil
}

// handleGetLoadedCommands retrieves loaded commands for a callback
func (s *Server) handleGetLoadedCommands(ctx context.Context, req *mcp.CallToolRequest, args getLoadedCommandsArgs) (*mcp.CallToolResult, any, error) {
	// args.CallbackID is a display_id, but GetLoadedCommands filters on the
	// internal callback_id. Resolve display_id -> internal id first, otherwise
	// the query silently returns zero rows.
	callback, err := s.mythicClient.GetCallbackByID(ctx, args.CallbackID)
	if err != nil {
		return nil, nil, translateError(err)
	}

	commands, err := s.mythicClient.GetLoadedCommands(ctx, callback.ID)
	if err != nil {
		return nil, nil, translateError(err)
	}

	data, err := json.MarshalIndent(commands, "", "  ")
	if err != nil {
		return nil, nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: fmt.Sprintf("Loaded commands for callback %d (%d total):\n\n%s", args.CallbackID, len(commands), string(data)),
			},
		},
	}, wrapList(commands), nil
}

// handleExportCallbackConfig exports a callback's configuration
func (s *Server) handleExportCallbackConfig(ctx context.Context, req *mcp.CallToolRequest, args exportCallbackConfigArgs) (*mcp.CallToolResult, any, error) {
	config, err := s.mythicClient.ExportCallbackConfig(ctx, args.AgentCallbackID)
	if err != nil {
		return nil, nil, translateError(err)
	}

	return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: fmt.Sprintf("Successfully exported callback configuration\n\nConfig (length: %d bytes):\n%s",
						len(config), config),
				},
			},
		}, map[string]interface{}{
			"agent_callback_id": args.AgentCallbackID,
			"config":            config,
			"size":              len(config),
		}, nil
}

// handleImportCallbackConfig imports a callback configuration
func (s *Server) handleImportCallbackConfig(ctx context.Context, req *mcp.CallToolRequest, args importCallbackConfigArgs) (*mcp.CallToolResult, any, error) {
	err := s.mythicClient.ImportCallbackConfig(ctx, args.Config)
	if err != nil {
		return nil, nil, translateError(err)
	}

	return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: "Successfully imported callback configuration",
				},
			},
		}, map[string]interface{}{
			"success": true,
		}, nil
}

// handleGetCallbackTokens retrieves tokens for a callback
func (s *Server) handleGetCallbackTokens(ctx context.Context, req *mcp.CallToolRequest, args getCallbackTokensArgs) (*mcp.CallToolResult, any, error) {
	tokens, err := s.mythicClient.GetCallbackTokensByCallback(ctx, args.CallbackID)
	if err != nil {
		return nil, nil, translateError(err)
	}

	data, err := json.MarshalIndent(tokens, "", "  ")
	if err != nil {
		return nil, nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: fmt.Sprintf("Tokens for callback %d (%d total):\n\n%s", args.CallbackID, len(tokens), string(data)),
			},
		},
	}, wrapList(tokens), nil
}

// handleAddCallbackEdge adds a P2P callback edge
func (s *Server) handleAddCallbackEdge(ctx context.Context, req *mcp.CallToolRequest, args addCallbackEdgeArgs) (*mcp.CallToolResult, any, error) {
	err := s.mythicClient.AddCallbackGraphEdge(ctx, args.SourceID, args.DestinationID, args.C2ProfileName)
	if err != nil {
		return nil, nil, translateError(err)
	}

	return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: fmt.Sprintf("Successfully added P2P edge: callback %d → callback %d (via %s)",
						args.SourceID, args.DestinationID, args.C2ProfileName),
				},
			},
		}, map[string]interface{}{
			"source_id":       args.SourceID,
			"destination_id":  args.DestinationID,
			"c2_profile_name": args.C2ProfileName,
			"success":         true,
		}, nil
}

// handleRemoveCallbackEdge removes a P2P callback edge
func (s *Server) handleRemoveCallbackEdge(ctx context.Context, req *mcp.CallToolRequest, args removeCallbackEdgeArgs) (*mcp.CallToolResult, any, error) {
	err := s.mythicClient.RemoveCallbackGraphEdge(ctx, args.EdgeID)
	if err != nil {
		return nil, nil, translateError(err)
	}

	return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: fmt.Sprintf("Successfully removed P2P edge %d", args.EdgeID),
				},
			},
		}, map[string]interface{}{
			"edge_id": args.EdgeID,
			"success": true,
		}, nil
}
