// SPDX-FileCopyrightText: 2026 Josef Gallmetzer
//
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"os"

	"github.com/galjos/odh-cli/internal/commands"
)

func main() {
	runner := commands.NewDefaultRunner()
	os.Exit(runner.Run(context.Background(), os.Args[1:], os.Stdout, os.Stderr))
}
