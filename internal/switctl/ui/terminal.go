// Copyright © 2025 jackelyj <dreamerlyj@gmail.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

// Package ui provides interactive user interface components for switctl.
package ui

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/innovationmech/swit/internal/switctl/interfaces"
)

// TerminalUI implements the InteractiveUI interface for terminal-based interactions.
type TerminalUI struct {
	input    io.Reader
	reader   *bufio.Reader
	output   io.Writer
	style    interfaces.UIStyle
	spinners []string
	noColor  bool
	verbose  bool
	mu       sync.RWMutex
}

// ColorImpl implements the Color interface using fatih/color.
type ColorImpl struct {
	color *color.Color
}

// Sprint colorizes text.
func (c *ColorImpl) Sprint(a ...interface{}) string {
	if c.color == nil {
		return fmt.Sprint(a...)
	}
	return c.color.Sprint(a...)
}

// Sprintf colorizes formatted text.
func (c *ColorImpl) Sprintf(format string, a ...interface{}) string {
	if c.color == nil {
		return fmt.Sprintf(format, a...)
	}
	return c.color.Sprintf(format, a...)
}

// ProgressBarImpl implements the ProgressBar interface.
type ProgressBarImpl struct {
	title    string
	total    int
	current  int
	message  string
	output   io.Writer
	style    interfaces.UIStyle
	finished bool
	mu       sync.RWMutex
}

// NewTerminalUI creates a new terminal-based UI instance.
func NewTerminalUI(options ...TerminalUIOption) *TerminalUI {
	ui := &TerminalUI{
		input:   os.Stdin,
		output:  os.Stdout,
		style:   DefaultUIStyle(),
		noColor: false,
		verbose: false,
		spinners: []string{
			"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏",
		},
	}

	for _, option := range options {
		option(ui)
	}

	// Initialize reader with the final input
	ui.reader = bufio.NewReader(ui.input)

	// Disable colors if no-color is set
	if ui.noColor {
		color.NoColor = true
	}

	return ui
}

// TerminalUIOption represents a configuration option for TerminalUI.
type TerminalUIOption func(*TerminalUI)

// WithInput sets the input reader.
func WithInput(input io.Reader) TerminalUIOption {
	return func(ui *TerminalUI) {
		ui.input = input
	}
}

// WithOutput sets the output writer.
func WithOutput(output io.Writer) TerminalUIOption {
	return func(ui *TerminalUI) {
		ui.output = output
	}
}

// WithNoColor disables color output.
func WithNoColor(noColor bool) TerminalUIOption {
	return func(ui *TerminalUI) {
		ui.noColor = noColor
	}
}

// WithVerbose enables verbose output.
func WithVerbose(verbose bool) TerminalUIOption {
	return func(ui *TerminalUI) {
		ui.verbose = verbose
	}
}

// WithStyle sets the UI style.
func WithStyle(style interfaces.UIStyle) TerminalUIOption {
	return func(ui *TerminalUI) {
		ui.style = style
	}
}

// DefaultUIStyle creates the default UI color scheme.
func DefaultUIStyle() interfaces.UIStyle {
	return interfaces.UIStyle{
		Primary:   &ColorImpl{color: color.New(color.FgCyan, color.Bold)},
		Success:   &ColorImpl{color: color.New(color.FgGreen, color.Bold)},
		Warning:   &ColorImpl{color: color.New(color.FgYellow, color.Bold)},
		Error:     &ColorImpl{color: color.New(color.FgRed, color.Bold)},
		Info:      &ColorImpl{color: color.New(color.FgBlue)},
		Highlight: &ColorImpl{color: color.New(color.FgMagenta)},
	}
}

// ShowWelcome displays the welcome screen with ASCII art and branding.
func (ui *TerminalUI) ShowWelcome() error {
	welcome := `
╭─────────────────────────────────────────────────────────╮
│                                                         │
│   🚀 Welcome to Swit Framework Scaffolding Tool        │
│                                                         │
│   Build powerful microservices with ease!              │
│                                                         │
│   ✨ Features:                                          │
│   • Interactive service generation                     │
│   • Code quality checks                                │
│   • Beautiful CLI interface                            │
│   • Development workflow automation                    │
│                                                         │
╰─────────────────────────────────────────────────────────╯
`
	fmt.Fprint(ui.output, ui.style.Primary.Sprint(welcome))
	fmt.Fprintln(ui.output)
	return nil
}

// ShowSuccess displays a success message with appropriate styling.
func (ui *TerminalUI) ShowSuccess(message string) error {
	fmt.Fprintf(ui.output, "%s %s\n",
		ui.style.Success.Sprint("✅"),
		ui.style.Success.Sprint(message))
	return nil
}

