package main

import (
	"flag"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/UnitVectorY-Labs/datacur8/internal/cli"
)

var Version = "dev" // This will be set by the build systems to the release version

func usage() {
	fmt.Fprintln(os.Stderr, `Usage: datacur8 <command> [flags]

Commands:
  validate    Validate configuration and data files
  export      Export validated data to configured outputs
  tidy        Normalize file formatting for stable diffs
  version     Print the version

Run 'datacur8 <command> --help' for more information on a command.`)
}

// main is the entry point for the datacur8 command-line tool.
func main() {
	// Set the build version from the build info if not set by the build system
	if Version == "dev" || Version == "" {
		if bi, ok := debug.ReadBuildInfo(); ok {
			if bi.Main.Version != "" && bi.Main.Version != "(devel)" {
				Version = bi.Main.Version
			}
		}
	}

	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "validate":
		validateFlags := flag.NewFlagSet("validate", flag.ExitOnError)
		configOnly := validateFlags.Bool("config-only", false, "Only validate configuration, not data files")
		format := validateFlags.String("format", "", "Output format (text, json, yaml)")
		validateFlags.Parse(os.Args[2:])
		os.Exit(cli.RunValidate(*configOnly, *format, Version))

	case "export":
		exportFlags := flag.NewFlagSet("export", flag.ExitOnError)
		exportFlags.Parse(os.Args[2:])
		os.Exit(cli.RunExport(Version))

	case "tidy":
		tidyFlags := flag.NewFlagSet("tidy", flag.ExitOnError)
		write := tidyFlags.Bool("write", false, "Rewrite files in place (default is check-only diff mode)")
		tidyFlags.Parse(os.Args[2:])
		os.Exit(cli.RunTidy(*write, Version))

	case "version":
		fmt.Println(Version)
		os.Exit(0)

	default:
		usage()
		os.Exit(1)
	}
}
