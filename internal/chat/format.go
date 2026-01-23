package chat

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

type FormatTemplate struct {
	Name        string
	Description string
	Prompt      string
}

var formatTemplates = map[string]FormatTemplate{
	"json": {
		Name:        "json",
		Description: "Structured JSON output with comprehensive fields",
		Prompt: "IMPORTANT: Respond with raw JSON only. Do NOT wrap your response in markdown code blocks. Return the raw JSON object directly starting with { and ending with }.\n\n" +
			"Always respond in valid JSON format with the following structure:\n" +
			"{\n" +
			"  \"response\": \"main answer/explanation text\",\n" +
			"  \"status\": \"success|info|warning|error\",\n" +
			"  \"tags\": [\"tag1\", \"tag2\", \"tag3\"],\n" +
			"  \"steps\": [\n" +
			"    {\"action\": \"what was done\", \"result\": \"outcome or finding\"}\n" +
			"  ],\n" +
			"  \"urls\": [\n" +
			"    {\"title\": \"reference title\", \"url\": \"https://example.com\"}\n" +
			"  ],\n" +
			"  \"code\": [\n" +
			"    {\"language\": \"go\", \"snippet\": \"code example\"}\n" +
			"  ],\n" +
			"  \"references\": [\"additional notes or references\"],\n" +
			"  \"summary\": \"brief one-line summary\"\n" +
			"}\n\n" +
			"Field descriptions:\n" +
			"- response: Main detailed answer (required)\n" +
			"- status: success/info/warning/error (required)\n" +
			"- tags: Relevant categorization tags (optional)\n" +
			"- steps: Step-by-step breakdown for processes (optional)\n" +
			"- urls: Relevant links with titles (optional)\n" +
			"- code: Code examples with language specification (optional)\n" +
			"- references: Additional notes, tips, or references (optional)\n" +
			"- summary: One-line summary of the response (optional)\n\n" +
			"All fields except response and status are optional - only include them if relevant to the question.\n\n" +
			"Remember: Return raw JSON directly, no markdown code blocks, no backticks.",
	},
}

func GetFormatTemplate(name string) (*FormatTemplate, error) {
	template, ok := formatTemplates[name]
	if !ok {
		return nil, fmt.Errorf("unknown format template: %s", name)
	}
	return &template, nil
}

type JSONResponse struct {
	Response   string              `json:"response"`
	Status     string              `json:"status"`
	Tags       []string            `json:"tags,omitempty"`
	Steps      []map[string]string `json:"steps,omitempty"`
	URLs       []map[string]string `json:"urls,omitempty"`
	Code       []map[string]string `json:"code,omitempty"`
	References []string            `json:"references,omitempty"`
	Summary    string              `json:"summary,omitempty"`
}

func CleanMarkdownCodeBlocks(content string) string {
	content = strings.TrimSpace(content)

	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")

	content = strings.TrimSuffix(content, "```")

	return strings.TrimSpace(content)
}

func HasMarkdownCodeBlocks(content string) bool {
	return strings.Contains(content, "```")
}

// FormatForTerminal converts markdown and LaTeX formatting to terminal-friendly text
func FormatForTerminal(content string) string {
	// First, preprocess LaTeX to Unicode (glamour doesn't handle LaTeX)
	result := preprocessLaTeX(content)

	// Render markdown with glamour
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(100),
	)
	if err != nil {
		return result
	}

	rendered, err := renderer.Render(result)
	if err != nil {
		return result
	}

	return strings.TrimSpace(rendered)
}

// preprocessLaTeX converts LaTeX notation to Unicode before markdown rendering
func preprocessLaTeX(content string) string {
	result := content

	// LaTeX display math blocks \[ ... \] → content
	displayMathRegex := regexp.MustCompile(`\\\[\s*([\s\S]*?)\s*\\\]`)
	result = displayMathRegex.ReplaceAllStringFunc(result, func(match string) string {
		inner := displayMathRegex.FindStringSubmatch(match)
		if len(inner) > 1 {
			return "\n" + cleanLaTeX(inner[1]) + "\n"
		}
		return match
	})

	// LaTeX inline math \( ... \) → content
	inlineMathRegex := regexp.MustCompile(`\\\(\s*(.*?)\s*\\\)`)
	result = inlineMathRegex.ReplaceAllStringFunc(result, func(match string) string {
		inner := inlineMathRegex.FindStringSubmatch(match)
		if len(inner) > 1 {
			return cleanLaTeX(inner[1])
		}
		return match
	})

	// \boxed{...} → [content]
	boxedRegex := regexp.MustCompile(`\\boxed\{([^}]+)\}`)
	result = boxedRegex.ReplaceAllString(result, "**[$1]**")

	return result
}

