package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"vpn-tool/pkg/cmdutil"
)

// AuthData represents the structure of auth.json
type AuthData struct {
	Version int    `json:"version"`
	PinHash string `json:"pin_hash"`
}

// NewCmdAuth creates the auth command group with all subcommands.
func NewCmdAuth() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "auth <command>",
		Short:   "Authentication commands",
		GroupID: "core",
		Long: heredoc.Doc(`
			Manage local PIN authentication and session status for vpn-tool.
		`),
	}

	cmd.AddCommand(NewCmdAuthRegister())
	cmd.AddCommand(NewCmdAuthLogin())
	cmd.AddCommand(NewCmdAuthLogout())
	cmd.AddCommand(NewCmdAuthStatus())
	cmd.AddCommand(NewCmdAuthReset())
	cmd.AddCommand(NewCmdAuthRemove())

	return cmd
}

// NewCmdAuthRegister handles PIN registration
func NewCmdAuthRegister() *cobra.Command {
	return &cobra.Command{
		Use:   "register",
		Short: "Register a new PIN",
		Example: heredoc.Doc(`
			$ vpn-tool auth register
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if registered() {
				return fmt.Errorf("PIN already registered.")
			}

			pin, err := cmdutil.PromptPassword("Enter new PIN: ")
			if err != nil {
				return err
			}
			if pin == "" {
				return fmt.Errorf("PIN cannot be empty.")
			}

			confirm, err := cmdutil.PromptPassword("Confirm new PIN: ")
			if err != nil {
				return err
			}
			if pin != confirm {
				return fmt.Errorf("PINs do not match.")
			}

			if err := ensureConfig(); err != nil {
				return err
			}

			authData := AuthData{
				Version: 1,
				PinHash: hashPin(pin),
			}

			data, err := json.MarshalIndent(authData, "", "  ")
			if err != nil {
				return err
			}

			authFile, sessionFile := getPaths()
			if err := os.WriteFile(authFile, data, 0600); err != nil {
				return err
			}
			if err := os.WriteFile(sessionFile, []byte("logged_out"), 0600); err != nil {
				return err
			}

			fmt.Println("PIN registered successfully.")
			fmt.Println("Run: vpn-tool auth login")
			return nil
		},
	}
}

// NewCmdAuthLogin handles logging in with a PIN
func NewCmdAuthLogin() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Login with PIN",
		Example: heredoc.Doc(`
			$ vpn-tool auth login
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !registered() {
				return fmt.Errorf("No PIN registered. Run: vpn-tool auth register")
			}

			authData := loadAuth()
			pin, err := cmdutil.PromptPassword("Enter PIN: ")
			if err != nil {
				return err
			}

			if authData == nil || authData.PinHash != hashPin(pin) {
				return fmt.Errorf("Invalid PIN.")
			}

			if err := ensureConfig(); err != nil {
				return err
			}

			_, sessionFile := getPaths()
			if err := os.WriteFile(sessionFile, []byte("logged_in"), 0600); err != nil {
				return err
			}

			fmt.Println("Login successful.")
			return nil
		},
	}
}

