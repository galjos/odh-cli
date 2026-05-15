// SPDX-FileCopyrightText: 2026 Josef Gallmetzer
//
// SPDX-License-Identifier: MPL-2.0

package commands

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/galjos/odh-cli/internal/output"
	"github.com/galjos/odh-cli/internal/version"
)

type doctorCheck struct {
	Name       string `json:"name"`
	OK         bool   `json:"ok"`
	StatusCode int    `json:"status_code,omitempty"`
	URL        string `json:"url,omitempty"`
	Message    string `json:"message,omitempty"`
}

func (r *Runner) runDoctor(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("doctor", stderr)
	network := fs.Bool("network", true, "run network reachability checks")
	timeout := fs.Duration("timeout", 10*time.Second, "overall timeout for doctor checks")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "doctor does not accept positional arguments")
		return 2
	}
	if *timeout <= 0 {
		fmt.Fprintln(stderr, "--timeout must be greater than zero")
		return 2
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

	if *network {
		doctorCtx, cancel := context.WithTimeout(ctx, *timeout)
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

	if err := output.WriteJSON(stdout, map[string]any{
		"ok":      ok,
		"version": version.Current(),
		"checks":  checks,
	}); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if !ok {
		return 1
	}
	return 0
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
		} else {
			check.OK = true
			check.Message = "HTTP " + strconv.Itoa(resp.StatusCode)
		}
		checks = append(checks, check)
	}
	return checks
}

func (r *Runner) runVersion(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("version", stderr)
	format := fs.String("format", "json", "output format: json or text")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "version does not accept positional arguments")
		return 2
	}

	info := version.Current()
	switch *format {
	case "json":
		if err := output.WriteJSON(stdout, info); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	case "text":
		fmt.Fprintf(stdout, "odh %s (%s, %s, %s/%s)\n", info.Version, info.Commit, info.Date, info.GoOS, info.GoArch)
	default:
		fmt.Fprintf(stderr, "unsupported format %q\n", *format)
		return 2
	}
	return 0
}
