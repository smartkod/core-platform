// Command corectl scaffolds and maintains applications built on the platform.
//
// It exists so starting a new application is one command instead of a manual
// copy of auth, config, migrations and Docker files. See PRD.md §6
package main

import (
	"fmt"
	"os"

	"core-platform/cmd/corectl/commands"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	var err error
	switch os.Args[1] {
	case "new":
		err = commands.New(os.Args[2:])
	case "add":
		err = commands.Add(os.Args[2:])
	case "doctor":
		err = commands.Doctor(os.Args[2:])
	case "upgrade":
		err = commands.Upgrade(os.Args[2:])
	case "version", "-v", "--version":
		fmt.Println("corectl 0.1.0")
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "corectl: unknown command %q\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "corectl: %v\n", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Print(`corectl — scaffold applications on the core platform

Usage:
  corectl new <app> --level=L0|L1|L2 [--only=mod,mod] [--dir=path]
  corectl add module <name>
  corectl add level <LN>
  corectl doctor
  corectl upgrade

Levels:
  L0  kernel only (config, db, migrations, modules, http, health)
  L1  L0 + identity, authz, files, notifications
  L2  L1 + organization, catalog (the service layer)

Examples:
  corectl new fitscope --level=L1 --only=identity,files
  corectl new eduscope --level=L2
`)
}