// NewCmdAuthLogout handles logging out
func NewCmdAuthLogout() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Logout",
		Example: heredoc.Doc(`
			$ vpn-tool auth logout
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !registered() {
				return fmt.Errorf("No PIN registered.")
			}

			authData := loadAuth()
			pin, err := cmdutil.PromptPassword("Enter PIN: ")
			if err != nil {
				return err
			}

			if authData == nil || authData.PinHash != hashPin(pin) {
				return fmt.Errorf("Invalid PIN.")
			}

			_, sessionFile := getPaths()
			if err := os.WriteFile(sessionFile, []byte("logged_out"), 0600); err != nil {
				return err
			}

			fmt.Println("Logout successful.")
			return nil
		},
	}
}

// NewCmdAuthStatus displays the current authentication and session status
func NewCmdAuthStatus() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show authentication status",
		Example: heredoc.Doc(`
			$ vpn-tool auth status
		`),
		Run: func(cmd *cobra.Command, args []string) {
			regStr := "no"
			if registered() {
				regStr = "yes"
			}
			logStr := "no"
			if loggedIn() {
				logStr = "yes"
			}

			fmt.Printf("Registered: %s\n", regStr)
			fmt.Printf("Logged in: %s\n", logStr)
		},
	}
}

// NewCmdAuthReset handles PIN reset
func NewCmdAuthReset() *cobra.Command {
	return &cobra.Command{
		Use:   "reset",
		Short: "Reset PIN",
		Example: heredoc.Doc(`
			$ vpn-tool auth reset
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !registered() {
				return fmt.Errorf("No PIN registered.")
			}

			authData := loadAuth()
			current, err := cmdutil.PromptPassword("Enter current PIN: ")
			if err != nil {
				return err
			}

			if authData == nil || authData.PinHash != hashPin(current) {
				return fmt.Errorf("Invalid current PIN.")
			}

			newPin, err := cmdutil.PromptPassword("Enter new PIN: ")
			if err != nil {
				return err
			}
			if newPin == "" {
				return fmt.Errorf("New PIN cannot be empty.")
			}

			confirm, err := cmdutil.PromptPassword("Confirm new PIN: ")
			if err != nil {
				return err
			}
			if newPin != confirm {
				return fmt.Errorf("PINs do not match.")
			}

			updatedAuth := AuthData{
				Version: 1,
				PinHash: hashPin(newPin),
			}

			data, err := json.MarshalIndent(updatedAuth, "", "  ")
			if err != nil {
				return err
			}

			authFile, sessionFile := getPaths()
			if err := os.WriteFile(authFile, data, 0600); err != nil {
				return err
			}
			if err := os.WriteFile(sessionFile, []byte("logged_out"), 0600); err != nil {
				return err
			}

			fmt.Println("PIN reset successfully.")
			return nil
		},
	}
}

// NewCmdAuthRemove removes authentication files completely
func NewCmdAuthRemove() *cobra.Command {
	return &cobra.Command{
		Use:   "remove",
		Short: "Remove authentication",
		Example: heredoc.Doc(`
			$ vpn-tool auth remove
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !registered() {
				return fmt.Errorf("No PIN registered.")
			}

			authData := loadAuth()
			pin, err := cmdutil.PromptPassword("Enter PIN: ")
			if err != nil {
				return err
			}

			if authData == nil || authData.PinHash != hashPin(pin) {
				return fmt.Errorf("Invalid PIN.")
			}

			authFile, sessionFile := getPaths()
			_ = os.Remove(authFile)
			_ = os.Remove(sessionFile)

			fmt.Println("Authentication removed.")
			return nil
		},
	}
}

// --- Helper Functions ---

func getPaths() (string, string) {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	configDir := filepath.Join(home, ".vpn-tool")
	return filepath.Join(configDir, "auth.json"), filepath.Join(configDir, "session")
}

func ensureConfig() error {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	configDir := filepath.Join(home, ".vpn-tool")
	return os.MkdirAll(configDir, 0700)
}

func hashPin(pin string) string {
	hasher := sha256.New()
	hasher.Write([]byte(pin))
	return hex.EncodeToString(hasher.Sum(nil))
}

func registered() bool {
	authFile, _ := getPaths()
	_, err := os.Stat(authFile)
	return err == nil
}

func loggedIn() bool {
	_, sessionFile := getPaths()
	data, err := os.ReadFile(sessionFile)
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(data)) == "logged_in"
}

func loadAuth() *AuthData {
	if !registered() {
		return nil
	}
	authFile, _ := getPaths()
	data, err := os.ReadFile(authFile)
	if err != nil {
		return nil
	}

	var authData AuthData
	if err := json.Unmarshal(data, &authData); err != nil {
		return nil
	}
	return &authData
}

func requireLogin() error {
	if !registered() {
		return fmt.Errorf("Not registered. Run: vpn-tool auth register")
	}

	if !loggedIn() {
		return fmt.Errorf("Not logged in. Run: vpn-tool auth login")
	}
	return nil
}