// cleanLaTeX converts LaTeX commands to Unicode symbols
func cleanLaTeX(content string) string {
	result := content

	replacements := []struct {
		pattern string
		replace string
	}{
		{`\\frac\{([^}]+)\}\{([^}]+)\}`, "($1)/($2)"},
		{`\\sqrt\{([^}]+)\}`, "√($1)"},
		{`\\sqrt`, "√"},
		{`\\pm`, "±"},
		{`\\cdot`, "·"},
		{`\\times`, "×"},
		{`\\div`, "÷"},
		{`\\leq`, "≤"},
		{`\\geq`, "≥"},
		{`\\neq`, "≠"},
		{`\\approx`, "≈"},
		{`\\infty`, "∞"},
		{`\\sum`, "Σ"},
		{`\\prod`, "Π"},
		{`\\int`, "∫"},
		{`\\alpha`, "α"},
		{`\\beta`, "β"},
		{`\\gamma`, "γ"},
		{`\\delta`, "δ"},
		{`\\pi`, "π"},
		{`\\theta`, "θ"},
		{`\\lambda`, "λ"},
		{`\\sigma`, "σ"},
		{`\\omega`, "ω"},
		{`\\text\{([^}]+)\}`, "$1"},
		{`\\quad`, "  "},
		{`\\,`, " "},
		{`\\;`, " "},
		{`\\ `, " "},
	}

	for _, r := range replacements {
		re := regexp.MustCompile(r.pattern)
		result = re.ReplaceAllString(result, r.replace)
	}

	// Superscripts: x^2 → x², x^{10} → x¹⁰
	superscriptRegex := regexp.MustCompile(`\^(\{[^}]+\}|[0-9])`)
	result = superscriptRegex.ReplaceAllStringFunc(result, func(match string) string {
		inner := strings.TrimPrefix(match, "^")
		inner = strings.Trim(inner, "{}")
		return toSuperscript(inner)
	})

	// Remove remaining LaTeX commands
	result = regexp.MustCompile(`\\([a-zA-Z]+)`).ReplaceAllString(result, "$1")

	return result
}

func toSuperscript(s string) string {
	sup := map[rune]rune{
		'0': '⁰', '1': '¹', '2': '²', '3': '³', '4': '⁴',
		'5': '⁵', '6': '⁶', '7': '⁷', '8': '⁸', '9': '⁹',
		'+': '⁺', '-': '⁻', 'n': 'ⁿ',
	}
	var out strings.Builder
	for _, r := range s {
		if v, ok := sup[r]; ok {
			out.WriteRune(v)
		} else {
			out.WriteString("^" + string(r))
		}
	}
	return out.String()
}

func ParseJSONResponse(content string) (*JSONResponse, error) {
	var parsed JSONResponse

	cleaned := CleanMarkdownCodeBlocks(content)

	start := strings.Index(cleaned, "{")
	end := strings.LastIndex(cleaned, "}")

	if start == -1 || end == -1 || start >= end {
		return nil, fmt.Errorf("no valid JSON object found in response")
	}

	jsonContent := cleaned[start : end+1]

	if err := json.Unmarshal([]byte(jsonContent), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return &parsed, nil
}

var (
	FieldNameStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("75")).
			Bold(true)

	ResponseStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	StatusSuccessStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("46")).
			Bold(true)

	StatusInfoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Bold(true)

	StatusWarningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("226")).
			Bold(true)

	StatusErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	TagStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("141"))

	SummaryStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("117")).
			Italic(true)

	StepStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("228"))

	URLStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("87")).
			Underline(true)

	CodeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("120"))

	ReferenceStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("213"))
)

func FormatJSONTable(parsed *JSONResponse) string {
	var result strings.Builder
	result.WriteString("\n")

	if parsed.Response != "" {
		result.WriteString(FieldNameStyle.Render("Response:") + "\n")
		result.WriteString(ResponseStyle.Render(parsed.Response) + "\n\n")
	}

	if parsed.Status != "" {
		result.WriteString(FieldNameStyle.Render("Status:") + " ")
		switch parsed.Status {
		case "success":
			result.WriteString(StatusSuccessStyle.Render(parsed.Status) + "\n\n")
		case "info":
			result.WriteString(StatusInfoStyle.Render(parsed.Status) + "\n\n")
		case "warning":
			result.WriteString(StatusWarningStyle.Render(parsed.Status) + "\n\n")
		case "error":
			result.WriteString(StatusErrorStyle.Render(parsed.Status) + "\n\n")
		default:
			result.WriteString(parsed.Status + "\n\n")
		}
	}

	if parsed.Summary != "" {
		result.WriteString(FieldNameStyle.Render("Summary:") + "\n")
		result.WriteString(SummaryStyle.Render(parsed.Summary) + "\n\n")
	}

	if len(parsed.Tags) > 0 {
		result.WriteString(FieldNameStyle.Render("Tags:") + "\n")
		for _, tag := range parsed.Tags {
			result.WriteString("  • " + TagStyle.Render(tag) + "\n")
		}
		result.WriteString("\n")
	}

	if len(parsed.Steps) > 0 {
		result.WriteString(FieldNameStyle.Render("Steps:") + "\n")
		for i, step := range parsed.Steps {
			action := step["action"]
			stepResult := step["result"]

			result.WriteString(fmt.Sprintf("  %d. %s\n", i+1, StepStyle.Render(action)))
			if stepResult != "" {
				result.WriteString("     → " + StepStyle.Render(stepResult) + "\n")
			}
		}
		result.WriteString("\n")
	}

	if len(parsed.URLs) > 0 {
		result.WriteString(FieldNameStyle.Render("URLs:") + "\n")
		for _, url := range parsed.URLs {
			title := url["title"]
			link := url["url"]
			result.WriteString("  • " + title + "\n")
			result.WriteString("    " + URLStyle.Render(link) + "\n")
		}
		result.WriteString("\n")
	}

	if len(parsed.Code) > 0 {
		result.WriteString(FieldNameStyle.Render("Code:") + "\n")
		for _, code := range parsed.Code {
			lang := code["language"]
			snippet := code["snippet"]

			result.WriteString("  [" + CodeStyle.Render(lang) + "]\n")
			if snippet != "" {
				snippetLines := strings.Split(snippet, "\n")
				for _, line := range snippetLines {
					result.WriteString("    " + CodeStyle.Render(line) + "\n")
				}
			}
		}
		result.WriteString("\n")
	}

	if len(parsed.References) > 0 {
		result.WriteString(FieldNameStyle.Render("References:") + "\n")
		for _, ref := range parsed.References {
			result.WriteString("  • " + ReferenceStyle.Render(ref) + "\n")
		}
		result.WriteString("\n")
	}

	return result.String()
}

