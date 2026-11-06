package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/zalando/go-keyring"
	"golang.org/x/term"
)

const credentialService = "gundo"
const credentialAccount = "github:mamadub03/gundo-test-repo"

// Initialized before starting either mode's goroutines; read-only afterward.
// The token stays in process memory, never in the environment or client state.
var githubToken string

func currentGitHubToken() (string, error) {
	if githubToken == "" {
		return "", fmt.Errorf("GitHub authentication is not initialized; run gundo setup")
	}
	return githubToken, nil
}

func loadStoredToken() (string, error) {
	return keyring.Get(credentialService, credentialAccount)
}

func saveStoredToken(token string) error {
	return keyring.Set(credentialService, credentialAccount, token)
}

func promptForToken() (string, error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return "", fmt.Errorf("token setup requires an interactive terminal; run gundo setup as the user who will run Gundo")
	}
	fmt.Print("GitHub token (input hidden): ")
	input, err := term.ReadPassword(fd)
	fmt.Println()
	if err != nil {
		return "", fmt.Errorf("could not read token; setup cancelled")
	}
	return string(input), nil
}

func validateEnteredToken(token string) error {
	client := &http.Client{
		Timeout: 10 * time.Second,
		// Setup must validate this destination, not follow a repository move or login redirect.
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	return validateGitHubToken(client, token)
}

func initializeGitHubToken() error {
	token, err := resolveGitHubToken(loadStoredToken, promptForToken, validateEnteredToken, saveStoredToken)
	if err != nil {
		return err
	}
	githubToken = token
	return nil
}

func setupGitHubToken() error {
	_, err := enterAndSaveToken(promptForToken, validateEnteredToken, saveStoredToken)
	if err == nil {
		fmt.Println("Token saved. Authentication and Issues read access verified; write permission was not tested. Restart running Gundo processes to use it.")
	}
	return err
}

// Function arguments keep tests away from real credentials, terminals, and GitHub.
func resolveGitHubToken(load func() (string, error), prompt func() (string, error), validate func(string) error, save func(string) error) (string, error) {
	token, err := load()
	if err == nil {
		if strings.TrimSpace(token) == "" {
			return "", fmt.Errorf("saved credential is empty; run gundo setup to replace it")
		}
		return token, nil
	}
	if !errors.Is(err, keyring.ErrNotFound) {
		// Never treat denied/locked storage as missing, or include backend secret output.
		return "", fmt.Errorf("cannot access the OS credential store; unlock it or check access for the current user")
	}
	return enterAndSaveToken(prompt, validate, save)
}

func enterAndSaveToken(prompt func() (string, error), validate func(string) error, save func(string) error) (string, error) {
	token, err := prompt()
	if err != nil {
		return "", err
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return "", fmt.Errorf("token cannot be empty")
	}
	if strings.ContainsAny(token, "\r\n\t ") {
		return "", fmt.Errorf("token must not contain whitespace")
	}
	// Validation does not post anything. A rejected replacement never reaches storage.
	if err := validate(token); err != nil {
		return "", err
	}
	if err := save(token); err != nil {
		return "", fmt.Errorf("could not save token in the OS credential store; check store access and retry setup")
	}
	return token, nil
}