// ShowError displays an error message with appropriate styling.
func (ui *TerminalUI) ShowError(err error) error {
	fmt.Fprintf(ui.output, "%s %s\n",
		ui.style.Error.Sprint("❌"),
		ui.style.Error.Sprint(err.Error()))
	return nil
}

// ShowInfo displays an informational message.
func (ui *TerminalUI) ShowInfo(message string) error {
	fmt.Fprintf(ui.output, "%s %s\n",
		ui.style.Info.Sprint("ℹ️"),
		ui.style.Info.Sprint(message))
	return nil
}

// ShowWarning displays a warning message.
func (ui *TerminalUI) ShowWarning(message string) error {
	fmt.Fprintf(ui.output, "%s %s\n",
		ui.style.Warning.Sprint("⚠️"),
		ui.style.Warning.Sprint(message))
	return nil
}

// ShowTable displays tabular data with headers and rows.
func (ui *TerminalUI) ShowTable(headers []string, rows [][]string) error {
	if len(headers) == 0 {
		return fmt.Errorf("headers cannot be empty")
	}

	// Calculate column widths
	colWidths := make([]int, len(headers))
	for i, header := range headers {
		colWidths[i] = len(header)
	}

	for _, row := range rows {
		for i, cell := range row {
			if i < len(colWidths) && len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}

	// Print top border
	ui.printTableBorder(colWidths, "┌", "┬", "┐")

	// Print headers
	fmt.Fprint(ui.output, "│")
	for i, header := range headers {
		padding := colWidths[i] - len(header)
		fmt.Fprintf(ui.output, " %s%s │",
			ui.style.Primary.Sprint(header),
			strings.Repeat(" ", padding))
	}
	fmt.Fprintln(ui.output)

	// Print separator
	ui.printTableBorder(colWidths, "├", "┼", "┤")

	// Print rows
	for _, row := range rows {
		fmt.Fprint(ui.output, "│")
		for i, cell := range row {
			if i >= len(colWidths) {
				break
			}
			padding := colWidths[i] - len(cell)
			fmt.Fprintf(ui.output, " %s%s │", cell, strings.Repeat(" ", padding))
		}
		fmt.Fprintln(ui.output)
	}

	// Print bottom border
	ui.printTableBorder(colWidths, "└", "┴", "┘")

	return nil
}

// printTableBorder prints a table border with the specified characters.
func (ui *TerminalUI) printTableBorder(colWidths []int, left, middle, right string) {
	fmt.Fprint(ui.output, left)
	for i, width := range colWidths {
		fmt.Fprint(ui.output, strings.Repeat("─", width+2))
		if i < len(colWidths)-1 {
			fmt.Fprint(ui.output, middle)
		}
	}
	fmt.Fprintln(ui.output, right)
}

// ShowProgress creates and returns a new progress bar.
func (ui *TerminalUI) ShowProgress(title string, total int) interfaces.ProgressBar {
	return NewProgressBar(title, total, ui.output, ui.style)
}

// NewProgressBar creates a new progress bar instance.
func NewProgressBar(title string, total int, output io.Writer, style interfaces.UIStyle) *ProgressBarImpl {
	return &ProgressBarImpl{
		title:    title,
		total:    total,
		current:  0,
		output:   output,
		style:    style,
		finished: false,
	}
}

// Update updates the progress bar with the current value.
func (pb *ProgressBarImpl) Update(current int) error {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	return pb.updateInternal(current)
}

// updateInternal updates the progress bar without acquiring the mutex.
// This method assumes the mutex is already held by the caller.
func (pb *ProgressBarImpl) updateInternal(current int) error {
	if pb.finished {
		return nil
	}

	pb.current = current
	if pb.current > pb.total {
		pb.current = pb.total
	}

	// Calculate percentage (handle negative values and zero total)
	displayCurrent := pb.current
	if displayCurrent < 0 {
		displayCurrent = 0
	}

	percentage := float64(displayCurrent) / float64(pb.total) * 100
	if pb.total == 0 {
		percentage = 100
	}

	// Create progress bar visual
	barWidth := 40
	filledWidth := int(float64(barWidth) * percentage / 100)
	if filledWidth < 0 {
		filledWidth = 0
	}
	emptyWidth := barWidth - filledWidth

	progressBar := fmt.Sprintf("[%s%s]",
		strings.Repeat("█", filledWidth),
		strings.Repeat("░", emptyWidth))

	// Print progress
	fmt.Fprintf(pb.output, "\r%s %s %.1f%% (%d/%d)",
		pb.style.Info.Sprint(pb.title),
		pb.style.Primary.Sprint(progressBar),
		percentage,
		pb.current,
		pb.total)

	if pb.message != "" {
		fmt.Fprintf(pb.output, " %s", pb.style.Highlight.Sprint(pb.message))
	}

	return nil
}

// SetMessage sets the progress message.
func (pb *ProgressBarImpl) SetMessage(message string) error {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	pb.message = message
	return pb.updateInternal(pb.current)
}

// SetTotal sets the total value.
func (pb *ProgressBarImpl) SetTotal(total int) error {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	pb.total = total
	return pb.updateInternal(pb.current)
}

// Finish completes the progress bar.
func (pb *ProgressBarImpl) Finish() error {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	if pb.finished {
		return nil
	}

	pb.finished = true
	pb.current = pb.total

	// Show completed progress bar
	progressBar := fmt.Sprintf("[%s]", strings.Repeat("█", 40))
	fmt.Fprintf(pb.output, "\r%s %s %s (%d/%d)     \n",
		pb.style.Success.Sprint("✅"),
		pb.style.Primary.Sprint(pb.title),
		pb.style.Success.Sprint(progressBar),
		pb.current,
		pb.total)

	return nil
}

// ShowSpinner displays a spinning indicator for long-running operations.
func (ui *TerminalUI) ShowSpinner(message string, done <-chan bool) {
	spinnerIndex := 0
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			// Clear the spinner line
			fmt.Fprintf(ui.output, "\r%s\r", strings.Repeat(" ", len(message)+10))
			return
		case <-ticker.C:
			fmt.Fprintf(ui.output, "\r%s %s",
				ui.style.Primary.Sprint(ui.spinners[spinnerIndex]),
				ui.style.Info.Sprint(message))
			spinnerIndex = (spinnerIndex + 1) % len(ui.spinners)
		}
	}
}

