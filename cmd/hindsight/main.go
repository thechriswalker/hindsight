package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/0x6377/hindsight"
	"github.com/mattn/go-isatty"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// filled in via BUILDFLAGS
var VERSION string = "0.0.0"
var COMMIT string = "-"

const timeFormatMs = "2006-01-02T15:04:05.000Z07:00"
const timeFormatLocal = "2006-01-02 15:04:05.000"

// This is the CLI and Main binary for hindsight

func main() {

	var configFile string
	var debug bool
	var pretty bool

	// debug is overriden by flags, but the default is set by env
	if v, isset := os.LookupEnv("HS_DEBUG"); isset {
		debug, _ = strconv.ParseBool(v)
	}
	// pretty is defaulted by the presence of a tty
	pretty = isatty.IsTerminal(os.Stdout.Fd())

	var config *hindsight.Config

	var rootCmd = &cobra.Command{
		Use:   "hindsight",
		Short: "Hindsight Analytics",

		Version: VERSION,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// configure the logger.
			zerolog.TimeFieldFormat = timeFormatMs
			if pretty {
				log.Logger = log.Output(zerolog.NewConsoleWriter(func(cw *zerolog.ConsoleWriter) {
					cw.TimeFormat = timeFormatLocal
				}))
			}
			if debug {
				zerolog.SetGlobalLevel(zerolog.DebugLevel)
			} else {
				zerolog.SetGlobalLevel(zerolog.InfoLevel)
			}
			// load configuration.
			var err error
			config, err = hindsight.LoadConfig(configFile)
			if err != nil {
				log.Err(err).Str("file", configFile).Msg("error loading config file")
				os.Exit(1)
			}
			log.Debug().Str("file", configFile).Msg("config file loaded successfully")
		},
	}
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "config.toml", "the configuration file to use")
	rootCmd.PersistentFlags().BoolVar(&debug, "verbose", debug, "enable more detailed logging (default set with ENV var HS_DEBUG)")
	rootCmd.PersistentFlags().BoolVar(&pretty, "pretty", pretty, "enable human readable logging (default set by TTY")

	var ingest = &cobra.Command{
		Use:   "ingest",
		Short: "ingest log files",
		Run: func(cmd *cobra.Command, args []string) {
			// nope
			log.Fatal().Msg("not implemented")
		},
	}

	var run = &cobra.Command{
		Use:   "run",
		Short: "run the daemon",
		Run: func(cmd *cobra.Command, args []string) {
			err := run(config)
			if err != nil {
				log.Fatal().Err(err).Msg("Error running daemon")
			}
		},
	}

	rootCmd.AddCommand(run, ingest)
	rootCmd.SetVersionTemplate(fmt.Sprintf("{{.Name}} v{{.Version}} (%s)\n", COMMIT))

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
