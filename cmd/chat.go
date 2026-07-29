package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/LyWilson/prompt-diet-coach/internal/client"
	"github.com/LyWilson/prompt-diet-coach/internal/config"
	"github.com/LyWilson/prompt-diet-coach/internal/tui"
)

var modelFlag string

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Start interactive chat TUI",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		if cfg.NvidiaAPIKey == "" && cfg.APIKey == "" {
			return fmt.Errorf("no API key set - run 'paritok-cli auth-nvidia <key>' or set NVIDIA_API_KEY")
		}


		cl := client.New(cfg)
		modelName := modelFlag
		if modelName == "" {
			modelName = "openai/gpt-oss-20b"
		}

		p := tea.NewProgram(tui.New(cl, cfg.APIKey, modelName), tea.WithAltScreen(), tea.WithMouseCellMotion())
		if _, err := p.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		return nil
	},
}

func init() {
	chatCmd.Flags().StringVarP(&modelFlag, "model", "m", "", "model name to use (default: openai/gpt-oss-20b)")
	rootCmd.AddCommand(chatCmd)
}
