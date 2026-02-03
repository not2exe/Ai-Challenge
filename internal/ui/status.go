package ui

import (
	"fmt"
)

// StatusDisplay manages status messages with optional spinner animation
type StatusDisplay struct {
	formatter *Formatter
	enabled   bool
	spinner   *Spinner
	useSpinner bool
}

// NewStatusDisplay creates a new status display
func NewStatusDisplay(formatter *Formatter, enabled bool) *StatusDisplay {
	return &StatusDisplay{
		formatter:  formatter,
		enabled:    enabled,
		spinner:    NewSpinner(formatter.colored),
		useSpinner: true, // Enable spinner by default
	}
}

// SetUseSpinner enables or disables the animated spinner
func (s *StatusDisplay) SetUseSpinner(use bool) {
	s.useSpinner = use
}

// Show displays a status message with animation
func (s *StatusDisplay) Show(message string) {
	if !s.enabled {
		return
	}

	if s.useSpinner {
		s.spinner.Start(message)
	} else {
		fmt.Print("\r\033[K")
		fmt.Print(s.formatter.FormatStatus(message))
	}
}

// Hide clears the status display
func (s *StatusDisplay) Hide() {
	if !s.enabled {
		return
	}

	if s.useSpinner {
		s.spinner.Stop()
	} else {
		fmt.Print("\r\033[K")
	}
}

// Update changes the status message without flicker
func (s *StatusDisplay) Update(message string) {
	if !s.enabled {
		return
	}

	if s.useSpinner {
		s.spinner.Update(message)
	} else {
		s.Hide()
		s.Show(message)
	}
}

// ShowWithNewline displays a status message with newline (persistent)
func (s *StatusDisplay) ShowWithNewline(message string) {
	if !s.enabled {
		return
	}

	fmt.Println(s.formatter.FormatStatus(message))
}

// Success stops with a success checkmark
func (s *StatusDisplay) Success(message string) {
	if !s.enabled {
		return
	}

	if s.useSpinner {
		s.spinner.StopWithMessage(message)
	} else {
		fmt.Print("\r\033[K")
		fmt.Println(s.formatter.FormatInfo("✓ " + message))
	}
}

// Error stops with an error mark
func (s *StatusDisplay) Error(message string) {
	if !s.enabled {
		return
	}

	if s.useSpinner {
		s.spinner.StopWithError(message)
	} else {
		fmt.Print("\r\033[K")
		fmt.Println(s.formatter.FormatError(fmt.Errorf(message)))
	}
}
