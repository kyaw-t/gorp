package cmd

import (
	"flag"
	"fmt"
	"gorp/cmd/start"
	"os"
)

func Execute() {
	// subcommands
	startCmd := flag.NewFlagSet("start", flag.ExitOnError)

	// flag pointers for subcommands
	portPtr := startCmd.Int("port", -1, "Port number")
	locationPtr := startCmd.String("location", "", "Metric flag")
	dryRunPtr := startCmd.Bool("dry-run", false, "Dry run flag")
	flag.Parse()

	if len(os.Args) < 2 {
		fmt.Println("subcommand is required")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "start":
		startCmd.Parse(os.Args[2:])

	case "init":
		fmt.Println("init subcommand")
		return

	default:
		flag.PrintDefaults()
		fmt.Println("Invalid subcommand")
		os.Exit(1)
	}

	// Check which subcommand is parsed
	if startCmd.Parsed() {
		start.Start(locationPtr, portPtr, dryRunPtr)
	}

}
