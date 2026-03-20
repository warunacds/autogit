package provider

import (
	"fmt"
	"net/url"
	"os"

	"github.com/warunacds/autogit/internal/config"
	"github.com/warunacds/autogit/internal/provider/anthropic"
	"github.com/warunacds/autogit/internal/provider/openai"
)

type Provider interface {
	GenerateMessage(diff string) (string, error)
}

func New(cfg *config.Config) (Provider, error) {
	switch cfg.Provider {
	case "claude":
		apiKey := os.Getenv("ANTHROPIC_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("ANTHROPIC_API_KEY is not set.\n  Export it with: export ANTHROPIC_API_KEY=your-key-here")
		}
		return anthropic.New(apiKey, cfg.Claude.Model), nil

	case "openai":
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" && !isLocalURL(cfg.OpenAI.BaseURL) {
			return nil, fmt.Errorf("OPENAI_API_KEY is not set.\n  Export it with: export OPENAI_API_KEY=your-key-here")
		}
		return openai.New(apiKey, cfg.OpenAI.BaseURL, cfg.OpenAI.Model), nil

	default:
		return nil, fmt.Errorf("unknown provider %q, supported: claude, openai", cfg.Provider)
	}
}

func isLocalURL(baseURL string) bool {
	u, err := url.Parse(baseURL)
	if err != nil {
		return false
	}
	host := u.Hostname()
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}
