package initialize

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/warunacds/autogit/internal/config"
)

// Run interactively prompts the user to configure autogit, writing the result
// to ~/.autogit.yaml. If a config file already exists the user is asked whether
// to overwrite it before proceeding.
func Run() error {
	reader := bufio.NewReader(os.Stdin)

	// Check if config already exists
	if config.ConfigExists() {
		cfg, err := config.Load()
		if err == nil {
			fmt.Printf("Config already exists at %s:\n", config.Path())
			fmt.Printf("  provider: %s\n", cfg.Provider)
			switch cfg.Provider {
			case "claude":
				fmt.Printf("  model: %s\n", cfg.Claude.Model)
			case "claudecode":
				if cfg.ClaudeCode.Model != "" {
					fmt.Printf("  model: %s\n", cfg.ClaudeCode.Model)
				} else {
					fmt.Printf("  model: (CLI default)\n")
				}
			default:
				fmt.Printf("  base_url: %s\n", cfg.OpenAI.BaseURL)
				fmt.Printf("  model: %s\n", cfg.OpenAI.Model)
			}
			fmt.Print("\nOverwrite? [y/N] ")
			line, _ := reader.ReadString('\n')
			if strings.TrimSpace(strings.ToLower(line)) != "y" {
				fmt.Println("Aborted.")
				return nil
			}
		}
	}

	// Select provider
	fmt.Println("\nSelect a provider:")
	fmt.Println("  1) Claude (Anthropic API key)")
	fmt.Println("  2) Claude Code (Pro/Max subscription — no API key needed)")
	fmt.Println("  3) OpenAI-compatible (ChatGPT, Ollama, LM Studio, Gemini, etc.)")
	fmt.Print("> ")
	line, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	choice := strings.TrimSpace(line)
	cfg := config.DefaultConfig()

	switch choice {
	case "1":
		cfg.Provider = "claude"

		fmt.Printf("\nModel name [%s]: ", cfg.Claude.Model)
		line, _ = reader.ReadString('\n')
		if model := strings.TrimSpace(line); model != "" {
			cfg.Claude.Model = model
		}

		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("\nConfig saved to %s\n", config.Path())
		fmt.Println("\nMake sure ANTHROPIC_API_KEY is set in your environment:")
		fmt.Println("  export ANTHROPIC_API_KEY=your-key-here")

	case "2":
		cfg.Provider = "claudecode"

		fmt.Print("\nModel name (leave blank for CLI default): ")
		line, _ = reader.ReadString('\n')
		if model := strings.TrimSpace(line); model != "" {
			cfg.ClaudeCode.Model = model
		}

		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("\nConfig saved to %s\n", config.Path())
		fmt.Println("\nMake sure the claude CLI is installed and you're logged in:")
		fmt.Println("  https://docs.anthropic.com/en/docs/claude-code")

	case "3":
		cfg.Provider = "openai"

		fmt.Printf("\nBase URL [%s]: ", cfg.OpenAI.BaseURL)
		line, _ = reader.ReadString('\n')
		if baseURL := strings.TrimSpace(line); baseURL != "" {
			cfg.OpenAI.BaseURL = strings.TrimRight(baseURL, "/")
		}

		fmt.Printf("\nModel name [%s]: ", cfg.OpenAI.Model)
		line, _ = reader.ReadString('\n')
		if model := strings.TrimSpace(line); model != "" {
			cfg.OpenAI.Model = model
		}

		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("\nConfig saved to %s\n", config.Path())
		fmt.Println("\nMake sure OPENAI_API_KEY is set in your environment:")
		fmt.Println("  export OPENAI_API_KEY=your-key-here")
		fmt.Println("(For local models like Ollama, you can skip this or set it to any value.)")

	default:
		return fmt.Errorf("invalid choice %q, please enter 1, 2, or 3", choice)
	}

	return nil
}
