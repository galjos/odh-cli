// SPDX-FileCopyrightText: 2026 Josef Gallmetzer
//
// SPDX-License-Identifier: MPL-2.0

package version

import (
	"runtime"
	"runtime/debug"
	"strings"
)

// These variables can be overridden at build time with -ldflags.
var (
	Version = "0.1.8-dev"
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
	resolvedVersion, resolvedCommit, resolvedDate := Version, Commit, Date
	if buildInfo, ok := debug.ReadBuildInfo(); ok {
		hasVCSRevision := false
		for _, setting := range buildInfo.Settings {
			switch setting.Key {
			case "vcs.revision":
				hasVCSRevision = setting.Value != ""
				if resolvedCommit == "unknown" && setting.Value != "" {
					resolvedCommit = shortenCommit(setting.Value)
				}
			case "vcs.time":
				if resolvedDate == "unknown" && setting.Value != "" {
					resolvedDate = setting.Value
				}
			}
		}
		if strings.HasSuffix(resolvedVersion, "-dev") && !hasVCSRevision && buildInfo.Main.Version != "" && buildInfo.Main.Version != "(devel)" {
			resolvedVersion = buildInfo.Main.Version
		}
	}

	return Info{
		Version: resolvedVersion,
		Commit:  resolvedCommit,
		Date:    resolvedDate,
		GoOS:    runtime.GOOS,
		GoArch:  runtime.GOARCH,
	}
}

func shortenCommit(commit string) string {
	if len(commit) <= 12 {
		return commit
	}
	return commit[:12]
}