// PrintSeparator prints a visual separator line.
func (ui *TerminalUI) PrintSeparator() {
	separator := strings.Repeat("─", 60)
	fmt.Fprintln(ui.output, ui.style.Highlight.Sprint(separator))
}

// PrintHeader prints a section header with styling.
func (ui *TerminalUI) PrintHeader(title string) {
	fmt.Fprintln(ui.output)
	fmt.Fprintf(ui.output, "╭─ %s ─%s╮\n",
		ui.style.Primary.Sprint(title),
		strings.Repeat("─", max(0, 50-len(title))))
	fmt.Fprintln(ui.output)
}

// PrintSubHeader prints a subsection header.
func (ui *TerminalUI) PrintSubHeader(title string) {
	fmt.Fprintf(ui.output, "\n%s %s\n",
		ui.style.Info.Sprint("▶"),
		ui.style.Info.Sprint(title))
}

// ClearLine clears the current line.
func (ui *TerminalUI) ClearLine() {
	fmt.Fprint(ui.output, "\r\033[K")
}

// MoveCursorUp moves the cursor up by n lines.
func (ui *TerminalUI) MoveCursorUp(n int) {
	fmt.Fprintf(ui.output, "\033[%dA", n)
}

// MoveCursorDown moves the cursor down by n lines.
func (ui *TerminalUI) MoveCursorDown(n int) {
	fmt.Fprintf(ui.output, "\033[%dB", n)
}

// HideCursor hides the terminal cursor.
func (ui *TerminalUI) HideCursor() {
	fmt.Fprint(ui.output, "\033[?25l")
}

// ShowCursor shows the terminal cursor.
func (ui *TerminalUI) ShowCursor() {
	fmt.Fprint(ui.output, "\033[?25h")
}

// GetTerminalWidth attempts to get the terminal width.
func (ui *TerminalUI) GetTerminalWidth() int {
	// Default width if we can't determine it
	return 80
}

// max returns the maximum of two integers.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// GetStyle returns the current UI style.
func (ui *TerminalUI) GetStyle() interfaces.UIStyle {
	ui.mu.RLock()
	defer ui.mu.RUnlock()
	return ui.style
}

// SetStyle sets the UI style.
func (ui *TerminalUI) SetStyle(style interfaces.UIStyle) {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	ui.style = style
}

// IsVerbose returns whether verbose mode is enabled.
func (ui *TerminalUI) IsVerbose() bool {
	ui.mu.RLock()
	defer ui.mu.RUnlock()
	return ui.verbose
}

// SetVerbose sets verbose mode.
func (ui *TerminalUI) SetVerbose(verbose bool) {
	ui.mu.Lock()
	defer ui.mu.Unlock()
	ui.verbose = verbose
}
