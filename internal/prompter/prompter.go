package prompter

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

// Prompter handles interactive user input
type Prompter struct {
	reader *bufio.Reader
}

// New creates a new Prompter
func New() *Prompter {
	return &Prompter{
		reader: bufio.NewReader(os.Stdin),
	}
}

// PromptString prompts for a string input with an optional default value
func (p *Prompter) PromptString(prompt, defaultValue string) (string, error) {
	if defaultValue != "" {
		fmt.Printf("%s [%s]: ", prompt, defaultValue)
	} else {
		fmt.Printf("%s: ", prompt)
	}

	input, err := p.reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}

	input = strings.TrimSpace(input)
	if input == "" {
		return defaultValue, nil
	}
	return input, nil
}

// PromptPassword prompts for a password (hidden input)
func (p *Prompter) PromptPassword(prompt string) (string, error) {
	fmt.Printf("%s: ", prompt)

	// Read password without echoing
	passwordBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println() // Print newline after password input

	if err != nil {
		return "", fmt.Errorf("failed to read password: %w", err)
	}

	return string(passwordBytes), nil
}

// PromptSelect prompts the user to select from a list of options
// Returns the index of the selected option
func (p *Prompter) PromptSelect(prompt string, options []string) (int, error) {
	fmt.Println(prompt)
	for i, opt := range options {
		fmt.Printf("  [%d] %s\n", i+1, opt)
	}
	fmt.Print("Selection: ")

	input, err := p.reader.ReadString('\n')
	if err != nil {
		return -1, fmt.Errorf("failed to read input: %w", err)
	}

	input = strings.TrimSpace(input)
	selection, err := strconv.Atoi(input)
	if err != nil {
		return -1, fmt.Errorf("invalid selection: %s", input)
	}

	if selection < 1 || selection > len(options) {
		return -1, fmt.Errorf("selection out of range: %d (must be 1-%d)", selection, len(options))
	}

	return selection - 1, nil // Return 0-based index
}

// PromptConfirm prompts for a yes/no confirmation
func (p *Prompter) PromptConfirm(prompt string, defaultYes bool) (bool, error) {
	var hint string
	if defaultYes {
		hint = "[Y/n]"
	} else {
		hint = "[y/N]"
	}

	fmt.Printf("%s %s: ", prompt, hint)

	input, err := p.reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read input: %w", err)
	}

	input = strings.TrimSpace(strings.ToLower(input))
	if input == "" {
		return defaultYes, nil
	}

	switch input {
	case "y", "yes":
		return true, nil
	case "n", "no":
		return false, nil
	default:
		return false, fmt.Errorf("invalid input: %s (expected y/n)", input)
	}
}

// Package-level convenience functions using a default Prompter

var defaultPrompter = New()

// String prompts for a string input
func String(prompt, defaultValue string) (string, error) {
	return defaultPrompter.PromptString(prompt, defaultValue)
}

// Password prompts for a password
func Password(prompt string) (string, error) {
	return defaultPrompter.PromptPassword(prompt)
}

// Select prompts for selection from options
func Select(prompt string, options []string) (int, error) {
	return defaultPrompter.PromptSelect(prompt, options)
}

// Confirm prompts for yes/no confirmation
func Confirm(prompt string, defaultYes bool) (bool, error) {
	return defaultPrompter.PromptConfirm(prompt, defaultYes)
}
