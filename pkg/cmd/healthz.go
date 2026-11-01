package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
)

const API_BASE = "https://vpn-gateway-manager--jjajajjajahsh.replit.app"

// NewCmdHealthz creates the healthz command to check API health status
func NewCmdHealthz() *cobra.Command {
	return &cobra.Command{
		Use:   "healthz",
		Short: "Check API health",
		Example: heredoc.Doc(`
			$ vpn-tool healthz
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Ensure the user is authenticated
			if err := requireLogin(); err != nil {
				return err
			}

			// Call the API endpoint using apiRequest
			resp, err := apiRequest("GET", "/api/healthz")
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("API health check failed: status code %d", resp.StatusCode)
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
		},
	}
}

// apiRequest performs an HTTP request against the VPN Gateway Manager API backend
func apiRequest(method, endpoint string) (*http.Response, error) {
	url := API_BASE + endpoint
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	return client.Do(req)
}
