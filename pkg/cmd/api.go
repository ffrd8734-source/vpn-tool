package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
)

const API_BASE = "https://vpn-gateway-manager--jjajajjajahsh.replit.app"

// RequestOptions holds parsed flags for apiRequest
type RequestOptions struct {
	Method    string
	Hostname  string
	URLFlag   string
	Headers   []string
	Fields    []string
	RawFields []string
	Input     string
	Cache     string
	Include   bool
	Silent    bool
	Verbose   bool
	Jq        string
	Template  string
	Paginate  bool
	Slurp     bool
}

// NewCmdAPI creates the api command for making raw HTTP requests to the VPN Gateway API
func NewCmdAPI() *cobra.Command {
	var opts RequestOptions

	cmd := &cobra.Command{
		Use:   "api [<endpoint>|<full-url>]",
		Short: "Make an authenticated HTTP request to the VPN API",
		Long: heredoc.Doc(`
			Makes an authenticated HTTP request to the VPN Gateway Manager API and prints the response.
			The endpoint argument should be a path of an API endpoint (e.g. "/api/vpn/countries") or a full URL via --url.
		`),
		Example: heredoc.Doc(`
			$ vpn-tool api /api/vpn/countries
			$ vpn-tool api /api/vpn/create --method POST -f countryCode=US
			$ vpn-tool api --url https://custom-endpoint.com/api/test
			$ vpn-tool api /api/healthz --include
		`),
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireLogin(); err != nil {
				return err
			}

			var targetURL string
			if opts.URLFlag != "" {
				targetURL = opts.URLFlag
			} else if len(args) > 0 {
				endpoint := args[0]
				baseURL := API_BASE
				if opts.Hostname != "" {
					baseURL = opts.Hostname
				}
				if !strings.HasPrefix(endpoint, "/") && !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
					endpoint = "/" + endpoint
				}
				if strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
					targetURL = endpoint
				} else {
					targetURL = baseURL + endpoint
				}
			} else {
				return fmt.Errorf("accepts 1 arg(s), received 0; specify an endpoint or use --url")
			}

			// Default method logic: GET, or POST if fields are present
			method := strings.ToUpper(opts.Method)
			if cmd.Flags().Changed("method") == false && (len(opts.Fields) > 0 || len(opts.RawFields) > 0) {
				method = "POST"
			}
			if method == "" {
				method = "GET"
			}

			// Build request body or query parameters
			var bodyReader io.Reader
			queryParams := make(url.Values)

			// Parse --raw-field (-f) and --field (-F)
			payloadMap := make(map[string]interface{})
			for _, f := range opts.RawFields {
				parts := strings.SplitN(f, "=", 2)
				if len(parts) == 2 {
					payloadMap[parts[0]] = parts[1]
				}
			}
			for _, f := range opts.Fields {
				parts := strings.SplitN(f, "=", 2)
				if len(parts) == 2 {
					val := parts[1]
					if strings.HasPrefix(val, "@") {
						filename := val[1:]
						var content []byte
						var err error
						if filename == "-" {
							content, err = io.ReadAll(os.Stdin)
						} else {
							content, err = os.ReadFile(filename)
						}
						if err != nil {
							return fmt.Errorf("failed to read field file %s: %w", filename, err)
						}
						payloadMap[parts[0]] = string(content)
					} else {
						payloadMap[parts[0]] = val
					}
				}
			}

			// Handle --input body file or stdin
			if opts.Input != "" {
				if opts.Input == "-" {
					bodyReader = os.Stdin
				} else {
					f, err := os.Open(opts.Input)
					if err != nil {
						return fmt.Errorf("failed to open input file: %w", err)
					}
					defer f.Close()
					bodyReader = f
				}
			} else if len(payloadMap) > 0 {
				jsonBytes, err := json.Marshal(payloadMap)
				if err != nil {
					return fmt.Errorf("failed to encode request payload: %w", err)
				}
				bodyReader = bytes.NewReader(jsonBytes)
			}

			// If GET method with payload map, append to query string instead
			if method == "GET" && len(payloadMap) > 0 {
				for k, v := range payloadMap {
					if strVal, ok := v.(string); ok {
						queryParams.Add(k, strVal)
					}
				}
			}

			if len(queryParams) > 0 {
				if strings.Contains(targetURL, "?") {
					targetURL += "&" + queryParams.Encode()
				} else {
					targetURL += "?" + queryParams.Encode()
				}
			}

			req, err := http.NewRequest(method, targetURL, bodyReader)
			if err != nil {
				return err
			}

			// Apply headers
			for _, h := range opts.Headers {
				parts := strings.SplitN(h, ":", 2)
				if len(parts) == 2 {
					req.Header.Set(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
				}
			}
			if bodyReader != nil && req.Header.Get("Content-Type") == "" {
				req.Header.Set("Content-Type", "application/json")
			}

			client := &http.Client{Timeout: 30 * time.Second}

			// Verbose request dumping
			if opts.Verbose {
				dump, _ := httputil.DumpRequestOut(req, true)
				fmt.Printf("%s\n\n", string(dump))
			}

			resp, err := client.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if opts.Verbose {
				dump, _ := httputil.DumpResponse(resp, true)
				fmt.Printf("%s\n", string(dump))
				return nil
			}

			if opts.Include {
				fmt.Printf("%s %s\n", resp.Proto, resp.Status)
				for k, vals := range resp.Header {
					for _, v := range vals {
						fmt.Printf("%s: %s\n", k, v)
					}
				}
				fmt.Println()
			}

			if opts.Silent {
				return nil
			}

			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}

			var rawJSON interface{}
			if err := json.Unmarshal(respBody, &rawJSON); err == nil {
				formatted, _ := json.MarshalIndent(rawJSON, "", "  ")
				fmt.Println(string(formatted))
			} else {
				fmt.Println(string(respBody))
			}

			return nil
		},
	}

	// Bind flags
	cmd.Flags().StringVar(&opts.Cache, "cache", "", `Cache the response, e.g. "3600s", "60m", "1h"`)
	cmd.Flags().StringSliceVarP(&opts.Fields, "field", "F", nil, `Add a typed parameter in key=value format (use "@<path>" or "@-" to read value from file or stdin)`)
	cmd.Flags().StringSliceVarP(&opts.Headers, "header", "H", nil, `Add a HTTP request header in key=value format`)
	cmd.Flags().StringVar(&opts.Hostname, "hostname", "", `The vpn gateway manager api hostname for the request`)
	cmd.Flags().StringVar(&opts.URLFlag, "url", "", `The full URL to make the HTTP request to`)
	cmd.Flags().BoolVarP(&opts.Include, "include", "i", false, `Include HTTP response status line and headers in the output`)
	cmd.Flags().StringVar(&opts.Input, "input", "", `The file to use as body for the HTTP request (use "-" to read from standard input)`)
	cmd.Flags().StringVarP(&opts.Jq, "jq", "q", "", `Query to select values from the response using jq syntax`)
	cmd.Flags().StringVarP(&opts.Method, "method", "X", "GET", `The HTTP method for the request`)
	cmd.Flags().BoolVar(&opts.Paginate, "paginate", false, `Make additional HTTP requests to fetch all pages of results`)
	cmd.Flags().StringSliceVarP(&opts.Preview, "preview", "p", nil, `Opt into GitHub API previews (names should omit '-preview')`)
	cmd.Flags().StringSliceVarP(&opts.RawFields, "raw-field", "f", nil, `Add a string parameter in key=value format`)
	cmd.Flags().BoolVar(&opts.Silent, "silent", false, `Do not print the response body`)
	cmd.Flags().BoolVar(&opts.Slurp, "slurp", false, `Use with "--paginate" to return an array of all pages`)
	cmd.Flags().StringVarP(&opts.Template, "template", "t", "", `Format JSON output using a Go template`)
	cmd.Flags().BoolVar(&opts.Verbose, "verbose", false, `Include full HTTP request and response in the output`)

	return cmd
}

// Backward-compatibility alias helper for other commands inside the package
func apiRequest(method, endpoint string) (*http.Response, error) {
	fullURL := API_BASE + endpoint
	req, err := http.NewRequest(method, fullURL, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{}
	return client.Do(req)
}
