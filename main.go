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
	case "help", "--help", "-help", "-h":
		usage()
		os.Exit(0)

	case "validate":
		validateFlags := flag.NewFlagSet("validate", flag.ExitOnError)
		validateFlags.Usage = func() {
			fmt.Fprintln(os.Stderr, `Usage: datacur8 validate [flags]

Validate the .datacur8 configuration and all matching data files.

Flags:`)
			validateFlags.PrintDefaults()
		}
		configOnly := validateFlags.Bool("config-only", false, "Only validate configuration, not data files")
		format := validateFlags.String("format", "", "Output format: text, json, or yaml (default: text)")
		validateFlags.Parse(os.Args[2:])
		if validateFlags.NArg() > 0 {
			fmt.Fprintf(os.Stderr, "unexpected argument: %s\n", validateFlags.Arg(0))
			validateFlags.Usage()
			os.Exit(1)
		}
		os.Exit(cli.RunValidate(*configOnly, *format, Version))

	case "export":
		exportFlags := flag.NewFlagSet("export", flag.ExitOnError)
		exportFlags.Usage = func() {
			fmt.Fprintln(os.Stderr, `Usage: datacur8 export [flags]

Export validated data to configured output files. Runs full validation first;
if validation fails, export does not proceed.

Flags:`)
			exportFlags.PrintDefaults()
		}
		format := exportFlags.String("format", "", "Output format: text, json, or yaml (default: text)")
		exportFlags.Parse(os.Args[2:])
		if exportFlags.NArg() > 0 {
			fmt.Fprintf(os.Stderr, "unexpected argument: %s\n", exportFlags.Arg(0))
			exportFlags.Usage()
			os.Exit(1)
		}
		os.Exit(cli.RunExport(*format, Version))

	case "tidy":
		tidyFlags := flag.NewFlagSet("tidy", flag.ExitOnError)
		tidyFlags.Usage = func() {
			fmt.Fprintln(os.Stderr, `Usage: datacur8 tidy [flags]

Normalize file formatting for stable diffs. Default mode is check-only,
which prints a colored diff and exits non-zero if changes are needed.

Flags:`)
			tidyFlags.PrintDefaults()
		}
		write := tidyFlags.Bool("write", false, "Rewrite files in place (default is check-only diff mode)")
		format := tidyFlags.String("format", "", "Output format: text, json, or yaml (default: text)")
		tidyFlags.Parse(os.Args[2:])
		if tidyFlags.NArg() > 0 {
			fmt.Fprintf(os.Stderr, "unexpected argument: %s\n", tidyFlags.Arg(0))
			tidyFlags.Usage()
			os.Exit(1)
		}
		os.Exit(cli.RunTidy(*write, *format, Version))

	case "version":
		fmt.Println(Version)
		os.Exit(0)

	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}
