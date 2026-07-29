package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/LyWilson/prompt-diet-coach/internal/agent"
	"github.com/LyWilson/prompt-diet-coach/internal/client"
	"github.com/LyWilson/prompt-diet-coach/internal/config"
)

var codeFlags struct {
	iterations int
}

var codeCmd = &cobra.Command{
	Use:   "code <prompt>",
	Short: "Run a coding agent that generates and iterates on code",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		prompt := strings.Join(args, " ")
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		modelName := modelFlag
		if modelName == "" {
			modelName = "openai/gpt-oss-20b"
		}
		cl := client.New(cfg)

		fmt.Fprintf(os.Stderr, "Code agent: %s\n", prompt)

		messages := []client.Message{
			{Role: "user", Content: prompt + "\n\nReturn ONLY code in a single ``` block."},
		}

		for i := 0; i < codeFlags.iterations; i++ {
			fmt.Fprintf(os.Stderr, "\n--- Iteration %d/%d ---\n", i+1, codeFlags.iterations)

			resp, err := streamAndAccumulate(cl, modelName, messages)
			if err != nil {
				return fmt.Errorf("iteration %d: %w", i, err)
			}
			messages = append(messages, client.Message{Role: "assistant", Content: resp})

			code := agent.ExtractCodeBlock(resp)
			if code == "" {
				fmt.Fprintf(os.Stderr, "No code block found, stopping.\n")
				fmt.Println(resp)
				return nil
			}

			fmt.Fprintf(os.Stderr, "\n--- Generated code ---\n")
			fmt.Println(code)

			if i == codeFlags.iterations-1 {
				fmt.Fprintf(os.Stderr, "\n--- Final iteration ---\n")
				return nil
			}

			lang := agent.DetectLang(resp)
			fmt.Fprintf(os.Stderr, "\n--- Running (%s) ---\n", lang)
			output, err := agent.RunCode(code, lang)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %s\n", err)
				messages = append(messages, client.Message{Role: "user", Content: "Error:\n" + err.Error() + "\nFix and return ONLY the corrected code."})
			} else {
				fmt.Fprintf(os.Stderr, "Output:\n%s\n", output)
				if output == "" {
					output = "(no output)"
				}
				messages = append(messages, client.Message{Role: "user", Content: "Output:\n" + output + "\nIf complete, return the FINAL code. Otherwise improve it."})
			}
		}
		return nil
	},
}

func streamAndAccumulate(cl *client.Client, model string, messages []client.Message) (string, error) {
	ctx := context.Background()
	events, err := cl.StreamChat(ctx, model, messages, nil)
	if err != nil {
		return "", err
	}
	var resp strings.Builder
	for evt := range events {
		switch evt.Type {
		case client.EventText:
			fmt.Print(evt.Content)
			resp.WriteString(evt.Content)
		case client.EventDone:
			fmt.Println()
			return resp.String(), nil
		case client.EventError:
			return "", evt.Error
		}
	}
	return resp.String(), nil
}

func init() {
	codeCmd.Flags().IntVarP(&codeFlags.iterations, "iterations", "n", 5, "max code iterations")
	codeCmd.Flags().StringVarP(&modelFlag, "model", "m", "", "model name")
	rootCmd.AddCommand(codeCmd)
}
