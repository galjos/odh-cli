// SPDX-FileCopyrightText: 2026 Josef Gallmetzer
//
// SPDX-License-Identifier: MPL-2.0

package version

import "runtime"

// These variables can be overridden at build time with -ldflags.
var (
	Version = "0.1.0-dev"
	Commit  = "unknown"
	Date    = "unknown"
)

// Info is the machine-readable version payload emitted by the CLI.
type Info struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
	GoOS    string `json:"goos"`
	GoArch  string `json:"goarch"`
}

// Current returns the current build metadata.
func Current() Info {
	return Info{
		Version: Version,
		Commit:  Commit,
		Date:    Date,
		GoOS:    runtime.GOOS,
		GoArch:  runtime.GOARCH,
	}
}
