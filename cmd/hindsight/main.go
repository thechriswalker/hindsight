package main

import (
	"log"

	hindsight "github.com/0x6377/hindsight"
	"github.com/spf13/cobra"
)

// This is the CLI and Main binary for hindsight

func main() {
	// use something like cobra to handle the CLI parsing
	// then switch on function
	//
	// run: run the daemon
	// hindsight.Run()
	// ingest: ingest a file
	// hindsight.Ingest()
	_, err := hindsight.LoadConfig("config.toml")
	if err != nil {
		log.Fatalln(err)
	}

	var rootCmd = &cobra.Command{
		Use:     "hindsight",
		Short:   "Hindsight Analytics",
		Version: "1.0.0",
		Run: func(cmd *cobra.Command, args []string) {
			// Do Stuff Here
		},
	}

	var ingest = &cobra.Command{
		Use:   "ingest",
		Short: "Ingest log files",
		Run: func(cmd *cobra.Command, args []string) {
			// nope
		},
	}

	var run = &cobra.Command{
		Use:   "run",
		Short: "run the daemon",
		Run: func(cmd *cobra.Command, args []string) {
			// nope
		},
	}

	rootCmd.AddCommand(run, ingest)

	err = rootCmd.Execute()
	if err != nil {
		log.Fatalln(err)
	}
}
