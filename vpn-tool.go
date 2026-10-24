package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const Version = "1.0.1"
const ApiBase = "https://vpn-gateway-manager--jjajajjajahsh.replit.app"

type AuthProfile struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	PinHash   string `json:"pin_hash"`
	CreatedAt string `json:"created_at"`
}

type AuthStore struct {
	Version       int           `json:"version"`
	ActiveProfile string        `json:"active_profile"`
	Profiles      []AuthProfile `json:"profiles"`
}

func getConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
		os.Exit(1)
	}
	return filepath.Join(home, ".vpn-tool")
}

func getAuthFile() string {
	return filepath.Join(getConfigDir(), "auth.json")
}

func getSessionFile() string {
	return filepath.Join(getConfigDir(), "session")
}

func getConfigFile() string {
	return filepath.Join(getConfigDir(), "config.json")
}

func getAliasFile() string {
	return filepath.Join(getConfigDir(), "aliases.json")
}

func ensureConfig() {
	dir := getConfigDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating config directory: %v\n", err)
		os.Exit(1)
	}
}

func hashPin(pin string) string {
	hasher := sha256.New()
	hasher.Write([]byte(pin))
	return hex.EncodeToString(hasher.Sum(nil))
}

func loadStore() *AuthStore {
	file := getAuthFile()
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return nil
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return nil
	}
	var store AuthStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil
	}
	if store.Profiles == nil {
		store.Profiles = []AuthProfile{}
	}
	return &store
}

func saveStore(store *AuthStore) {
	ensureConfig()
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error saving auth store: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(getAuthFile(), data, 0600); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing auth file: %v\n", err)
		os.Exit(1)
	}
}

func registered() bool {
	store := loadStore()
	return store != nil && len(store.Profiles) > 0
}

func loggedIn() bool {
	file := getSessionFile()
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return false
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(data)) == "logged_in"
}

func setLoggedIn(status bool) {
	ensureConfig()
	val := "logged_out"
	if status {
		val = "logged_in"
	}
	_ = os.WriteFile(getSessionFile(), []byte(val), 0600)
}

func prompt(promptStr string, secure bool) string {
	fmt.Print(promptStr)
	if secure {
		bytes, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(bytes))
	}
	var res string
	_, _ = fmt.Scanln(&res)
	return strings.TrimSpace(res)
}

func requireLogin() {
	if !registered() {
		fmt.Fprintln(os.Stderr, "Not registered. Run: vpn-tool auth register")
		os.Exit(1)
	}
	if !loggedIn() {
		fmt.Fprintln(os.Stderr, "Not logged in. Run: vpn-tool auth login")
		os.Exit(1)
	}
}

func apiRequest(method, endpoint string) {
	client := &http.Client{}
	req, err := http.NewRequest(method, ApiBase+endpoint, nil)
	if err != nil {
		printError(0, err.Error())
		return
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		printError(0, err.Error())
		return
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	var bodyJSON interface{}
	if err := json.Unmarshal(bodyBytes, &bodyJSON); err != nil {
		bodyJSON = string(bodyBytes)
	}

	out, _ := json.MarshalIndent(map[string]interface{}{
		"status": resp.StatusCode,
		"body":   bodyJSON,
	}, "", "  ")
	fmt.Println(string(out))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		os.Exit(1)
	}
}

func printError(status int, msg string) {
	out, _ := json.MarshalIndent(map[string]interface{}{
		"status": status,
		"body":   map[string]string{"error": msg},
	}, "", "  ")
	fmt.Println(string(out))
	os.Exit(1)
}

