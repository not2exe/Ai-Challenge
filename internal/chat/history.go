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

	if len(h.messages) > h.maxSize {
		h.messages = h.messages[1:]
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
		// Nothing to replace, just prepend summary
		h.messages = append([]api.Message{summary}, h.messages...)
		return
	}

	// Keep the last keepLast messages
	kept := make([]api.Message, keepLast)
	copy(kept, h.messages[len(h.messages)-keepLast:])

	// Build new history: summary + kept messages
	h.messages = append([]api.Message{summary}, kept...)
}
