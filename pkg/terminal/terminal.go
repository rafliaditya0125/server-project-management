package terminal

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"golang.org/x/term"
)

const (
	ColorRed    = "\033[0;31m"
	ColorGreen  = "\033[0;32m"
	ColorYellow = "\033[1;33m"
	ColorBlue   = "\033[0;34m"
	ColorCyan   = "\033[0;36m"
	ColorBold   = "\033[1m"
	ColorReset  = "\033[0m"
)

// ReadPrompt reads a line from stdin with a prompt and default value.
func ReadPrompt(prompt, defaultVal string) string {
	if defaultVal != "" {
		fmt.Printf("%s [default: %s]: ", prompt, defaultVal)
	} else {
		fmt.Printf("%s: ", prompt)
	}

	reader := bufio.NewReader(os.Stdin)
	text, err := reader.ReadString('\n')
	if err != nil {
		return defaultVal
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return defaultVal
	}
	return text
}

// ReadPassword prompts the user to enter a secret password without echoing.
func ReadPassword(prompt string) (string, error) {
	fmt.Print(prompt)
	// Check if stdin is a terminal
	fd := int(syscall.Stdin)
	if term.IsTerminal(fd) {
		bytePassword, err := term.ReadPassword(fd)
		fmt.Println()
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(bytePassword)), nil
	}

	// Fallback if not a terminal (e.g. piped input)
	reader := bufio.NewReader(os.Stdin)
	text, err := reader.ReadString('\n')
	fmt.Println()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(text), nil
}

// PadRight pads string to given length.
func PadRight(str string, length int) string {
	if len(str) >= length {
		return str[:length]
	}
	return str + strings.Repeat(" ", length-len(str))
}

// Confirm prompts user with yes/no.
func Confirm(prompt string, defaultYes bool) bool {
	var suffix string
	if defaultYes {
		suffix = " (Y/n): "
	} else {
		suffix = " (y/N): "
	}

	fmt.Print(prompt + suffix)
	reader := bufio.NewReader(os.Stdin)
	text, err := reader.ReadString('\n')
	if err != nil {
		return defaultYes
	}
	text = strings.TrimSpace(strings.ToLower(text))
	if text == "" {
		return defaultYes
	}
	return text == "y" || text == "yes"
}
