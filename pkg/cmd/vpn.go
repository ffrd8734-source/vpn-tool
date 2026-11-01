package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
)

const API_BASE = "https://vpn-gateway-manager--jjajajjajahsh.replit.app"

// NewCmdCountries creates the countries command to list available VPN countries
func NewCmdCountries() *cobra.Command {
	return &cobra.Command{
		Use:   "countries",
		Short: "List VPN countries",
		Example: heredoc.Doc(`
			$ vpn-tool countries
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireLogin(); err != nil {
				return err
			}

			resp, err := apiRequest("GET", "/api/vpn/countries")
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			return handleAPIResponse(resp)
		},
	}
}

// NewCmdVPNCreate creates the "create" VPN command
func NewCmdVPNCreate() *cobra.Command {
	var country string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create VPN",
		Example: heredoc.Doc(`
			$ vpn-tool create --country US
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireLogin(); err != nil {
				return err
			}

			endpoint := fmt.Sprintf("/api/vpn/create?countryCode=%s", url.QueryEscape(country))
			resp, err := apiRequest("POST", endpoint)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			return handleAPIResponse(resp)
		},
	}

	cmd.Flags().StringVarP(&country, "country", "c", "", "Country code (required)")
	_ = cmd.MarkFlagRequired("country")

	return cmd
}

// NewCmdVPNConnect creates the "connect" VPN command
func NewCmdVPNConnect() *cobra.Command {
	var country string

	cmd := &cobra.Command{
		Use:   "connect",
		Short: "Connect VPN",
		Example: heredoc.Doc(`
			$ vpn-tool connect --country US
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireLogin(); err != nil {
				return err
			}

			endpoint := fmt.Sprintf("/api/vpn/connect?countryCode=%s", url.QueryEscape(country))
			resp, err := apiRequest("POST", endpoint)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			return handleAPIResponse(resp)
		},
	}

	cmd.Flags().StringVarP(&country, "country", "c", "", "Country code (required)")
	_ = cmd.MarkFlagRequired("country")

	return cmd
}

// NewCmdVPNDisconnect creates the "disconnect" VPN command
func NewCmdVPNDisconnect() *cobra.Command {
	var country string

	cmd := &cobra.Command{
		Use:   "disconnect",
		Short: "Disconnect VPN",
		Example: heredoc.Doc(`
			$ vpn-tool disconnect --country US
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireLogin(); err != nil {
				return err
			}

			endpoint := fmt.Sprintf("/api/vpn/disconnect?countryCode=%s", url.QueryEscape(country))
			resp, err := apiRequest("POST", endpoint)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			return handleAPIResponse(resp)
		},
	}

	cmd.Flags().StringVarP(&country, "country", "c", "", "Country code (required)")
	_ = cmd.MarkFlagRequired("country")

	return cmd
}

// NewCmdStatus creates the VPN status command
func NewCmdStatus() *cobra.Command {
	var country string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Get VPN status",
		Example: heredoc.Doc(`
			$ vpn-tool status --country US
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireLogin(); err != nil {
				return err
			}

			endpoint := fmt.Sprintf("/api/vpn/status?countryCode=%s", url.QueryEscape(country))
			resp, err := apiRequest("GET", endpoint)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			return handleAPIResponse(resp)
		},
	}

	cmd.Flags().StringVarP(&country, "country", "c", "", "Country code (required)")
	_ = cmd.MarkFlagRequired("country")

	return cmd
}

// NewCmdTest creates the test command to check API delay
func NewCmdTest() *cobra.Command {
	var ms string

	cmd := &cobra.Command{
		Use:   "test",
		Short: "Test API delay",
		Example: heredoc.Doc(`
			$ vpn-tool test --ms 200
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireLogin(); err != nil {
				return err
			}

			endpoint := fmt.Sprintf("/api/vpn/test/sleep?ms=%s", url.QueryEscape(ms))
			resp, err := apiRequest("GET", endpoint)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			return handleAPIResponse(resp)
		},
	}

	cmd.Flags().StringVar(&ms, "ms", "100", "Milliseconds")

	return cmd
}

// NewCmdIP creates the ip command to fetch VPN IP addresses
func NewCmdIP() *cobra.Command {
	var country string
	var count string

	cmd := &cobra.Command{
		Use:   "ip",
		Short: "Get VPN IP addresses",
		Example: heredoc.Doc(`
			$ vpn-tool ip --country US --count 10
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireLogin(); err != nil {
				return err
			}

			endpoint := fmt.Sprintf(
				"/api/vpn/ip?countryCode=%s&count=%s",
				url.QueryEscape(country),
				url.QueryEscape(count),
			)
			resp, err := apiRequest("POST", endpoint)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			return handleAPIResponse(resp)
		},
	}

	cmd.Flags().StringVarP(&country, "country", "c", "", "Country code (required)")
	cmd.Flags().StringVar(&count, "count", "10", "Number of IPs")
	_ = cmd.MarkFlagRequired("country")

	return cmd
}

// --- API Helper Functions ---

func apiRequest(method, endpoint string) (*http.Response, error) {
	fullURL := API_BASE + endpoint
	req, err := http.NewRequest(method, fullURL, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	return client.Do(req)
}

func handleAPIResponse(resp *http.Response) error {
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API request failed: status code %d", resp.StatusCode)
	}

	var result interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	prettyJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}

	fmt.Println(string(prettyJSON))
	return nil
}
