package chat

import (
	"github.com/notexe/cli-chat/internal/api"
)

type History struct {
	messages []api.Message
	maxSize  int
}

func NewHistory(maxSize int) *History {
	return &History{
		messages: make([]api.Message, 0),
		maxSize:  maxSize,
	}
}

func (h *History) Add(msg api.Message) {
	h.messages = append(h.messages, msg)

	for len(h.messages) > h.maxSize {
		h.messages = h.messages[1:]
	}

	// Ensure we never start with orphaned tool messages.
	// A "tool" message must follow an "assistant" message with tool_calls.
	h.dropOrphanedToolMessages()
}

// dropOrphanedToolMessages removes leading tool messages that lost
// their preceding assistant+tool_calls message due to truncation.
func (h *History) dropOrphanedToolMessages() {
	for len(h.messages) > 0 && h.messages[0].Role == "tool" {
		h.messages = h.messages[1:]
	}
	// Also drop an assistant message with tool_calls if the following
	// tool results were already trimmed away.
	if len(h.messages) > 0 && h.messages[0].Role == "assistant" && len(h.messages[0].ToolCalls) > 0 {
		// Check if the next message is a matching tool result
		if len(h.messages) < 2 || h.messages[1].Role != "tool" {
			h.messages = h.messages[1:]
			h.dropOrphanedToolMessages() // recurse in case more orphans
		}
	}
}

func (h *History) GetAll() []api.Message {
	return h.messages
}

func (h *History) Clear() {
	h.messages = make([]api.Message, 0)
}

func (h *History) Size() int {
	return len(h.messages)
}

func (h *History) IsEmpty() bool {
	return len(h.messages) == 0
}

// ReplaceWithSummary replaces old messages with a summary, keeping the last keepLast messages.
func (h *History) ReplaceWithSummary(summary api.Message, keepLast int) {
	if len(h.messages) <= keepLast {
		h.messages = append([]api.Message{summary}, h.messages...)
		return
	}

	// Keep the last keepLast messages
	kept := make([]api.Message, keepLast)
	copy(kept, h.messages[len(h.messages)-keepLast:])

	// Build new history: summary + kept messages
	h.messages = append([]api.Message{summary}, kept...)
	h.dropOrphanedToolMessages()
}
