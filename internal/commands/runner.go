// SPDX-FileCopyrightText: 2026 Josef Gallmetzer
//
// SPDX-License-Identifier: MPL-2.0

package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/galjos/odh-cli/internal/apis"
	"github.com/galjos/odh-cli/internal/client"
	"github.com/galjos/odh-cli/internal/openapi"
	"github.com/galjos/odh-cli/internal/output"
	"github.com/spf13/cobra"
)

type usageError struct {
	err error
}

func (e usageError) Error() string {
	return e.err.Error()
}

func usageErrorf(format string, args ...any) error {
	return usageError{err: fmt.Errorf(format, args...)}
}

func requireSubcommand(cmd *cobra.Command, _ []string) error {
	return usageErrorf("usage: %s <subcommand>", cmd.CommandPath())
}

// Runner owns command execution dependencies.
type Runner struct {
	Registry *apis.Registry
	Client   *client.Client
}

// NewDefaultRunner creates a runner for production use.
func NewDefaultRunner() *Runner {
	return &Runner{
		Registry: apis.DefaultRegistry(),
		Client:   client.New(30 * time.Second),
	}
}

// NewRootCmd builds the Cobra command hierarchy for the CLI.
func (r *Runner) NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "odh",
		Short:         "odh is a JSON-first CLI for Open Data Hub APIs",
		Long:          `odh is an unofficial JSON-first command-line interface for public Open Data Hub APIs.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	rootCmd.AddCommand(r.newVersionCmd())
	rootCmd.AddCommand(r.newDoctorCmd())
	rootCmd.AddCommand(r.newAPIsCmd())
	rootCmd.AddCommand(r.newDatasetsCmd())
	rootCmd.AddCommand(r.newOpenAPICmd())
	rootCmd.AddCommand(r.newCallCmd())
	rootCmd.AddCommand(r.newTourismCmd())
	rootCmd.AddCommand(r.newGTFSCmd())
	rootCmd.AddCommand(r.newTransitCmd())
	rootCmd.AddCommand(r.newDiagnosticsCmd())
	rootCmd.AddCommand(r.newTrafficCmd())
	rootCmd.AddCommand(r.newMobilityCmd())
	rootCmd.AddCommand(r.newA22Cmd())

	return rootCmd
}

// Run executes the CLI and returns a process exit code.
func (r *Runner) Run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}
	if r.Registry == nil {
		r.Registry = apis.DefaultRegistry()
	}
	if r.Client == nil {
		r.Client = client.New(30 * time.Second)
	}

	rootCmd := r.NewRootCmd()
	rootCmd.SetArgs(args)
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)
	if len(args) == 0 {
		rootCmd.SetOut(stderr)
		_ = rootCmd.Help()
		return 2
	}

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(stderr, err)
		if isUsageError(err) {
			return 2
		}
		return 1
	}
	return 0
}

func isUsageError(err error) bool {
	var typed usageError
	if errors.As(err, &typed) {
		return true
	}
	message := err.Error()
	usageFragments := []string{
		"unknown command ",
		"unknown flag: ",
		"accepts ",
		"requires at least ",
		"requires exactly ",
		"requires key=value",
		"must use key=value",
		" is required",
		" must be greater than zero",
		" must not be negative",
		"unsupported ",
		"unsupported format ",
		"unsupported traffic source ",
		"unsupported traffic type ",
		"unsupported mobility type kind ",
		"unsupported tourism types dataset ",
		"unsupported GTFS feed ",
		"unsupported transit mode ",
		"unknown api ",
		"unknown traffic zone-id ",
		"invalid --fresh-within ",
		"invalid --window ",
		"--date must use YYYY-MM-DD",
		"--near must use lat,lon",
		"use either --",
	}
	for _, fragment := range usageFragments {
		if strings.Contains(message, fragment) {
			return true
		}
	}
	return false
}

func (r *Runner) newAPIsCmd() *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:   "apis",
		Short: "List known Open Data Hub APIs",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			switch format {
			case "json":
				return output.WriteJSON(cmd.OutOrStdout(), map[string]any{"apis": r.Registry.List()})
			case "table":
				fmt.Fprintln(cmd.OutOrStdout(), "NAME\tBASE URL\tOPENAPI")
				for _, api := range r.Registry.List() {
					fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n", api.Name, api.BaseURL, api.OpenAPIURL)
				}
				return nil
			default:
				return fmt.Errorf("unsupported format %q", format)
			}
		},
	}
	cmd.Flags().StringVar(&format, "format", "json", "output format: json or table")
	return cmd
}

func (r *Runner) newOpenAPICmd() *cobra.Command {
	return &cobra.Command{
		Use:   "openapi <api>",
		Short: "Fetch OpenAPI specs as JSON",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, ok := r.Registry.Find(args[0])
			if !ok {
				return usageErrorf("unknown api %q; known APIs: %s", args[0], strings.Join(r.Registry.Names(), ", "))
			}
			if api.OpenAPIURL == "" {
				return fmt.Errorf("api %q does not have a known OpenAPI URL", api.Name)
			}
			resp, err := r.Client.GetCached(cmd.Context(), api.OpenAPIURL, 24*time.Hour)
			if err != nil {
				return err
			}
			encoded, _, err := openapi.ToJSON(resp.Body)
			if err != nil {
				return err
			}
			_, err = cmd.OutOrStdout().Write(encoded)
			return err
		},
	}
}

func (r *Runner) newCallCmd() *cobra.Command {
	var params []string
	cmd := &cobra.Command{
		Use:   "call <api> <path>",
		Short: "Call any registered API path with query parameters",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			apiName := args[0]
			path := args[1]
			api, ok := r.Registry.Find(apiName)
			if !ok {
				return usageErrorf("unknown api %q; known APIs: %s", apiName, strings.Join(r.Registry.Names(), ", "))
			}

			values := url.Values{}
			for _, p := range params {
				key, value, ok := strings.Cut(p, "=")
				if !ok || strings.TrimSpace(key) == "" {
					return usageErrorf("parameter %q must use key=value", p)
				}
				values.Add(key, value)
			}

			requestURL, err := BuildURL(api.BaseURL, path, values)
			if err != nil {
				return err
			}
			return r.fetchJSONCobra(cmd.Context(), requestURL, cmd.OutOrStdout())
		},
	}
	cmd.Flags().StringSliceVar(&params, "param", nil, "additional query parameter as key=value; repeatable")
	return cmd
}

func (r *Runner) fetchJSONCobra(ctx context.Context, requestURL string, stdout io.Writer) error {
	resp, err := r.Client.Get(ctx, requestURL)
	if err != nil {
		return err
	}
	return output.WriteRawJSON(stdout, resp.Body)
}

func (r *Runner) fetchJSONValueCached(ctx context.Context, requestURL string, ttl time.Duration) (any, error) {
	resp, err := r.Client.GetCached(ctx, requestURL, ttl)
	if err != nil {
		return nil, err
	}
	var value any
	if err := json.Unmarshal(resp.Body, &value); err != nil {
		return nil, fmt.Errorf("response is not valid JSON: %w", err)
	}
	return value, nil
}

func (r *Runner) fetchJSONValue(ctx context.Context, requestURL string) (any, error) {
	return r.fetchJSONValueCached(ctx, requestURL, 0)
}

func (r *Runner) fetchListFromMobility(ctx context.Context, baseURL, path string, limit int) ([]map[string]any, string, error) {
	values := url.Values{}
	values.Set("limit", strconv.Itoa(limit))
	requestURL, err := BuildURL(baseURL, path, values)
	if err != nil {
		return nil, "", err
	}
	value, err := r.fetchJSONValue(ctx, requestURL)
	if err != nil {
		return nil, requestURL, err
	}
	return extractDataList(value), requestURL, nil
}

// BuildURL joins a base URL, path, and query parameters.
func BuildURL(baseURL, path string, params url.Values) (string, error) {
	if strings.TrimSpace(baseURL) == "" {
		return "", errors.New("base URL is required")
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid base URL %q", baseURL)
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		absolute, err := url.Parse(path)
		if err != nil {
			return "", err
		}
		parsed = absolute
	} else {
		joined, err := url.JoinPath(parsed.String(), path)
		if err != nil {
			return "", err
		}
		parsed, err = url.Parse(joined)
		if err != nil {
			return "", err
		}
	}
	query := parsed.Query()
	for key, values := range params {
		for _, value := range values {
			query.Add(key, value)
		}
	}
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}
