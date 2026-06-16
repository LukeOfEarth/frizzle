package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/LukeOfEarth/frizzle/internal/config"
	"github.com/LukeOfEarth/frizzle/internal/server"
	"github.com/spf13/cobra"
)

var (
	startPort    int
	startForward string
	startConfig  string
)

func init() {
	startCmd.Flags().IntVarP(&startPort, "port", "p", 0, "Port to listen on (overrides config)")
	startCmd.Flags().StringVarP(&startForward, "forward", "f", "", "Forward all events to a downstream HTTP endpoint (simple mode)")
	startCmd.Flags().StringVarP(&startConfig, "config", "c", ".frizzle/frizzle.json", "Path to config file")
	rootCmd.AddCommand(startCmd)
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the local event bus",
	Long: `Start a local event bus that listens for events, matches them against
configured rules, logs them to the console, and forwards them to targets.

With no arguments, it reads .frizzle/frizzle.json from the current directory.

For quick setups without a config file, use --forward to create a single bus
that logs and forwards every event to a downstream endpoint:

  frizzle start --port 4000 --forward http://localhost:8080/webhook`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		var cfg *config.Config
		var err error

		if startForward != "" || startPort != 0 {
			port := startPort
			if port == 0 {
				port = 4000
			}
			cfg = config.SimpleConfig(port, startForward)
			fmt.Fprintf(os.Stderr, "Running in simple mode (no config file)\n")
		} else {
			cfg, err = config.Load(startConfig)
			if err != nil {
				if os.IsNotExist(err) {
					if startConfig == ".frizzle/frizzle.json" && startForward == "" && startPort == 0 {
						return fmt.Errorf("no .frizzle/frizzle.json found — run 'frizzle init' to scaffold one, or use --forward for simple mode")
					}
				}
				return fmt.Errorf("failed to load config: %w", err)
			}
		}

		if startPort != 0 {
			cfg.Port = startPort
		}

		logger := log.New(os.Stdout, "", log.LstdFlags|log.Lmsgprefix)
		srv := server.New(cfg, logger)
		return srv.ListenAndServe()
	},
}
