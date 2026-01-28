package reminder

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	serverName    = "reminder"
	serverVersion = "1.0.0"
)

// Server is the MCP server for reminder management.
type Server struct {
	mcpServer *server.MCPServer
	store     *Store
}

// NewServer creates a new Reminder MCP server backed by the given store.
func NewServer(store *Store) *Server {
	s := &Server{
		store: store,
	}

	s.mcpServer = server.NewMCPServer(
		serverName,
		serverVersion,
		server.WithToolCapabilities(false),
	)

	s.registerTools()
	return s
}

// MCPServer returns the underlying MCP server for serving.
func (s *Server) MCPServer() *server.MCPServer {
	return s.mcpServer
}

func (s *Server) registerTools() {
	// add_reminder
	s.mcpServer.AddTool(
		mcp.NewTool("add_reminder",
			mcp.WithDescription("Add a new reminder with a title, due date, optional description and priority"),
			mcp.WithString("title", mcp.Required(), mcp.Description("Reminder title")),
			mcp.WithString("due_date", mcp.Required(), mcp.Description("Due date in RFC3339 format (e.g. 2025-01-15T09:00:00Z)")),
			mcp.WithString("description", mcp.Description("Optional description")),
			mcp.WithString("priority", mcp.Description("Priority: low, medium, high (default: medium)")),
		),
		s.handleAddReminder,
	)

	// list_reminders
	s.mcpServer.AddTool(
		mcp.NewTool("list_reminders",
			mcp.WithDescription("List all reminders, optionally filtered by status (pending or completed)"),
			mcp.WithString("status", mcp.Description("Filter by status: pending, completed, or empty for all")),
		),
		s.handleListReminders,
	)

	// get_due_reminders
	s.mcpServer.AddTool(
		mcp.NewTool("get_due_reminders",
			mcp.WithDescription("Get all pending reminders that are due now or overdue"),
		),
		s.handleGetDueReminders,
	)

	// complete_reminder
	s.mcpServer.AddTool(
		mcp.NewTool("complete_reminder",
			mcp.WithDescription("Mark a reminder as completed"),
			mcp.WithNumber("id", mcp.Required(), mcp.Description("Reminder ID")),
		),
		s.handleCompleteReminder,
	)

	// delete_reminder
	s.mcpServer.AddTool(
		mcp.NewTool("delete_reminder",
			mcp.WithDescription("Delete a reminder permanently"),
			mcp.WithNumber("id", mcp.Required(), mcp.Description("Reminder ID")),
		),
		s.handleDeleteReminder,
	)

	// update_reminder
	s.mcpServer.AddTool(
		mcp.NewTool("update_reminder",
			mcp.WithDescription("Update a reminder's fields (title, description, due_date, priority)"),
			mcp.WithNumber("id", mcp.Required(), mcp.Description("Reminder ID")),
			mcp.WithString("title", mcp.Description("New title")),
			mcp.WithString("description", mcp.Description("New description")),
			mcp.WithString("due_date", mcp.Description("New due date in RFC3339 format")),
			mcp.WithString("priority", mcp.Description("New priority: low, medium, high")),
		),
		s.handleUpdateReminder,
	)
}

func (s *Server) handleAddReminder(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	title := req.GetString("title", "")
	dueDateStr := req.GetString("due_date", "")
	description := req.GetString("description", "")
	priority := req.GetString("priority", "")

	if title == "" {
		return mcp.NewToolResultError("title is required"), nil
	}
	if dueDateStr == "" {
		return mcp.NewToolResultError("due_date is required"), nil
	}

	dueDate, err := time.Parse(time.RFC3339, dueDateStr)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid due_date format: %v (use RFC3339, e.g. 2025-01-15T09:00:00Z)", err)), nil
	}

	if priority == "" {
		priority = PriorityMedium
	}

	r := Reminder{
		Title:       title,
		Description: description,
		DueDate:     dueDate,
		Priority:    priority,
	}

	added, err := s.store.Add(r)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to add reminder: %v", err)), nil
	}

	output, _ := json.MarshalIndent(added, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func (s *Server) handleListReminders(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	status := req.GetString("status", "")

	reminders, err := s.store.List(status)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list reminders: %v", err)), nil
	}

	if len(reminders) == 0 {
		return mcp.NewToolResultText("No reminders found."), nil
	}

	output, _ := json.MarshalIndent(reminders, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func (s *Server) handleGetDueReminders(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	reminders, err := s.store.GetDue()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get due reminders: %v", err)), nil
	}

	if len(reminders) == 0 {
		return mcp.NewToolResultText("No due reminders."), nil
	}

	output, _ := json.MarshalIndent(reminders, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func (s *Server) handleCompleteReminder(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	idFloat := req.GetFloat("id", -1)
	if idFloat < 0 {
		return mcp.NewToolResultError("id is required and must be a positive number"), nil
	}
	id := int64(idFloat)

	if err := s.store.Complete(id); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to complete reminder: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Reminder %d marked as completed.", id)), nil
}

func (s *Server) handleDeleteReminder(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	idFloat := req.GetFloat("id", -1)
	if idFloat < 0 {
		return mcp.NewToolResultError("id is required and must be a positive number"), nil
	}
	id := int64(idFloat)

	if err := s.store.Delete(id); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to delete reminder: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Reminder %d deleted.", id)), nil
}

func (s *Server) handleUpdateReminder(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	idFloat := req.GetFloat("id", -1)
	if idFloat < 0 {
		return mcp.NewToolResultError("id is required and must be a positive number"), nil
	}
	id := int64(idFloat)

	var fields UpdateFields

	if v := req.GetString("title", ""); v != "" {
		fields.Title = &v
	}
	if v := req.GetString("description", ""); v != "" {
		fields.Description = &v
	}
	if v := req.GetString("due_date", ""); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid due_date: %v", err)), nil
		}
		fields.DueDate = &t
	}
	if v := req.GetString("priority", ""); v != "" {
		fields.Priority = &v
	}

	updated, err := s.store.Update(id, fields)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to update reminder: %v", err)), nil
	}

	output, _ := json.MarshalIndent(updated, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}
