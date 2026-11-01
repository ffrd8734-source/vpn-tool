package cmdutil

import (
	"fmt"
	"strings"
	"syscall"

	"golang.org/x/term"
)

// PromptPassword securely prompts the user for a PIN or password without echoing input to the terminal.
func PromptPassword(prompt string) (string, error) {
	fmt.Print(prompt)
	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println() // Print a newline after hidden input
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(bytePassword)), nil
}
