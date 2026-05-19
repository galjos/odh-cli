// SPDX-FileCopyrightText: 2026 Josef Gallmetzer
//
// SPDX-License-Identifier: MPL-2.0

package commands

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/galjos/odh-cli/internal/client"
	"github.com/galjos/odh-cli/internal/output"
	"github.com/galjos/odh-cli/internal/version"
	"github.com/spf13/cobra"
)

type doctorCheck struct {
	Name       string `json:"name"`
	OK         bool   `json:"ok"`
	StatusCode int    `json:"status_code,omitempty"`
	URL        string `json:"url,omitempty"`
	Message    string `json:"message,omitempty"`
}

func (r *Runner) newVersionCmd() *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version metadata",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			info := version.Current()
			switch format {
			case "json":
				return output.WriteJSON(cmd.OutOrStdout(), info)
			case "text":
				fmt.Fprintf(cmd.OutOrStdout(), "odh %s (%s, %s, %s/%s)\n", info.Version, info.Commit, info.Date, info.GoOS, info.GoArch)
				return nil
			default:
				return fmt.Errorf("unsupported format %q", format)
			}
		},
	}
	cmd.Flags().StringVar(&format, "format", "json", "output format: json or text")
	return cmd
}

func (r *Runner) newDoctorCmd() *cobra.Command {
	var network bool
	var timeout time.Duration
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check the local CLI and upstream API reachability",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if timeout <= 0 {
				return fmt.Errorf("--timeout must be greater than zero")
			}

			checks := []doctorCheck{
				{
					Name:    "version",
					OK:      true,
					Message: version.Current().Version,
				},
				{
					Name:    "api_registry",
					OK:      len(r.Registry.List()) > 0,
					Message: fmt.Sprintf("%d APIs configured", len(r.Registry.List())),
				},
			}

			if network {
				doctorCtx, cancel := context.WithTimeout(cmd.Context(), timeout)
				defer cancel()
				checks = append(checks, r.runNetworkDoctorChecks(doctorCtx)...)
			}

			ok := true
			for _, check := range checks {
				if !check.OK {
					ok = false
					break
				}
			}

			if err := output.WriteJSON(cmd.OutOrStdout(), map[string]any{
				"ok":      ok,
				"version": version.Current(),
				"checks":  checks,
			}); err != nil {
				return err
			}
			if !ok {
				return fmt.Errorf("doctor checks failed")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&network, "network", true, "run network reachability checks")
	cmd.Flags().DurationVar(&timeout, "timeout", 10*time.Second, "overall timeout for doctor checks")
	return cmd
}

func (r *Runner) runNetworkDoctorChecks(ctx context.Context) []doctorCheck {
	type target struct {
		name string
		api  string
		path string
	}
	targets := []target{
		{name: "tourism_openapi", api: "tourism", path: "/swagger/v1/swagger.json"},
		{name: "mobility_openapi", api: "mobility", path: "/v2/apispec"},
		{name: "mobility_station_types", api: "mobility", path: "/v2/flat"},
		{name: "a22_events", api: "mobility", path: "/v2/flat,event/A22/latest"},
	}

	checks := make([]doctorCheck, 0, len(targets))
	for _, target := range targets {
		api, ok := r.Registry.Find(target.api)
		if !ok {
			checks = append(checks, doctorCheck{
				Name:    target.name,
				OK:      false,
				Message: fmt.Sprintf("api %q is not configured", target.api),
			})
			continue
		}
		values := url.Values{}
		values.Set("limit", "1")
		if strings.Contains(target.name, "openapi") {
			values = url.Values{}
		}
		requestURL, err := BuildURL(api.BaseURL, target.path, values)
		if err != nil {
			checks = append(checks, doctorCheck{
				Name:    target.name,
				OK:      false,
				Message: err.Error(),
			})
			continue
		}
		resp, err := r.Client.Get(ctx, requestURL)
		check := doctorCheck{
			Name:       target.name,
			URL:        requestURL,
			StatusCode: resp.StatusCode,
		}
		if err != nil {
			check.OK = false
			check.Message = err.Error()
			var httpErr *client.HTTPError
			if errors.As(err, &httpErr) {
				check.StatusCode = httpErr.StatusCode
			}
		} else {
			check.OK = true
			check.Message = "HTTP " + strconv.Itoa(resp.StatusCode)
		}
		checks = append(checks, check)
	}
	return checks
}
