package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/LyWilson/paritok-cli/internal/config"
)

var rootCmd = &cobra.Command{
	Use:   "paritok-cli",
	Short: "Paritok CLI – interactive coding agent",
	Long: `A cost-optimized interactive coding agent that connects to the
Paritok proxy endpoint for AI-powered code assistance.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var authCmd = &cobra.Command{
	Use:   "auth <api_key>",
	Short: "Save your Paritok API key",
	Long:  "Store the API key in $HOME/.paritok.json for future sessions.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		if key == "" {
			return fmt.Errorf("api key cannot be empty")
		}
		if err := config.Save(key); err != nil {
			return fmt.Errorf("save api key: %w", err)
		}
		fmt.Fprintln(os.Stderr, "✓ API key saved to "+config.ConfigFileName)
		return nil
	},
}

var authNvidiaCmd = &cobra.Command{
	Use:   "auth-nvidia <api_key>",
	Short: "Save your NVIDIA API key",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		if key == "" {
			return fmt.Errorf("api key cannot be empty")
		}
		if err := config.SaveNvidia(key); err != nil {
			return fmt.Errorf("save nvidia api key: %w", err)
		}
		fmt.Fprintln(os.Stderr, "NVIDIA API key saved")
		return nil
	},
}

func Execute() error {
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(authNvidiaCmd)
	return rootCmd.Execute()
}
