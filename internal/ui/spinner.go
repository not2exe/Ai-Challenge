package ui

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Spinner frames for animation
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Alternative spinner styles
var (
	dotsSpinner  = []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}
	arrowSpinner = []string{"←", "↖", "↑", "↗", "→", "↘", "↓", "↙"}
	pulseSpinner = []string{"█", "▓", "▒", "░", "▒", "▓"}
)

// Spinner provides an animated loading indicator
type Spinner struct {
	frames    []string
	message   string
	running   bool
	stopCh    chan struct{}
	done      chan struct{}
	mu        sync.Mutex
	style     lipgloss.Style
	msgStyle  lipgloss.Style
	interval  time.Duration
	colored   bool
}

// SpinnerStyle defines different spinner visual styles
type SpinnerStyle int

const (
	SpinnerDots SpinnerStyle = iota
	SpinnerBraille
	SpinnerArrow
	SpinnerPulse
)

// NewSpinner creates a new spinner with the given style
func NewSpinner(colored bool, style ...SpinnerStyle) *Spinner {
	frames := spinnerFrames
	if len(style) > 0 {
		switch style[0] {
		case SpinnerDots:
			frames = dotsSpinner
		case SpinnerArrow:
			frames = arrowSpinner
		case SpinnerPulse:
			frames = pulseSpinner
		}
	}

	spinStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	msgStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Italic(true)

	return &Spinner{
		frames:   frames,
		style:    spinStyle,
		msgStyle: msgStyle,
		interval: 80 * time.Millisecond,
		colored:  colored,
	}
}

// Start begins the spinner animation with a message
func (s *Spinner) Start(message string) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		s.Update(message)
		return
	}

	s.message = message
	s.running = true
	s.stopCh = make(chan struct{})
	s.done = make(chan struct{})
	s.mu.Unlock()

	go s.animate()
}

// Stop stops the spinner and clears the line
func (s *Spinner) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	close(s.stopCh)
	s.mu.Unlock()

	<-s.done
	fmt.Print("\r\033[K")
}

// Update changes the spinner message without stopping
func (s *Spinner) Update(message string) {
	s.mu.Lock()
	s.message = message
	s.mu.Unlock()
}

// StopWithMessage stops and displays a final message
func (s *Spinner) StopWithMessage(message string) {
	s.Stop()
	if s.colored {
		successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
		fmt.Println(successStyle.Render("✓") + " " + message)
	} else {
		fmt.Println("✓ " + message)
	}
}

// StopWithError stops and displays an error message
func (s *Spinner) StopWithError(message string) {
	s.Stop()
	if s.colored {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		fmt.Println(errorStyle.Render("✗") + " " + message)
	} else {
		fmt.Println("✗ " + message)
	}
}

func (s *Spinner) animate() {
	defer close(s.done)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	frame := 0
	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.mu.Lock()
			msg := s.message
			s.mu.Unlock()

			s.render(frame, msg)
			frame = (frame + 1) % len(s.frames)
		}
	}
}

func (s *Spinner) render(frame int, message string) {
	spinChar := s.frames[frame]

	var output string
	if s.colored {
		output = fmt.Sprintf("\r\033[K%s %s",
			s.style.Render(spinChar),
			s.msgStyle.Render(message))
	} else {
		output = fmt.Sprintf("\r\033[K%s %s", spinChar, message)
	}

	fmt.Print(output)
	os.Stdout.Sync() // Flush to ensure animation renders immediately
}