func main() {
	// Alias Expansion: intercept os.Args[1] before Cobra processes it
	aliasFile := getAliasFile()
	if data, err := os.ReadFile(aliasFile); err == nil {
		var aliases map[string]string
		if json.Unmarshal(data, &aliases) == nil {
			if len(os.Args) > 1 {
				if target, exists := aliases[os.Args[1]]; exists {
					fields := strings.Fields(target)
					newArgs := make([]string, 0, len(fields)+len(os.Args)-1)
					newArgs = append(newArgs, os.Args[0])
					newArgs = append(newArgs, fields...)
					if len(os.Args) > 2 {
						newArgs = append(newArgs, os.Args[2:]...)
					}
					os.Args = newArgs
				}
			}
		}
	}

	var rootCmd = &cobra.Command{
		Use:     "vpn-tool",
		Short:   "VPN Gateway Manager CLI",
		Version: Version,
	}

	// Auth Subcommand
	var authCmd = &cobra.Command{
		Use:   "auth",
		Short: "Authentication commands",
	}

	authCmd.AddCommand(&cobra.Command{
		Use:   "register",
		Short: "Register a new PIN profile",
		Run: func(cmd *cobra.Command, args []string) {
			store := loadStore()
			if store == nil {
				store = &AuthStore{Version: 1, Profiles: []AuthProfile{}}
			}

			pin := prompt("Enter new PIN: ", true)
			if pin == "" {
				fmt.Fprintln(os.Stderr, "PIN cannot be empty.")
				os.Exit(1)
			}
			confirm := prompt("Confirm new PIN: ", true)
			if pin != confirm {
				fmt.Fprintln(os.Stderr, "PINs do not match.")
				os.Exit(1)
			}

			label := prompt("Enter profile label (optional): ", false)
			if label == "" {
				label = fmt.Sprintf("profile-%d", len(store.Profiles)+1)
			}

			profileID := fmt.Sprintf("id-%d", len(store.Profiles)+1)
			newProfile := AuthProfile{
				ID:        profileID,
				Label:     label,
				PinHash:   hashPin(pin),
				CreatedAt: fmt.Sprintf("%d", os.Getpid()),
			}

			store.Profiles = append(store.Profiles, newProfile)
			store.ActiveProfile = profileID
			saveStore(store)
			setLoggedIn(false)

			fmt.Printf("PIN registered successfully with profile '%s' (%s).\n", label, profileID)
			fmt.Println("Run: vpn-tool auth login")
		},
	})

	authCmd.AddCommand(&cobra.Command{
		Use:   "login",
		Short: "Login with PIN",
		Run: func(cmd *cobra.Command, args []string) {
			store := loadStore()
			if store == nil || len(store.Profiles) == 0 {
				fmt.Fprintln(os.Stderr, "No PIN registered. Run: vpn-tool auth register")
				os.Exit(1)
			}

			pin := prompt("Enter PIN: ", true)
			hashed := hashPin(pin)

			var matched *AuthProfile
			for _, p := range store.Profiles {
				if p.PinHash == hashed {
					matched = &p
					break
				}
			}

			if matched == nil {
				fmt.Fprintln(os.Stderr, "Invalid PIN.")
				os.Exit(1)
			}

			store.ActiveProfile = matched.ID
			saveStore(store)
			setLoggedIn(true)
			fmt.Printf("Login successful as profile '%s'.\n", matched.Label)
		},
	})

	authCmd.AddCommand(&cobra.Command{
		Use:   "logout",
		Short: "Logout",
		Run: func(cmd *cobra.Command, args []string) {
			if !registered() {
				fmt.Fprintln(os.Stderr, "No PIN registered.")
				os.Exit(1)
			}
			store := loadStore()
			pin := prompt("Enter PIN: ", true)
			hashed := hashPin(pin)

			var matched bool
			for _, p := range store.Profiles {
				if p.ID == store.ActiveProfile && p.PinHash == hashed {
					matched = true
					break
				}
			}

			if !matched {
				fmt.Fprintln(os.Stderr, "Invalid PIN or active profile mismatch.")
				os.Exit(1)
			}

			setLoggedIn(false)
			fmt.Println("Logout successful.")
		},
	})

	authCmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Show authentication status",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Registered: %s\n", map[bool]string{true: "yes", false: "no"}[registered()])
			fmt.Printf("Logged in: %s\n", map[bool]string{true: "yes", false: "no"}[loggedIn()])
		},
	})

	authCmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List all registered auth profiles",
		Run: func(cmd *cobra.Command, args []string) {
			store := loadStore()
			if store == nil || len(store.Profiles) == 0 {
				fmt.Println("No profiles registered.")
				return
			}
			fmt.Println("Registered Auth Profiles:")
			for _, p := range store.Profiles {
				activeMarker := " "
				if p.ID == store.ActiveProfile {
					activeMarker = "*"
				}
				fmt.Printf("%s [%s] Label: %s\n", activeMarker, p.ID, p.Label)
			}
		},
	})

	authCmd.AddCommand(&cobra.Command{
		Use:   "whoami",
		Short: "Show current active auth profile metadata",
		Run: func(cmd *cobra.Command, args []string) {
			store := loadStore()
			if store == nil || store.ActiveProfile == "" {
				fmt.Println("No active profile.")
				return
			}
			for _, p := range store.Profiles {
				if p.ID == store.ActiveProfile {
					out, _ := json.MarshalIndent(p, "", "  ")
					fmt.Println(string(out))
					return
				}
			}
			fmt.Println("Active profile not found in store.")
		},
	})

	authCmd.AddCommand(&cobra.Command{
		Use:   "switch [profile_id]",
		Short: "Switch active auth profile",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			targetID := args[0]
			store := loadStore()
			if store == nil {
				fmt.Fprintln(os.Stderr, "No profiles registered.")
				os.Exit(1)
			}

			found := false
			for _, p := range store.Profiles {
				if p.ID == targetID || p.Label == targetID {
					store.ActiveProfile = p.ID
					found = true
					break
				}
			}

			if !found {
				fmt.Fprintf(os.Stderr, "Profile '%s' not found.\n", targetID)
				os.Exit(1)
			}

			saveStore(store)
			setLoggedIn(false)
			fmt.Printf("Switched active profile to %s. Please login.\n", targetID)
		},
	})

	authCmd.AddCommand(&cobra.Command{
		Use:   "reset",
		Short: "Reset PIN for active profile",
		Run: func(cmd *cobra.Command, args []string) {
			store := loadStore()
			if store == nil || len(store.Profiles) == 0 {
				fmt.Fprintln(os.Stderr, "No PIN registered.")
				os.Exit(1)
			}

			current := prompt("Enter current PIN: ", true)
			hashed := hashPin(current)

			var index = -1
			for i, p := range store.Profiles {
				if p.ID == store.ActiveProfile && p.PinHash == hashed {
					index = i
					break
				}
			}

			if index == -1 {
				fmt.Fprintln(os.Stderr, "Invalid current PIN.")
				os.Exit(1)
			}

			newPin := prompt("Enter new PIN: ", true)
			if newPin == "" {
				fmt.Fprintln(os.Stderr, "New PIN cannot be empty.")
				os.Exit(1)
			}
			confirm := prompt("Confirm new PIN: ", true)
			if newPin != confirm {
				fmt.Fprintln(os.Stderr, "PINs do not match.")
				os.Exit(1)
			}

			store.Profiles[index].PinHash = hashPin(newPin)
			saveStore(store)
			setLoggedIn(false)
			fmt.Println("PIN reset successfully.")
		},
	})

	authCmd.AddCommand(&cobra.Command{
		Use:   "remove",
		Short: "Remove authentication configuration",
		Run: func(cmd *cobra.Command, args []string) {
			if !registered() {
				fmt.Fprintln(os.Stderr, "No PIN registered.")
				os.Exit(1)
			}
			_ = os.Remove(getAuthFile())
			_ = os.Remove(getSessionFile())
			fmt.Println("Authentication removed.")
		},
	})

	rootCmd.AddCommand(authCmd)

	// Config Subcommand
	var configCmd = &cobra.Command{
		Use:   "config",
		Short: "Configuration commands",
	}

	var globalFlag bool
	configCmd.PersistentFlags().BoolVarP(&globalFlag, "global", "g", false, "Use global config file.")

	configCmd.AddCommand(&cobra.Command{
		Use:   "edit",
		Short: "Opens the configuration file.",
		Run: func(cmd *cobra.Command, args []string) {
			ensureConfig()
			file := getConfigFile()
			if _, err := os.Stat(file); os.IsNotExist(err) {
				_ = os.WriteFile(file, []byte("{}"), 0600)
			}
			editor := os.Getenv("EDITOR")
			if editor == "" {
				editor = "vi"
			}
			c := exec.Command(editor, file)
			c.Stdin = os.Stdin
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			_ = c.Run()
		},
	})

	configCmd.AddCommand(&cobra.Command{
		Use:   "get <key>",
		Short: "Prints the value of a given configuration key.",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			key := args[0]
			file := getConfigFile()
			if _, err := os.Stat(file); os.IsNotExist(err) {
				fmt.Fprintln(os.Stderr, "Config file not found.")
				os.Exit(1)
			}
			data, err := os.ReadFile(file)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading config: %v\n", err)
				os.Exit(1)
			}
			var cfg map[string]interface{}
			if err := json.Unmarshal(data, &cfg); err != nil {
				fmt.Fprintln(os.Stderr, "Error parsing config JSON.")
				os.Exit(1)
			}
			val, ok := cfg[key]
			if !ok {
				fmt.Fprintf(os.Stderr, "Key '%s' not found.\n", key)
				os.Exit(1)
			}
			fmt.Println(val)
		},
	})

	configCmd.AddCommand(&cobra.Command{
		Use:   "set <key> <value>",
		Short: "Updates configuration with the value of a given key.",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			key, val := args[0], args[1]
			ensureConfig()
			file := getConfigFile()
			cfg := make(map[string]interface{})
			if data, err := os.ReadFile(file); err == nil {
				_ = json.Unmarshal(data, &cfg)
			}
			cfg[key] = val
			out, _ := json.MarshalIndent(cfg, "", "  ")
			_ = os.WriteFile(file, out, 0600)
			fmt.Println("Configuration updated successfully.")
		},
	})

	configCmd.AddCommand(&cobra.Command{
		Use:   "delete <key>",
		Short: "Delete a configuration key.",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			key := args[0]
			file := getConfigFile()
			if _, err := os.Stat(file); os.IsNotExist(err) {
				fmt.Fprintln(os.Stderr, "Config file not found.")
				os.Exit(1)
			}
			data, err := os.ReadFile(file)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error reading config.")
				os.Exit(1)
			}
			var cfg map[string]interface{}
			if err := json.Unmarshal(data, &cfg); err != nil {
				fmt.Fprintln(os.Stderr, "Error parsing config JSON.")
				os.Exit(1)
			}
			if _, ok := cfg[key]; !ok {
				fmt.Fprintf(os.Stderr, "Key '%s' not found.\n", key)
				os.Exit(1)
			}
			delete(cfg, key)
			out, _ := json.MarshalIndent(cfg, "", "  ")
			_ = os.WriteFile(file, out, 0600)
			fmt.Printf("Key '%s' deleted successfully.\n", key)
		},
	})

	rootCmd.AddCommand(configCmd)

	// Alias Subcommand
	var aliasCmd = &cobra.Command{
		Use:   "alias",
		Short: "Command alias management",
	}

	aliasCmd.AddCommand(&cobra.Command{
		Use:   "set <alias name> '<command>'",
		Short: "Set an alias for a longer command.",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			name, commandStr := args[0], args[1]
			ensureConfig()
			file := getAliasFile()
			aliases := make(map[string]string)
			if data, err := os.ReadFile(file); err == nil {
				_ = json.Unmarshal(data, &aliases)
			}
			aliases[name] = commandStr
			out, _ := json.MarshalIndent(aliases, "", "  ")
			_ = os.WriteFile(file, out, 0600)
			fmt.Printf("Alias '%s' set to '%s'.\n", name, commandStr)
		},
	})

	aliasCmd.AddCommand(&cobra.Command{
		Use:   "delete <alias name>",
		Short: "Delete an alias.",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]
			file := getAliasFile()
			if _, err := os.Stat(file); os.IsNotExist(err) {
				fmt.Fprintln(os.Stderr, "No aliases configured.")
				os.Exit(1)
			}
			data, _ := os.ReadFile(file)
			aliases := make(map[string]string)
			_ = json.Unmarshal(data, &aliases)
			if _, ok := aliases[name]; !ok {
				fmt.Fprintf(os.Stderr, "Alias '%s' not found.\n", name)
				os.Exit(1)
			}
			delete(aliases, name)
			out, _ := json.MarshalIndent(aliases, "", "  ")
			_ = os.WriteFile(file, out, 0600)
			fmt.Printf("Alias '%s' deleted.\n", name)
		},
	})

	aliasCmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List aliases.",
		Run: func(cmd *cobra.Command, args []string) {
			file := getAliasFile()
			if _, err := os.Stat(file); os.IsNotExist(file) { // note: os.IsNotExist(err) handled fine
				fmt.Println("No aliases configured.")
				return
			}
			data, err := os.ReadFile(file)
			if err != nil {
				fmt.Println("No aliases configured.")
				return
			}
			aliases := make(map[string]string)
			_ = json.Unmarshal(data, &aliases)
			if len(aliases) == 0 {
				fmt.Println("No aliases configured.")
				return
			}
			fmt.Println("Configured Aliases:")
			for k, v := range aliases {
				fmt.Printf("%s = %s\n", k, v)
			}
		},
	})

	rootCmd.AddCommand(aliasCmd)

	// API Commands
	rootCmd.AddCommand(&cobra.Command{
		Use:   "healthz",
		Short: "Check API health",
		Run: func(cmd *cobra.Command, args []string) {
			requireLogin()
			apiRequest("GET", "/api/healthz")
		},
	})

	rootCmd.AddCommand(&cobra.Command{
		Use:   "countries",
		Short: "List VPN countries",
		Run: func(cmd *cobra.Command, args []string) {
			requireLogin()
			apiRequest("GET", "/api/vpn/countries")
		},
	})

	for _, action := range []string{"create", "connect", "disconnect"} {
		c := &cobra.Command{
			Use:   action,
			Short: fmt.Sprintf("%s VPN", action),
			Run: func(cmd *cobra.Command, args []string) {
				requireLogin()
				country, _ := cmd.Flags().GetString("country")
				apiRequest("POST", fmt.Sprintf("/api/vpn/%s?countryCode=%s", action, country))
			},
		}
		c.Flags().String("country", "", "Country code (required)")
		_ = c.MarkFlagRequired("country")
		rootCmd.AddCommand(c)
	}

	var statusCmd = &cobra.Command{
		Use:   "status",
		Short: "Get VPN status",
		Run: func(cmd *cobra.Command, args []string) {
			requireLogin()
			country, _ := cmd.Flags().GetString("country")
			apiRequest("GET", fmt.Sprintf("/api/vpn/status?countryCode=%s", country))
		},
	}
	statusCmd.Flags().String("country", "", "Country code (required)")
	_ = statusCmd.MarkFlagRequired("country")
	rootCmd.AddCommand(statusCmd)

	var testCmd = &cobra.Command{
		Use:   "test",
		Short: "Test API delay",
		Run: func(cmd *cobra.Command, args []string) {
			requireLogin()
			ms, _ := cmd.Flags().GetString("ms")
			apiRequest("GET", fmt.Sprintf("/api/vpn/test/sleep?ms=%s", ms))
		},
	}
	testCmd.Flags().String("ms", "100", "Milliseconds")
	rootCmd.AddCommand(testCmd)

	var ipCmd = &cobra.Command{
		Use:   "ip",
		Short: "Get VPN IP addresses",
		Run: func(cmd *cobra.Command, args []string) {
			requireLogin()
			country, _ := cmd.Flags().GetString("country")
			count, _ := cmd.Flags().GetString("count")
			apiRequest("POST", fmt.Sprintf("/api/vpn/ip?countryCode=%s&count=%s", country, count))
		},
	}
	ipCmd.Flags().String("country", "", "Country code (required)")
	ipCmd.Flags().String("count", "10", "Number of IPs")
	_ = ipCmd.MarkFlagRequired("country")
	rootCmd.AddCommand(ipCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
