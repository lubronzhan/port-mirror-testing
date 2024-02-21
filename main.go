package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/lubronzhan/port-mirror-testing/pkg/mirror"
)

func main() {
	mirrorCmd := flag.NewFlagSet("mirror", flag.ExitOnError)
	fromNICName := mirrorCmd.String("from", "", "interface-name")
	toNICName := mirrorCmd.String("to", "", "interface-name")

	cleanupCmd := flag.NewFlagSet("cleanup", flag.ExitOnError)
	cleanupNICName := cleanupCmd.String("from", "", "interface-name")

	// The subcommand is expected as the first argument
	// to the program.
	if len(os.Args) < 2 {
		fmt.Println("expected 'mirror' or 'cleanup' subcommands")
		os.Exit(1)
	}

	// Check which subcommand is invoked.
	switch os.Args[1] {

	// For every subcommand, we parse its own flags and
	// have access to trailing positional arguments.
	case "mirror":
		mirrorCmd.Parse(os.Args[2:])
		fmt.Println("subcommand 'mirror'")
		fmt.Printf("traffic will be mirrored from interface %s to interface %s\n", *fromNICName, *toNICName)
		if err := mirror.MirrorTrafficFromNIC(*fromNICName, *toNICName); err != nil {
			fmt.Printf("failed to mirror traffic: %v\n", err)
			os.Exit(1)
		}
	case "cleanup":
		cleanupCmd.Parse(os.Args[2:])
		fmt.Println("subcommand 'cleanup'")
		fmt.Printf("clean up qdisc on interface %s\n", *cleanupNICName)
		if err := mirror.CleanupQDSICFromNIC(*cleanupNICName); err != nil {
			fmt.Printf("failed to clean up qdisc on nic: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Println("expected 'mirror' or 'cleanup' subcommands")
		os.Exit(1)
	}
}
