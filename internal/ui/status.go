package ui

import (
	"fmt"
)

type StatusDisplay struct {
	formatter *Formatter
	enabled   bool
}

func NewStatusDisplay(formatter *Formatter, enabled bool) *StatusDisplay {
	return &StatusDisplay{
		formatter: formatter,
		enabled:   enabled,
	}
}

func (s *StatusDisplay) Show(message string) {
	if !s.enabled {
		return
	}

	fmt.Print("\r\033[K")
	fmt.Print(s.formatter.FormatStatus(message))
}

func (s *StatusDisplay) Hide() {
	if !s.enabled {
		return
	}

	fmt.Print("\r\033[K")
}

func (s *StatusDisplay) Update(message string) {
	if !s.enabled {
		return
	}

	s.Hide()
	s.Show(message)
}

func (s *StatusDisplay) ShowWithNewline(message string) {
	if !s.enabled {
		return
	}

	fmt.Println(s.formatter.FormatStatus(message))
}
