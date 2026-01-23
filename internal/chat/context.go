package chat

// ContextManager handles context window tracking and summarization decisions.
type ContextManager struct {
	modelLimits map[string]int // model name -> context window size in tokens
	summarizeAt float64        // threshold percentage (0.70 = 70%)
	targetAfter float64        // target percentage after summarization (0.40 = 40%)
}

// DefaultModelLimits returns the default context window sizes for known models.
func DefaultModelLimits() map[string]int {
	return map[string]int{
		"deepseek-chat":     131072,
		"deepseek-reasoner": 131072,
		"llama3":            8192,
		"llama3.1":          128000,
		"llama3.2":          128000,
		"llama2":            4096,
		"mistral":           8192,
		"mixtral":           32768,
		"codellama":         16384,
		"qwen2":             32768,
		"qwen2.5":           32768,
		"gemma":             8192,
		"gemma2":            8192,
		"phi3":              4096,
	}
}

// NewContextManager creates a new context manager with the given thresholds.
func NewContextManager(summarizeAt, targetAfter float64) *ContextManager {
	return &ContextManager{
		modelLimits: DefaultModelLimits(),
		summarizeAt: summarizeAt,
		targetAfter: targetAfter,
	}
}

// GetModelLimit returns the context window size for a model.
// If the model is not found in the limits map, returns the default value.
func (cm *ContextManager) GetModelLimit(modelName string) int {
	if limit, ok := cm.modelLimits[modelName]; ok {
		return limit
	}
	// Default context window for unknown models
	return 8192
}

// SetModelLimit sets a custom context window size for a model.
func (cm *ContextManager) SetModelLimit(modelName string, limit int) {
	cm.modelLimits[modelName] = limit
}

// ShouldSummarize checks if the current token count exceeds the summarization threshold.
func (cm *ContextManager) ShouldSummarize(currentTokens, modelLimit int) bool {
	if modelLimit <= 0 {
		return false
	}
	threshold := float64(modelLimit) * cm.summarizeAt
	return float64(currentTokens) >= threshold
}

// GetTargetTokens returns the target token count after summarization.
func (cm *ContextManager) GetTargetTokens(modelLimit int) int {
	return int(float64(modelLimit) * cm.targetAfter)
}

// GetThresholdTokens returns the token count at which summarization should trigger.
func (cm *ContextManager) GetThresholdTokens(modelLimit int) int {
	return int(float64(modelLimit) * cm.summarizeAt)
}

// GetUsagePercent returns the current context usage as a percentage.
func (cm *ContextManager) GetUsagePercent(currentTokens, modelLimit int) float64 {
	if modelLimit <= 0 {
		return 0
	}
	return (float64(currentTokens) / float64(modelLimit)) * 100
}

// GetSummarizeAt returns the summarization threshold (0.0-1.0).
func (cm *ContextManager) GetSummarizeAt() float64 {
	return cm.summarizeAt
}

// GetTargetAfter returns the target percentage after summarization (0.0-1.0).
func (cm *ContextManager) GetTargetAfter() float64 {
	return cm.targetAfter
}
