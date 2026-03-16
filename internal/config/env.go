package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Env struct {
	ListenAddr       string
	WebhookSecret    string
	AppID            int64
	PrivateKeyPEM    string
	GitHubAPIBaseURL string
}

func LoadEnv() (Env, error) {
	listenAddr := strings.TrimSpace(os.Getenv("LISTEN_ADDR"))
	if listenAddr == "" {
		listenAddr = ":8080"
	}

	appIDRaw := strings.TrimSpace(os.Getenv("APP_ID"))
	if appIDRaw == "" {
		return Env{}, errors.New("APP_ID is required")
	}
	appID, err := strconv.ParseInt(appIDRaw, 10, 64)
	if err != nil {
		return Env{}, fmt.Errorf("parse APP_ID: %w", err)
	}

	privateKeyPEM := normalizePrivateKeyPEM(os.Getenv("PRIVATE_KEY"))
	if privateKeyPEM == "" {
		return Env{}, errors.New("PRIVATE_KEY is required")
	}

	webhookSecret := strings.TrimSpace(os.Getenv("WEBHOOK_SECRET"))
	if webhookSecret == "" {
		return Env{}, errors.New("WEBHOOK_SECRET is required")
	}

	apiBaseURL := strings.TrimSpace(os.Getenv("GITHUB_API_BASE_URL"))
	if apiBaseURL == "" {
		apiBaseURL = "https://api.github.com/"
	}

	return Env{
		ListenAddr:       listenAddr,
		WebhookSecret:    webhookSecret,
		AppID:            appID,
		PrivateKeyPEM:    privateKeyPEM,
		GitHubAPIBaseURL: apiBaseURL,
	}, nil
}

func normalizePrivateKeyPEM(value string) string {
	normalized := strings.TrimSpace(value)
	if len(normalized) >= 2 {
		if normalized[0] == '"' && normalized[len(normalized)-1] == '"' {
			if unquoted, err := strconv.Unquote(normalized); err == nil {
				normalized = unquoted
			} else {
				normalized = normalized[1 : len(normalized)-1]
			}
		} else if normalized[0] == '\'' && normalized[len(normalized)-1] == '\'' {
			normalized = normalized[1 : len(normalized)-1]
		}
	}
	replacer := strings.NewReplacer(`\r\n`, "\n", `\n`, "\n", `\r`, "\n")
	normalized = replacer.Replace(normalized)
	normalized = strings.ReplaceAll(normalized, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	return strings.TrimSpace(normalized)
}
