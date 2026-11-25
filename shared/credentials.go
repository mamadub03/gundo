package shared

import (
	"bufio"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/zalando/go-keyring"
	"golang.org/x/term"
)

const credentialService = "gundo"
const credentialRepoAccount = "github_repo"
const credentialTokenAccount = "github_token"
const legacyCredentialAccount = "github:mamadub03/gundo-test-repo"

// Initialized before starting either mode's goroutines; read-only afterward.
// The token stays in process memory, never in the environment or client state.
var githubToken string
var githubRepo string

func currentGitHubToken() (string, error) {
	if githubToken == "" {
		return "", fmt.Errorf("GitHub authentication is not initialized; set GITHUB_TOKEN or run gundo setup")
	}
	return githubToken, nil
}

func currentGitHubRepo() (string, error) {
	if githubRepo == "" {
		return "", fmt.Errorf("GitHub repository is not configured; set GITHUB_REPO or run gundo setup")
	}
	return githubRepo, nil
}

// CurrentGitHubToken returns the in-memory GitHub token.
func CurrentGitHubToken() string {
	return githubToken
}

// CurrentGitHubRepo returns the in-memory GitHub repository (owner/repo).
func CurrentGitHubRepo() string {
	return githubRepo
}

// SetGitHubCredentials sets the in-memory GitHub token and repository directly.
func SetGitHubCredentials(token, repo string) {
	githubToken = token
	githubRepo = repo
}

// ParseGitHubRepo normalizes a repository string or GitHub URL to 'owner/repo'.
func ParseGitHubRepo(input string) (string, error) {
	return parseGitHubRepo(input)
}

func parseGitHubRepo(input string) (string, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "", fmt.Errorf("repository cannot be empty")
	}

	// Handle git SSH syntax: git@github.com:owner/repo.git
	if strings.HasPrefix(trimmed, "git@github.com:") {
		trimmed = strings.TrimPrefix(trimmed, "git@github.com:")
	} else if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
		u, err := url.Parse(trimmed)
		if err != nil {
			return "", fmt.Errorf("invalid repository URL: %w", err)
		}
		trimmed = u.Path
	}

	trimmed = strings.TrimSuffix(trimmed, ".git")
	trimmed = strings.Trim(trimmed, "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", fmt.Errorf("invalid repository format %q; expected 'owner/repo' or GitHub repository URL", input)
	}

	return fmt.Sprintf("%s/%s", parts[0], parts[1]), nil
}

func loadStoredRepo() (string, error) {
	return keyring.Get(credentialService, credentialRepoAccount)
}

func saveStoredRepo(repo string) error {
	return keyring.Set(credentialService, credentialRepoAccount, repo)
}

func loadStoredToken() (string, error) {
	token, err := keyring.Get(credentialService, credentialTokenAccount)
	if err == nil && strings.TrimSpace(token) != "" {
		return token, nil
	}
	// Fallback to legacy account key if present
	legacy, lErr := keyring.Get(credentialService, legacyCredentialAccount)
	if lErr == nil && strings.TrimSpace(legacy) != "" {
		return legacy, nil
	}
	return token, err
}

func saveStoredToken(token string) error {
	return keyring.Set(credentialService, credentialTokenAccount, token)
}

func promptForRepo() (string, error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return "", fmt.Errorf("interactive terminal required to configure repository; set GITHUB_REPO or run gundo setup")
	}
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("GitHub repository (owner/repo or URL): ")
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("could not read repository input: %w", err)
	}
	return parseGitHubRepo(input)
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
	token := strings.TrimSpace(string(input))
	if token == "" {
		return "", fmt.Errorf("token cannot be empty")
	}
	if strings.ContainsAny(token, "\r\n\t ") {
		return "", fmt.Errorf("token must not contain whitespace")
	}
	return token, nil
}

func validateEnteredCredentials(token, repo string) error {
	client := &http.Client{
		Timeout: 10 * time.Second,
		// Setup must validate this destination, not follow a repository move or login redirect.
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	return validateGitHubToken(client, token, repo)
}

func getEnvCredential(keys ...string) string {
	for _, k := range keys {
		if val := strings.TrimSpace(os.Getenv(k)); val != "" {
			return val
		}
	}
	return ""
}

func InitializeGitHubToken() error {
	var repo string
	var token string
	saveRepo := false
	saveToken := false

	// 1. Resolve repository: ENV -> Keyring -> Prompt
	envRepo := getEnvCredential("GUNDO_REPO", "GITHUB_REPO")
	if envRepo != "" {
		parsed, err := parseGitHubRepo(envRepo)
		if err != nil {
			return fmt.Errorf("invalid repository from environment: %w", err)
		}
		repo = parsed
	} else {
		storedRepo, err := loadStoredRepo()
		if err == nil && strings.TrimSpace(storedRepo) != "" {
			parsed, pErr := parseGitHubRepo(storedRepo)
			if pErr == nil {
				repo = parsed
			}
		}
		if repo == "" {
			prompted, err := promptForRepo()
			if err != nil {
				return err
			}
			repo = prompted
			saveRepo = true
		}
	}

	// 2. Resolve token: ENV -> Keyring -> Prompt
	envToken := getEnvCredential("GUNDO_TOKEN", "GITHUB_TOKEN")
	if envToken != "" {
		token = envToken
	} else {
		storedToken, err := loadStoredToken()
		if err == nil && strings.TrimSpace(storedToken) != "" {
			token = strings.TrimSpace(storedToken)
		} else if err != nil && !errors.Is(err, keyring.ErrNotFound) {
			return fmt.Errorf("cannot access the OS credential store; unlock it or check access for the current user")
		}
		if token == "" {
			prompted, err := promptForToken()
			if err != nil {
				return err
			}
			token = prompted
			saveToken = true
		}
	}

	// 3. If either credential was freshly prompted, validate with GitHub before saving to keyring
	if saveRepo || saveToken {
		if err := validateEnteredCredentials(token, repo); err != nil {
			return err
		}
		if saveRepo {
			if err := saveStoredRepo(repo); err != nil {
				return fmt.Errorf("could not save repository in OS credential store: %w", err)
			}
		}
		if saveToken {
			if err := saveStoredToken(token); err != nil {
				return fmt.Errorf("could not save token in OS credential store: %w", err)
			}
		}
	}

	githubToken = token
	githubRepo = repo
	return nil
}

func SetupGitHubToken() error {
	repo, err := promptForRepo()
	if err != nil {
		return err
	}
	token, err := promptForToken()
	if err != nil {
		return err
	}

	if err := validateEnteredCredentials(token, repo); err != nil {
		return err
	}

	if err := saveStoredRepo(repo); err != nil {
		return fmt.Errorf("could not save repository in OS credential store: %w", err)
	}
	if err := saveStoredToken(token); err != nil {
		return fmt.Errorf("could not save token in OS credential store: %w", err)
	}

	githubToken = token
	githubRepo = repo
	fmt.Printf("Configuration saved for %s. Authentication and Issues access verified.\n", repo)
	return nil
}
