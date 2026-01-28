package scheduler

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/notexe/cli-chat/internal/api"
	"github.com/notexe/cli-chat/internal/config"
	"github.com/notexe/cli-chat/internal/mcp"
)

// Scheduler runs periodic reminder checks and sends notifications via Telegram.
type Scheduler struct {
	provider api.Provider
	mcpMgr   *mcp.Manager
	telegram *TelegramSender
	config   *config.Config
}

// New creates a new Scheduler that reuses the existing MCP manager and provider.
func New(provider api.Provider, mcpMgr *mcp.Manager, telegram *TelegramSender, cfg *config.Config) *Scheduler {
	return &Scheduler{
		provider: provider,
		mcpMgr:   mcpMgr,
		telegram: telegram,
		config:   cfg,
	}
}

// Run blocks and runs tick() on interval + immediately on start.
// It exits when ctx is cancelled.
func (s *Scheduler) Run(ctx context.Context) error {
	if s.config.Scheduler.Interval <= 0 {
		return fmt.Errorf("scheduler interval must be positive, got %d", s.config.Scheduler.Interval)
	}
	interval := time.Duration(s.config.Scheduler.Interval) * time.Second

	log.Printf("[scheduler] Started. Interval: %ds", s.config.Scheduler.Interval)

	// Run immediately on start
	s.tick(ctx)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("[scheduler] Shutting down...")
			return nil
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

func (s *Scheduler) tick(ctx context.Context) {
	log.Println("[scheduler] Checking reminders...")

	summary, err := RunAgenticPrompt(
		ctx,
		s.provider,
		s.mcpMgr,
		s.config.Scheduler.SystemPrompt,
		s.config.Scheduler.PromptTemplate,
		s.config.Model.Name,
		s.config.Model.MaxTokens,
		s.config.Model.Temperature,
	)
	if err != nil {
		log.Printf("[scheduler] Error: agentic prompt failed: %v", err)
		return
	}

	trimmed := strings.TrimSpace(summary)
	if trimmed == "NO_REMINDERS" || trimmed == "NO_DUE_REMINDERS" {
		log.Println("[scheduler] No reminders to report.")
		return
	}

	log.Println("[scheduler] Sending Telegram notification...")
	if err := s.telegram.SendMessage(summary); err != nil {
		log.Printf("[scheduler] Error: Telegram send failed: %v", err)
		return
	}

	log.Println("[scheduler] Notification sent.")
}
