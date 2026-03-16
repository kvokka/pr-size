package main

import (
	"log"
	"net/http"
	"os"

	"pr-size-labeler/internal/auth"
	"pr-size-labeler/internal/config"
	"pr-size-labeler/internal/githubapi"
	"pr-size-labeler/internal/webhook"
)

func main() {
	env, err := config.LoadEnv()
	if err != nil {
		log.Fatal(err)
	}

	tokenProvider, err := auth.NewAppTokenProvider(env.AppID, env.PrivateKeyPEM, env.GitHubAPIBaseURL)
	if err != nil {
		log.Fatal(err)
	}

	handler := webhook.NewHandler(
		env.WebhookSecret,
		tokenProvider,
		func(token string) *githubapi.Client {
			return githubapi.NewClient(env.GitHubAPIBaseURL, token, http.DefaultClient)
		},
	)

	server := &http.Server{
		Addr:    env.ListenAddr,
		Handler: handler,
	}

	log.Printf("pr-size-labeler listening on %s", env.ListenAddr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}

	_ = os.Stdout.Sync()
	_ = os.Stderr.Sync()
}
