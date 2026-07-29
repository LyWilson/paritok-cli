package cmd

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/LyWilson/paritok-cli/internal/config"
)

var proxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "Manage the paritok proxy",
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the paritok proxy",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		if cfg.NvidiaAPIKey == "" {
			return fmt.Errorf("NVIDIA API key not set — run 'paritok-cli auth-nvidia <key>'")
		}

		paritokPath, err := findParitok()
		if err != nil {
			return err
		}

		workDir := getWorkDir()

		proc := exec.Command(paritokPath,
			"proxy", "--port", "8080",
			"--config-file", filepath.Join(workDir, "paritok.yaml"),
			"--openai-url", "https://integrate.api.nvidia.com",
		)
		proc.Dir = workDir
		proc.Env = append(os.Environ(),
			"PARITOK_API_KEY="+cfg.APIKey,
			"NVIDIA_API_KEY="+cfg.NvidiaAPIKey,
		)

		proc.Stdout = nil
		proc.Stderr = nil

		if err := proc.Start(); err != nil {
			return fmt.Errorf("start proxy: %w", err)
		}

		for i := 0; i < 30; i++ {
			time.Sleep(500 * time.Millisecond)
			if err := checkHealth(); err == nil {
				fmt.Fprintf(os.Stderr, "Proxy running on http://127.0.0.1:8080 (pid %d)\n", proc.Process.Pid)
				return nil
			}
		}
		return fmt.Errorf("proxy started but health check timed out")
	},
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the paritok proxy",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := exec.Command("taskkill", "/IM", "paritok.exe", "/F").Run(); err != nil {
			return fmt.Errorf("stop proxy: %w", err)
		}
		fmt.Fprintln(os.Stderr, "Proxy stopped")
		return nil
	},
}

func findParitok() (string, error) {
	if p, err := exec.LookPath("paritok"); err == nil {
		return p, nil
	}
	dirs := []string{
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Python", "pythoncore-3.14-64", "Scripts"),
		filepath.Join(os.Getenv("APPDATA"), "Python", "Scripts"),
	}
	for _, d := range dirs {
		p := filepath.Join(d, "paritok.exe")
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("paritok not found on PATH or in common Python Scripts directories")
}

func getWorkDir() string {
	if d := os.Getenv("PARITOK_CLI_DIR"); d != "" {
		return d
	}
	exe, _ := os.Executable()
	return filepath.Dir(exe)
}

func checkHealth() error {
	conn, err := net.DialTimeout("tcp", "127.0.0.1:8080", 2*time.Second)
	if err != nil {
		return err
	}
	conn.Close()
	return nil
}

func init() {
	proxyCmd.AddCommand(startCmd)
	proxyCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(proxyCmd)
}
